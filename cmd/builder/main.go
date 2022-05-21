package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"eval/pkg/db"
	"eval/pkg/grpc/server"

	pbasync "eval/proto/async_service"
	pb "eval/proto/builder"

	"github.com/google/uuid"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/anypb"
	"gorm.io/gorm"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	port = ":50051"
)

type BuildInfo struct {
	// we should split this first part as a generic piece supporting Operaations
	gorm.Model
	BuildID uuid.UUID `gorm:"type:uuid;"`

	// and this part which is the application specific data
	Branch    string
	CommitSHA string
	Targets   string
	ImageName string
	ImageTag  string
}

type serverContext struct {
	log       *zerolog.Logger
	v         *viper.Viper
	clientSet *kubernetes.Clientset
	asynq     *Asynq
	db        *gorm.DB
}

func (s *serverContext) Build(ctx context.Context, in *pb.BuildRequest) (*pbasync.Operation, error) {
	s.log.Info().Msg("Builder summoned")

	r := asynq.RedisClientOpt{Addr: "redis.eval.svc.cluster.local:6379"}
	c := asynq.NewClient(r)

	log.Printf("Enqueuing task")
	branch := in.Branch
	commit := in.CommitSHA
	target := in.Target

	imageTag, err := uuid.NewRandom()
	if err != nil {
		log.Printf("Cannot generate UUID: %v", err)
	}
	buildID, err := uuid.NewRandom()
	if err != nil {
		log.Printf("Cannot generate UUID: %v", err)
	}

	// // caching should really be in the caching service when we have it
	// // and the builder sshould  just build
	// var builds []BuildInfo
	// s.db.Where("commit_sha = ?", commit).Find(&builds)
	// for _, build := range builds {
	// 	log.Printf("*** BUILD %v", build.ImageTag)
	// 	// we will be able to count on the field tobe there
	// 	// here we should check tags are compatible and if there're multiple matches
	// 	// keep he image that has more tags as it is the most useful to keep around.
	// 	if false && len(build.ImageTag) > 0 {
	// 		s.log.Info().Msg("Returning available image")
	// 		return &pb.BuildResponse{Response: "something built", ImageName: "eval", ImageTag: build.ImageTag}, nil
	// 	}
	// }

	s.log.Info().Msg("Building image")
	targetJSON, _ := json.Marshal(target)
	s.db.Create(&BuildInfo{
		BuildID:   buildID,
		Branch:    branch,
		CommitSHA: commit,
		Targets:   string(targetJSON),
		ImageName: "eval",
		ImageTag:  imageTag.String(),
	})
	s.db.Commit()

	t := NewBuildTask(branch, commit, target, imageTag.String())
	taskInfo, err := c.Enqueue(t)
	if err != nil {
		log.Fatal("could not enqueue task: %v", err)
	}
	log.Printf("INFO: %v", taskInfo)

	count = 0

	// imageName should be passed to newBuildTask
	return &pbasync.Operation{
		Name: buildID.String(),
	}, nil

}

var count = 0

func (s *serverContext) GetOperation(ctx context.Context, in *pbasync.GetOperationRequest) (*pbasync.Operation, error) {
	var response *anypb.Any

	count++

	done := count > 4

	s.log.Info().Msg("Builder GetOperation")

	response, err := anypb.New(&pb.BuildResponse{})
	if err != nil {
		panic(err)
	}

	return &pbasync.Operation{
		Name:   in.Name,
		Done:   done,
		Result: &pbasync.Operation_Response{response},
	}, nil
}

func connectToK8s() *kubernetes.Clientset {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		log.Printf("token not found")
	} else {
		content, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
		if err != nil {
			log.Fatal(err)
		}
		// Convert []byte to string and print to screen
		text := string(content)
		fmt.Println(text)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Println("No cluster config: %v", err)
	} else {
		log.Println(config)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	return clientset
}

// func NewDB(service string, schema ...interface{}) (*gorm.DB, error) {
// 	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("/data/sqlite/%s.db", service)), &gorm.Config{})
// 	if err != nil {
// 		panic("failed to connect database")
// 	}

// 	// Migrate the schema
// 	err = db.AutoMigrate(schema...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return db, nil
// }

func serviceRegister(server server.Server, asynq *Asynq) func(*grpc.Server) {
	return func(s *grpc.Server) {
		context := serverContext{}
		context.log = server.Logger()
		context.v = server.Config()
		context.clientSet = connectToK8s()
		context.asynq = asynq
		context.db, _ = db.NewDB("builder", &BuildInfo{})
		context.log.Info().Msg("Registering service")
		pb.RegisterBuilderServiceServer(s, &context)
		pbasync.RegisterOperationsServer(s, &context)
		reflection.Register(s)
	}
}

