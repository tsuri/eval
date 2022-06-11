package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

var buildDone = make(map[string]bool)
var buildInfo = make(map[string]BuildInfo)

func (s *serverContext) Build(ctx context.Context, in *pb.BuildRequest) (*pbasync.Operation, error) {
	//	s.log.Info().Msg("Builder summoned")

	r := asynq.RedisClientOpt{Addr: "redis.eval.svc.cluster.local:6379"}
	c := asynq.NewClient(r)

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

	targetJSON, _ := json.Marshal(target)

	buildInfo[buildID.String()] = BuildInfo{
		BuildID:   buildID,
		Branch:    branch,
		CommitSHA: commit,
		Targets:   string(targetJSON),
		ImageName: "eval",
		ImageTag:  imageTag.String(),
	}

	s.db.Create(&BuildInfo{
		BuildID:   buildID,
		Branch:    branch,
		CommitSHA: commit,
		Targets:   string(targetJSON),
		ImageName: "eval",
		ImageTag:  imageTag.String(),
	})
	s.db.Commit()

	t := NewBuildTask(buildID.String(), branch, commit, target, imageTag.String())
	_, err = c.Enqueue(t)
	if err != nil {
		log.Fatal("could not enqueue task: %v", err)
	}

	// imageName should be passed to newBuildTask
	return &pbasync.Operation{
		Name: buildID.String(),
	}, nil

}

func getDigest(image string, tag string) (string, error) {
	url := "http://192.168.1.8:5000/"
	username := "" // anonymous
	password := "" // anonymous
	hub, err := registry.NewInsecure(url, username, password)
	if err != nil {
		return "", err
	}

	digest, err := hub.ManifestDigest(image, tag)
	if err != nil {
		return "", err
	}
	return digest.String(), nil
}

func (s *serverContext) GetOperation(ctx context.Context, in *pbasync.GetOperationRequest) (*pbasync.Operation, error) {
	if done, ok := buildDone[in.Name]; ok && done {
		// buildInfo := BuildInfo{}

		//		s.db.Debug().First(&buildInfo, "BuildID = ?", in.Name)
		//buildID, err := uuid.Parse(in.Name)
		// if err != nil {
		// 	s.log.Err(err).Msg("cannot parse uuid")
		// }

		// DB doesn't woork, stuff is inserted, but no dice
		// s.db.Debug().First(&buildInfo, "BuildID = ?", buildID)
		// s.log.Info().Msg("***********************************")
		// s.log.Info().Str("ID", in.Name).Str("Build Info", fmt.Sprintf("%v", buildInfo)).Msg("get operation")
		// s.log.Info().Msg("***********************************")

		s.log.Info().Str("ID", in.Name).Str("Build Info", fmt.Sprintf("%v", buildInfo[in.Name])).Msg("get operation")
		bi := buildInfo[in.Name]

		digest, err := getDigest(bi.ImageName, bi.ImageTag)
		if err != nil {
			s.log.Info().Msg(fmt.Sprintf("Cannot get digest: %v", err))
		}
		s.log.Info().Str("name", bi.ImageName).Str("tag", bi.ImageTag).Str("digest", digest).Msg("Image")
		response, err := anypb.New(&pb.BuildResponse{
			Response:    "will be nicer (but is real): " + in.Name,
			ImageName:   bi.ImageName,
			ImageTag:    bi.ImageTag,
			ImageDigest: digest,
		})
		if err != nil {
			panic(err)
		}

		return &pbasync.Operation{
			Name:   in.Name,
			Done:   true,
			Result: &pbasync.Operation_Response{response},
		}, nil
	} else {
		return &pbasync.Operation{
			Name: in.Name,
		}, nil
	}
}

func connectToK8s() *kubernetes.Clientset {
	// _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	// if err != nil {
	// 	log.Printf("token not found")
	// } else {
	// 	content, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	// Convert []byte to string and print to screen
	// 	text := string(content)
	// 	fmt.Println(text)
	// }

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
	BuildID   string
	Branch    string
	CommitSHA string
	Target    []string
	ImageTag  string
}

//  return errorin addition to task
func NewBuildTask(buildID string, branch string, commitSHA string, target []string, imageTag string) *asynq.Task {
	payload, err := json.Marshal(BuildPayload{BuildID: buildID, Branch: branch, CommitSHA: commitSHA, Target: target, ImageTag: imageTag})
	if err != nil {
		return nil
	}
	return asynq.NewTask("image:build", payload, asynq.Queue("critical"))
}

func buildJobSpec(buildID string, branch string, commitSHA string, targets []string, imageTag string) *batchv1.Job {
	var backOffLimit int32 = 0
	var ttlSecondsAfterFinished int32 = 10

	gitContext := "git://gitea-service.gitea-repo.svc.cluster.local:3000/mav/eval.git#refs/heads/" + branch + "#" + commitSHA

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko-",
			Namespace:    "default",
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"buildID": buildID,
					},
				},
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
								// MAV REGISTRY
								// "--destination=registry.other.net:5000/eval:" + imageTag,
								"--destination=kind-registry:5000/eval:" + imageTag,
								//								"--destination=registry.other.net:5000/eval:" + imageTag,
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

