package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"

	pb "eval/proto/grunt"

	"github.com/go-redis/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	port = ":50051"
)

const (
	baseDir    = "/app/Certs"
	CaCert     = "ca.crt"
	ServerCert = "tls.crt"
	ServerKey  = "tls.key"
)

type server struct{}

func (s *server) Eval(ctx context.Context, in *pb.EvalRequest) (*pb.EvalResponse, error) {
	log.Printf("Builder")
	return &pb.EvalResponse{Number: in.Number + 1000}, nil
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

func main() {
	log.Print("Hello there, this is the builder")

	//-----------------------------------------------------------------------------
	// TODO add an host entry for registry.other.net
	log.Println("Trying docker")
	// dockerClient, err := docker.NewClient("http://192.168.1.8:5000")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// imgs, err := dockerClient.ListImages(docker.ListImagesOptions{All: false})
	// if err != nil {
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

	// url := "https://registry-1.docker.io/"
	// username := "" // anonymous
	// password := "" // anonymous
	// hub, err := registry.New(url, username, password)

	// repositories, err := hub.Repositories()
	// for r := range repositories {
	// 	log.Println(r)
	// }
	// clientset := connectToK8s()

	//-----------------------------------------------------------------------------
	log.Println("Kubernetes")
	batchApi := clientset.BatchV1()
	jobs := batchApi.Jobs("default")

	var backOffLimit int32 = 0
	var ttlSecondsAfterFinished int32 = 10

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
								"--destination=registry.other.net:5000/test:bar",
								"--context=git://gitea-service.gitea-repo.svc.cluster.local:3000/mav/eval.git",
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

	_, err = jobs.Create(context.TODO(), jobSpec, metav1.CreateOptions{})
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
		log.Println(we.Type)
		job, ok := we.Object.(*batchv1.Job)
		if !ok {
			continue
		}
		log.Println(job.Status.Conditions)
	}

	//-----------------------------------------------------------------------------
	api := clientset.CoreV1()

	pods, err := api.Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	//-----------------------------------------------------------------------------
	var (
		host       = "redis.eval.svc.cluster.local"
		redis_port = "6379"
	)

	client := redis.NewClient(&redis.Options{
		Addr: host + ":" + redis_port,
		//		Password: password,
		DB: 0,
	})

	pong, err := client.Ping().Result()
	if err != nil {
		log.Println("---------------------")
		log.Fatal(err)
	}
	log.Println("---------------------")
	log.Println(pong)
	log.Println("---------------------")

	client.Incr("something.kcount")
	client.Incr("something.nkcount")

	val, err := client.Get("something.kcount").Result()
	if err != nil {
		panic(err)
	}
	result := string("key count: " + string(val))
	log.Println(result)

	cert, err := tls.LoadX509KeyPair(filepath.Join(baseDir, ServerCert),
		filepath.Join(baseDir, ServerKey))
	if err != nil {
		log.Fatalf("Failed to get certificate")
	}

	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile(filepath.Join(baseDir, CaCert))
	if err != nil {
		log.Fatalf("failed to read certificates chain: %s", err)
	}
	ok := certPool.AppendCertsFromPEM(bs)
	if !ok {
		log.Fatalf("failed to append certs")
	}

	opts := []grpc.ServerOption{
		grpc.Creds(
			credentials.NewTLS(&tls.Config{
				ClientAuth:   tls.RequireAndVerifyClientCert,
				Certificates: []tls.Certificate{cert},
				ClientCAs:    certPool})),
	}

	s := grpc.NewServer(opts...)
	pb.RegisterEngineServiceServer(s, &server{})

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

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

func printPVCs(pvcs *v1.PersistentVolumeClaimList) {
	template := "%-32s%-8s%-8s\n"
	fmt.Printf(template, "NAME", "STATUS", "CAPACITY")
	for _, pvc := range pvcs.Items {
		quant := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		fmt.Printf(
			template,
			pvc.Name,
			string(pvc.Status.Phase),
			quant.String())
	}
}