type BuildPayload struct {
	Branch    string
	CommitSHA string
	Target    []string
	ImageTag  string
}

//  return errorin addition to task
func NewBuildTask(branch string, commitSHA string, target []string, imageTag string) *asynq.Task {
	payload, err := json.Marshal(BuildPayload{Branch: branch, CommitSHA: commitSHA, Target: target, ImageTag: imageTag})
	if err != nil {
		return nil
	}
	return asynq.NewTask("image:build", payload, asynq.Queue("critical"))
}

func buildJobSpec(branch string, commitSHA string, targets []string, imageTag string) *batchv1.Job {
	var backOffLimit int32 = 0
	var ttlSecondsAfterFinished int32 = 60

	gitContext := "git://gitea-service.gitea-repo.svc.cluster.local:3000/mav/eval.git#refs/heads/" + branch + "#" + commitSHA
	//gitContext := "git://gitea-service.gitea-repo.svc.cluster.local:3000/mav/eval.git"

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko-",
			Namespace:    "default",
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					HostAliases: []v1.HostAlias{
						{
							IP:        "192.168.1.8",
							Hostnames: []string{"registry.other.net"},
						},
					},
					Containers: []v1.Container{
						{
							//we should use:
							// -git
							// Branch to clone if build context is a git repository
							// (default branch=,single-branch=false,recurse-submodules=false)
							Name:  "kaniko",
							Image: "gcr.io/kaniko-project/executor:debug",
							Args: []string{"--insecure",
								"--insecure-pull",
								"--skip-tls-verify",
								"--build-arg", "TARGETS=" + strings.Join(targets, " "),
								"--destination=registry.other.net:5000/eval:" + imageTag,
								"--context", gitContext,
								"--digest-file=/dev/termination-log",
								//								"--log-format=json",
								"--log-format=color",
								"--log-timestamp=true",
								"--use-new-run",
								//								"--verbosity=debug",
								"--dockerfile=dockerfile"},
							Env: []v1.EnvVar{
								{
									Name:  "GIT_TOKEN",
									Value: "969c5cb1eaee59d878648cb862bef551cac70d34",
								},
								{
									Name:  "GIT_PULL_METHOD",
									Value: "http",
								},
							},
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
				},
			},
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
		},
	}
	return jobSpec
}

func build(branch string, commitSHA string, targets []string, imageTag string) {
	clientset := connectToK8s()
	batchAPI := clientset.BatchV1()
	jobs := batchAPI.Jobs("default")

	playWithDocker()

	job, err := jobs.Create(context.TODO(), buildJobSpec(branch, commitSHA, targets, imageTag), metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("Failed to create K8s job. %v", err)
	}
	log.Printf("Kaniko job created: %T %v", job, job)

	timeout := int64(3600)
	watchres, err := jobs.Watch(context.Background(), metav1.ListOptions{
		TimeoutSeconds: &timeout,
	})

	if err != nil {
		log.Fatalln("Failed to watch job", err)
	}
	defer watchres.Stop()

	eventres := watchres.ResultChan()
	for we := range eventres {
		log.Println("----")
		log.Printf("TYPE: %T (%v) %v\n", we.Type, we.Type, we.Type == "DELETED")
		if we.Type == "DELETED" {
			break
		}
		job, ok := we.Object.(*batchv1.Job)
		if !ok {
			continue
		}
		log.Println("-")
		log.Println(job.Status.Conditions)
		log.Println(job.Status.Active > 0)
		log.Println(job.Status.Succeeded > 0)
	}

	// We should get to the pod for the job, which doesn't seem
	// directly doable a way is to assign a label to the pod, say the
	// build ID and then search for that. This would allow ud to get
	// to the status message, which is where we ask kaniko to put the
	// image SHA

	//	 create a pod with label, and then I get it through LabelSelector. Like it :
	// config, err := clientcmd.BuildConfigFromFlags("", "~/.kube/config")
	// if err != nil {
	//     println("config build error")
	// }

	// client, err := kubernetes.NewForConfig(config)

	// pods, err := client.CoreV1().Pods("test").List(context.TODO(),
	//     v1.ListOptions{LabelSelector: "name=label_name"})

	// for _, v := range pods.Items {
	//     log := client.CoreV1().Pods("test").GetLogs(v.Name, &v12.PodLogOptions{})
	// }

}