func build(buildID string, branch string, commitSHA string, targets []string, imageTag string) {
	clientset := connectToK8s()
	batchAPI := clientset.BatchV1()
	jobs := batchAPI.Jobs("default")

	_, err := jobs.Create(context.TODO(), buildJobSpec(buildID, branch, commitSHA, targets, imageTag), metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("Failed to create K8s job. %v", err)
	}
	//	log.Printf("Kaniko job created: %T %v", job, job)

	timeout := int64(600)
	watchres, err := jobs.Watch(context.Background(), metav1.ListOptions{
		TimeoutSeconds: &timeout,
	})

	if err != nil {
		log.Fatalln("Failed to watch job", err)
	}
	defer watchres.Stop()

	// cl, err := k8s.NewK8SClient()
	// if err != nil {
	// 	log.Printf("error getting a client")
	// }
	// go func() {
	// 	for {
	// 		status, err := cl.GetPodStatus("default", fmt.Sprintf("buildID=%s", buildID))
	// 		if err != nil {
	// 			log.Printf("Error: %v", err)
	// 		} else {
	// 			log.Printf(">>> Phase: %v", status.Phase)
	// 			log.Printf(">>> Message: %v", status.Message)
	// 		}
	// 		time.Sleep(5 * time.Second)
	// 	}
	// }()

	eventres := watchres.ResultChan()
	for we := range eventres {
		pods, err := clientset.CoreV1().Pods("default").List(context.TODO(),
			metav1.ListOptions{LabelSelector: fmt.Sprintf("buildID=%s", buildID)})
		if err != nil {
			log.Println("Cannot get pod info")
			return
		}
		//		log.Printf("Log info :\n%v\n", pods)
		for _, podInfo := range (*pods).Items {
			fmt.Printf("pods-name=%v\n", podInfo.Name)
			fmt.Printf("pods-status=%v\n", podInfo.Status.Phase)
			fmt.Printf("pods-message=%v\n", podInfo.Status.Message)
			//		fmt.Printf("pods-condition=%v\n", podInfo.Status.Conditions)
		}

		// log.Println("----")
		// log.Printf("TYPE: %T (%v) %v\n", we.Type, we.Type, we.Type == "DELETED")
		if we.Type == "DELETED" {
			break
		}
		_, ok := we.Object.(*batchv1.Job)
		if !ok {
			continue
		}
		// log.Println("-")
		// log.Println(job.Status.Conditions)
		// log.Println(job.Status.Active > 0)
		// log.Println(job.Status.Succeeded > 0)
	}

	buildDone[buildID] = true
	playWithDocker(imageTag)

	// We should get to the pod for the job, which doesn't seem
	// directly doable a way is to assign a label to the pod, say the
	// build ID and then search for that. This would allow ud to get
	// to the status message, which is where we ask kaniko to put the
	// image SHA
	//	 create a pod with label, and then I get it through LabelSelector. Like it :
	// config, err := clientcmd.BuildConfigFromFlags("", "~/.kube/config")
	// if err != nil {
	// 	println("config build error")
	// }
	// This is what we get:
	// State:          Terminated
	//   Reason:       Completed
	//   Message:      sha256:5bd5af3382faa4e15b33f03b6ab7dfc9405406dcfddf23723588dfc5faeec839
	//   Exit Code:    0
	//   Started:      Mon, 30 May 2022 11:24:45 -0400
	//   Finished:     Mon, 30 May 2022 11:27:49 -0400

	//	client, err := kubernetes.NewForConfig(config)
	// client, err := rest.InClusterConfig()
	// if err != nil {
	// 	log.Println("cannot connect")
	// 	return
	// }
	// pods, err := client.CoreV1().Pods("default").List(context.TODO(),
	// 	v1.ListOptions{LabelSelector: fmt.Sprintf("buildID=%s", buildID)})
	pods, err := clientset.CoreV1().Pods("default").List(context.TODO(),
		metav1.ListOptions{LabelSelector: fmt.Sprintf("buildID=%s", buildID)})
	if err != nil {
		log.Println("Cannot get pod info")
		return
	}
	log.Printf("Log info :\n%v\n", pods)
	for _, podInfo := range (*pods).Items {
		fmt.Printf("pods-name=%v\n", podInfo.Name)
		fmt.Printf("pods-status=%v\n", podInfo.Status.Phase)
		fmt.Printf("pods-condition=%v\n", podInfo.Status.Conditions)
	}

	// for _, v := range pods.Items {
	// 	log := client.CoreV1().Pods("default").GetLogs(v.Name, &v12.PodLogOptions{})
	// 	log.Println(log)
	// }

}

