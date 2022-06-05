package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"

	"eval/pkg/agraph"
	pbasync "eval/proto/async_service"
	pbrunner "eval/proto/runner"

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
	baseDir    = "/data/eval/certificates"
	CaCert     = "ca.crt"
	ServerCert = "tls.crt"
	ServerKey  = "tls.key"
)

type server struct {
	clientSet *kubernetes.Clientset
}

func connectToK8s() *kubernetes.Clientset {
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

func actionJobSpec(imageName string, imageTag string, target string) *batchv1.Job {

	var backOffLimit int32 = 0
	var ttlSecondsAfterFinished int32 = 10

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "action-",
			Namespace:    "eval",
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"action": "somehing",
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
							Name:    "action",
							Image:   "localhost:5000/" + imageName + ":" + imageTag,
							Command: []string{"/app/actions/wrapper/wrapper_/wrapper"},
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

func (s *server) CreateJob(ctx context.Context, in *pbrunner.CreateJobRequest) (*pbasync.Operation, error) {
	log.Printf("CreateJob")

	batchAPI := s.clientSet.BatchV1()
	jobs := batchAPI.Jobs("eval")

	agraph.Dump(in.Actions)

	_, err := jobs.Create(ctx, actionJobSpec("eval", "295c4e89-7a5e-454d-8600-457af37ec0b5", "//actions/wrapper"), metav1.CreateOptions{})
	if err != nil {
		log.Printf("ERROR RUNNING JOB: %v\n", err)
		//		return nil, errors.Wrap(err, "failed to create k8s job")
		return nil, fmt.Errorf("failed to create ks job")
	}

	return &pbasync.Operation{}, nil
}

func main() {
	log.Print("Hello there, this is a runner squad")

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
	pbrunner.RegisterRunnerServiceServer(s, &server{
		clientSet: connectToK8s(),
	})

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