func playWithDocker() {
	//	url := "http://registry.other.net:5000/"
	url := "http://192.168.1.8:5000/"
	username := "" // anonymous
	password := "" // anonymous
	hub, err := registry.NewInsecure(url, username, password)
	if err != nil {
		log.Printf("Cannot connect to docker registry")
	}

	repositories, err := hub.Repositories()
	if err != nil {
		log.Printf("Cannot get list of repositories")
	}
	for r := range repositories {
		log.Printf("Repo: %T %v\n", r, r)
		// tags, err := hub.Tags(r)
		// if err != nil {
		// 	log.Printf("Cannot get list of tags")
		// }
		// for t := range tags {
		// 	log.Printf("Tag: %v\n", t)
		// }

	}

}

func HandleBuildTask(ctx context.Context, t *asynq.Task) error {
	var p BuildPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	log.Printf("Building targets=%s @ branch=%s, commitSHA=%s -> imageTag: %s", p.Target, p.Branch, p.CommitSHA, p.ImageTag)
	build(p.Branch, p.CommitSHA, p.Target, p.ImageTag)
	log.Printf("Done building branch=%s, commitSHA=%s", p.Branch, p.CommitSHA)
	return nil
}

type Asynq struct {
	server    *asynq.Server
	inspector *asynq.Inspector
}

func NewAsynq() *Asynq {
	r := asynq.RedisClientOpt{Addr: "redis.eval.svc.cluster.local:6379"}
	server := asynq.NewServer(r, asynq.Config{
		// Specify how many concurrent workers to use. Up to 4 is ok, but we
		// stay lower to allow for other things to happen during a demo.
		Concurrency: 2,
		// Optionally specify multiple queues with different priority.
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
		// See the godoc for other configuration options
	})

	inspector := asynq.NewInspector(r)

	return &Asynq{
		server:    server,
		inspector: inspector,
	}
}

func main() {
	asynqClient := NewAsynq()

	go func() {
		mux := asynq.NewServeMux()
		mux.HandleFunc("image:build", HandleBuildTask)

		if err := asynqClient.server.Run(mux); err != nil {
			log.Fatalf("could not run server: %v", err)
		}
	}()

	server := server.Build(port)
	server.RegisterService(serviceRegister(server, asynqClient))
	server.Start()
}

// func _main() {
// 	log.Print("Hello there, this is the builder")

// 	//-----------------------------------------------------------------------------
// 	// TODO add an host entry for registry.other.net
// 	log.Println("Trying docker")
// 	// dockerClient, err := docker.NewClient("http://192.168.1.8:5000")
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }

// 	// imgs, err := dockerClient.ListImages(docker.ListImagesOptions{All: false})
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	// for _, img := range imgs {
// 	// 	fmt.Println("ID: ", img.ID)
// 	// 	fmt.Println("RepoTags: ", img.RepoTags)
// 	// 	fmt.Println("Created: ", img.Created)
// 	// 	fmt.Println("Size: ", img.Size)
// 	// 	fmt.Println("VirtualSize: ", img.VirtualSize)
// 	// 	fmt.Println("ParentId: ", img.ParentID)
// 	// }

// 	// url := "https://registry-1.docker.io/"
// 	// username := "" // anonymous
// 	// password := "" // anonymous
// 	// hub, err := registry.New(url, username, password)

// 	// repositories, err := hub.Repositories()
// 	// for r := range repositories {
// 	// 	log.Println(r)
// 	// }
// 	clientset := connectToK8s()

// 	//-----------------------------------------------------------------------------
// 	log.Println("Kubernetes")
// 	batchAPI := clientset.BatchV1()
// 	jobs := batchAPI.Jobs("default")

// 	var backOffLimit int32 = 0
// 	var ttlSecondsAfterFinished int32 = 10

// 	jobSpec := &batchv1.Job{
// 		ObjectMeta: metav1.ObjectMeta{
// 			GenerateName: "kaniko-",
// 			Namespace:    "default",
// 		},
// 		Spec: batchv1.JobSpec{
// 			Template: v1.PodTemplateSpec{
// 				Spec: v1.PodSpec{
// 					HostAliases: []v1.HostAlias{
// 						{
// 							IP:        "192.168.1.8",
// 							Hostnames: []string{"registry.other.net"},
// 						},
// 					},
// 					Containers: []v1.Container{
// 						{
// 							Name:  "kaniko",
// 							Image: "gcr.io/kaniko-project/executor:debug",
// 							Args: []string{"--insecure",
// 								"--insecure-pull",
// 								"--skip-tls-verify",
// 								"--destination=registry.other.net:5000/test:bar",
// 								"--context=git://gitea-service.gitea-repo.svc.cluster.local:3000/mav/eval.git",
// 								"--dockerfile=dockerfile"},
// 							Env: []v1.EnvVar{
// 								{
// 									Name:  "GIT_TOKEN",
// 									Value: "969c5cb1eaee59d878648cb862bef551cac70d34",
// 								},
// 								{
// 									Name:  "GIT_PULL_METHOD",
// 									Value: "http",
// 								},
// 							},
// 						},
// 					},
// 					RestartPolicy: v1.RestartPolicyNever,
// 				},
// 			},
// 			BackoffLimit:            &backOffLimit,
// 			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
// 		},
// 	}

