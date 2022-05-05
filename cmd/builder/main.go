package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"eval/pkg/grpc/server"
	pb "eval/proto/builder"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	port = ":50051"
)

const (
	baseDir    = "/data/eval/certificates"
	CaCert     = "ca.crt"
	ServerCert = "tls.crt"
	ServerKey  = "tls.key"
)

type serverContext struct {
	log       *zerolog.Logger
	v         *viper.Viper
	clientSet *kubernetes.Clientset
	asynq     *Asynq
}

func (s *serverContext) Build(ctx context.Context, in *pb.BuildRequest) (*pb.BuildResponse, error) {
	s.log.Info().Msg("Builder summoned")

	r := asynq.RedisClientOpt{Addr: "redis.eval.svc.cluster.local:6379"}
	c := asynq.NewClient(r)

	log.Printf("Enqueuing task")
	branch := in.Branch
	commit := in.CommitSHA
	target := in.Target
	t := NewBuildTask(branch, commit, target)
	ti, err := c.Enqueue(t)
	if err != nil {
		log.Fatal("could not enqueue task: %v", err)
	}
	log.Printf("INFO: %v", ti)

	//	job := buildJobSpec(in.Branch, in.CommitSHA)
	//	log.Printf("%v\n", job)
	return &pb.BuildResponse{Response: "something built"}, nil
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

func serviceRegister(server server.Server, asynq *Asynq) func(*grpc.Server) {
	return func(s *grpc.Server) {
		context := serverContext{}
		context.log = server.Logger()
		context.v = server.Config()
		context.clientSet = connectToK8s()
		context.asynq = asynq
		context.log.Info().Msg("Registering service")
		pb.RegisterBuilderServiceServer(s, &context)
		reflection.Register(s)
	}
}

type BuildPayload struct {
	Branch    string
	CommitSHA string
	Target    []string
}

//  return errorin addition to task
func NewBuildTask(branch string, commitSHA string, target []string) *asynq.Task {
	payload, err := json.Marshal(BuildPayload{Branch: branch, CommitSHA: commitSHA, Target: target})
	if err != nil {
		return nil
	}
	return asynq.NewTask("image:build", payload)
}

func build(branch string, commitSHA string, targets []string) {
	clientset := connectToK8s()
	batchAPI := clientset.BatchV1()
	jobs := batchAPI.Jobs("default")

	// 	cm := v1.ConfigMap{
	// 		TypeMeta: metav1.TypeMeta{
	// 			Kind:       "ConfigMap",
	// 			APIVersion: "v1",
	// 		},
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Name:      "my-config-map",
	// 			Namespace: "default",
	// 		},
	// 		Data: map[string]string{"dockerfile": `
	// FROM registry.other.net:5000/eval/base-build AS builder
	// #FROM debian:buster-slim AS builder

	// # RUN apt-get update
	// # RUN apt-get install --yes wget build-essential python3
	// # RUN wget -q https://releases.bazel.build/5.1.0/release/bazel-5.1.0-linux-x86_64 -O /usr/bin/bazel
	// # RUN chmod +x /usr/bin/bazel

	// COPY . /eval
	// WORKDIR /eval
	// RUN echo $PWD
	// RUN ls
	// #RUN /usr/bin/bazel build //test:runner
	// RUN /usr/bin/bazel build //test:test  //test:runner //test:sub //test:another

	// FROM debian:buster
	// #RUN apt-get update && apt-get install --yes python3
	// #FROM gcr.io/distroless/python3
	// COPY --from=builder /eval/bazel-bin/test     /app
	// ENTRYPOINT /app/runner_/runner
	// `,
	// 		},
	// 	}

	// 	_, err := clientset.CoreV1().ConfigMaps("default").Create(context.TODO(), &cm, metav1.CreateOptions{})
	// 	if err != nil {
	// 		log.Printf("Error creating config map: %v", err)
	// 	}

	var backOffLimit int32 = 0
	var ttlSecondsAfterFinished int32 = 10

	gitContext := "git://gitea-service.gitea-repo.svc.cluster.local:3000/mav/eval.git#" + branch + "#" + commitSHA

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
							Name:  "kaniko",
							Image: "gcr.io/kaniko-project/executor:debug",
							Args: []string{"--insecure",
								"--insecure-pull",
								"--skip-tls-verify",
								"--build-arg", "TARGETS=" + strings.Join(targets, " "),
								"--destination=registry.other.net:5000/test:bar",
								"--context", gitContext,
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

	_, err := jobs.Create(context.TODO(), jobSpec, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("Failed to create K8s job. %v", err)
	}
	log.Println("Kaniko job created")

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

}

func HandleBuildTask(ctx context.Context, t *asynq.Task) error {
	var p BuildPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	log.Printf("Building targets=%s @ branch=%s, commitSHA=%s", p.Target, p.Branch, p.CommitSHA)
	build(p.Branch, p.CommitSHA, p.Target)
	log.Printf("Done building branch=%s, commitSHA=%s", p.Branch, p.CommitSHA)
	return nil
}

type Asynq struct {
	server *asynq.Server
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
	return &Asynq{
		server: server,
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