func playWithDocker(imageTag string) {
	//	url := "http://registry.other.net:5000/"
	log.Printf("PlayWithDocker")
	url := "http://192.168.1.8:5000/"
	username := "" // anonymous
	password := "" // anonymous
	hub, err := registry.NewInsecure(url, username, password)
	if err != nil {
		log.Printf("Cannot connect to docker registry")
	}

	// repositories are actually image names
	//
	// 2022/06/02 03:06:26 Digest: sha256:420b8b829b3ecfa95768b68581bbf61b269cba00ad725fd67495668fa45d9f4e
	// 2022/06/02 03:06:26 Tag: a00a0805-815a-4890-9bf7-56283ebc28c3
	// 2022/06/02 03:06:26 registry.manifest.head url=http://192.168.1.8:5000/v2/eval/manifests/a00a0805-815a-4890-9bf7-56283ebc28c3 repository=eval reference=a00a0805-815a-4890-9bf7-56283ebc28c3
	// repositories, err := hub.Repositories()
	// if err != nil {
	// 	log.Printf("Cannot get list of repositories")
	// }
	// for _, r := range repositories {
	// 	log.Printf("Repo: %v\n", r, r)
	// 	tags, err := hub.Tags(r)
	// 	if err != nil {
	// 		log.Printf("Cannot get list of tags")
	// 	}
	// 	for _, t := range tags {
	// 		log.Printf("Tag: %v\n", t)
	// 		digest, err := hub.ManifestDigest(r, t)
	// 		if err != nil {
	// 			continue
	// 		}
	// 		log.Printf("Digest: %v\n", digest)
	// 	}

	// }

	digest, err := hub.ManifestDigest("eval", imageTag)
	if err == nil {
		log.Printf("========== DIGEST FOR eval:%s ===========", imageTag)
		log.Printf("Digest: %v\n", digest)
	}

	// tags, err := hub.Tags("eval")
	// if err != nil {
	// 	log.Printf("Cannot get list of tags")
	// }
	// for _, t := range tags {
	// 	log.Printf("Tag: %v\n", t)
	// 	manifest, err := hub.ManifestDigest("eval", t)
	// 	if err != nil {
	// 		log.Printf("Error %v", err)
	// 	} else {
	// 		log.Printf("Manifest: %v", manifest)
	// 	}
	// }

	log.Printf("===================================================")
	//	manifest, err := hub.ManifestDigest("eval", imageTag)
	manifest, err := hub.Manifest("eval", imageTag)
	if err != nil {
		log.Printf("Error getting digest for image %s", imageTag)
	} else {
		payload, _, _ := manifest.Payload()
		log.Printf("Payload %s", payload)
		log.Printf("Name %v", manifest.Name)
		log.Printf("Tag %v", manifest.Tag)
	}
	log.Printf("===================================================")

	// oc, err := otherdocker.NewClient("http://192.168.1.8:5000/")
	// if err != nil {
	// 	log.Println("Failed to connect")
	// 	panic(err)
	// }

	// history, err := oc.ImageHistory("eval")
	// if err != nil {
	// 	log.Printf("Failed to get history: %v", err)
	// 	panic(err)
	// }
	// for _, h := range history {
	// 	log.Printf("HISTORY: %v", h)
	// }

	// ii, err := oc.InspectImage("eval:" + imageTag)
	// if err != nil {
	// 	log.Printf("Failed to get image info: %v", err)
	// 	panic(err)
	// }
	// log.Printf("INFO: %v", ii)

	// imgs, err := oc.ListImages(docker.ListImagesOptions{All: false})
	// if err != nil {
	// 	log.Printf("Failed to get list of images: %v", err)
	// 	panic(err)
	// }
	// for _, img := range imgs {
	// 	fmt.Println("ID: ", img.ID)
	// 	fmt.Println("RepoTags: ", img.RepoTags)
	// 	fmt.Println("Created: ", img.Created)
	// 	fmt.Println("Size: ", img.Size)
	// 	fmt.Println("VirtualSize: ", img.VirtualSize)
	// 	fmt.Println("ParentId: ", img.ParentID)
	// }

	// client, err := docker.NewClient("http://192.168.1.8:5000/")
	// if err != nil {
	// 	log.Println("Failed to connect")
	// 	panic(err)
	// }
	// imgs, err := client.ListImages(docker.ListImagesOptions{All: false})
	// if err != nil {
	// 	log.Printf("Failed to get lsit of images: %v", err)
	// 	panic(err)
	// }
	// for _, img := range imgs {
	// 	fmt.Println("ID: ", img.ID)
	// 	fmt.Println("RepoTags: ", img.RepoTags)
	// 	fmt.Println("Created: ", img.Created)
	// 	fmt.Println("Size: ", img.Size)
	// 	fmt.Println("VirtualSize: ", img.VirtualSize)
	// 	fmt.Println("ParentId: ", img.ParentID)
	// }
}

func HandleBuildTask(ctx context.Context, t *asynq.Task) error {
	var p BuildPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	log.Printf("Building targets=%s @ branch=%s, commitSHA=%s -> imageTag: %s", p.Target, p.Branch, p.CommitSHA, p.ImageTag)
	log.Printf("-------------------")
	build(p.BuildID, p.Branch, p.CommitSHA, p.Target, p.ImageTag)
	log.Printf("-------------------")
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