// 	_, err := jobs.Create(context.TODO(), jobSpec, metav1.CreateOptions{})
// 	if err != nil {
// 		log.Fatalln("Failed to create K8s job. %v", err)
// 	}
// 	log.Println("Kaniko job created")

// 	timeout := int64(3600)
// 	watchres, err := jobs.Watch(context.Background(), metav1.ListOptions{
// 		TimeoutSeconds: &timeout,
// 	})

// 	if err != nil {
// 		log.Fatalln("Failed to watch job", err)
// 	}
// 	defer watchres.Stop()

// 	eventres := watchres.ResultChan()
// 	for we := range eventres {
// 		log.Println("----")
// 		log.Println(we.Type)
// 		job, ok := we.Object.(*batchv1.Job)
// 		if !ok {
// 			continue
// 		}
// 		log.Println(job.Status.Conditions)
// 	}

// 	//-----------------------------------------------------------------------------
// 	api := clientset.CoreV1()

// 	pods, err := api.Pods("").List(context.TODO(), metav1.ListOptions{})
// 	if err != nil {
// 		panic(err.Error())
// 	}
// 	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

// 	//-----------------------------------------------------------------------------
// 	var (
// 		host       = "redis.eval.svc.cluster.local"
// 		redis_port = "6379"
// 	)

// 	client := redis.NewClient(&redis.Options{
// 		Addr: host + ":" + redis_port,
// 		//		Password: password,
// 		DB: 0,
// 	})

// 	pong, err := client.Ping().Result()
// 	if err != nil {
// 		log.Println("---------------------")
// 		log.Fatal(err)
// 	}
// 	log.Println("---------------------")
// 	log.Println(pong)
// 	log.Println("---------------------")

// 	client.Incr("something.kcount")
// 	client.Incr("something.nkcount")

// 	val, err := client.Get("something.kcount").Result()
// 	if err != nil {
// 		panic(err)
// 	}

// 	result := string("key count: " + string(val))
// 	log.Println(result)

// 	cert, err := tls.LoadX509KeyPair(filepath.Join(baseDir, ServerCert),
// 		filepath.Join(baseDir, ServerKey))
// 	if err != nil {
// 		log.Fatalf("Failed to get certificate")
// 	}

// 	certPool := x509.NewCertPool()
// 	bs, err := ioutil.ReadFile(filepath.Join(baseDir, CaCert))
// 	if err != nil {
// 		log.Fatalf("failed to read certificates chain: %s", err)
// 	}
// 	ok := certPool.AppendCertsFromPEM(bs)
// 	if !ok {
// 		log.Fatalf("failed to append certs")
// 	}

// 	opts := []grpc.ServerOption{
// 		grpc.Creds(
// 			credentials.NewTLS(&tls.Config{
// 				ClientAuth:   tls.RequireAndVerifyClientCert,
// 				Certificates: []tls.Certificate{cert},
// 				ClientCAs:    certPool})),
// 	}

// 	s := grpc.NewServer(opts...)
// 	pb.RegisterEngineServiceServer(s, &server{})

// 	lis, err := net.Listen("tcp", port)
// 	if err != nil {
// 		log.Fatalf("failed to listen: %v", err)
// 	}
// 	if err := s.Serve(lis); err != nil {
// 		log.Fatalf("failed to serve: %v", err)
// 	}
// }

// func getPodObject() *core.Pod {
// 	return &core.Pod{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "my-test-pod",
// 			Namespace: "default",
// 			Labels: map[string]string{
// 				"app": "demo",
// 			},
// 		},
// 		Spec: core.PodSpec{
// 			Containers: []core.Container{
// 				{
// 					Name:            "busybox",
// 					Image:           "busybox",
// 					ImagePullPolicy: core.PullIfNotPresent,
// 					Command: []string{
// 						"sleep",
// 						"3600",
// 					},
// 				},
// 			},
// 		},
// 	}
// }

// func printPVCs(pvcs *v1.PersistentVolumeClaimList) {
// 	template := "%-32s%-8s%-8s\n"
// 	fmt.Printf(template, "NAME", "STATUS", "CAPACITY")
// 	for _, pvc := range pvcs.Items {
// 		quant := pvc.Spec.Resources.Requests[v1.ResourceStorage]
// 		fmt.Printf(
// 			template,
// 			pvc.Name,
// 			string(pvc.Status.Phase),
// 			quant.String())
// 	}
// }
