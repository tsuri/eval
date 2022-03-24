package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"

	pb "eval/proto/engine"

	"eval/pkg/grpc/client"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	port = "0.0.0.0:50051"
)

const (
	baseDir    = "/app/Certs"
	caCert     = "ca.crt"
	clientCert = "tls.crt"
	clientKey  = "tls.key"
)

type server struct{}

func grunt(n int64) int64 {
	log.Printf("Asking grunt")
	return n + 1000
	var conn *grpc.ClientConn
	conn, err := client.NewConnection("eval-grunt.default.svc.cluster.local:50051",
		filepath.Join(baseDir, caCert),
		filepath.Join(baseDir, clientCert),
		filepath.Join(baseDir, clientKey))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()
	client := pb.NewEngineServiceClient(conn)
	response, err := client.Eval(context.Background(), &pb.EvalRequest{Number: n})
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}

	return response.Number + 1
	// conn, err := grpc.Dial(
	// 	"eval-grunt.default.svc.cluster.local:10051",
	// 	grpc.WithInsecure())
	// if err != nil {
	// 	//		log.Fatalf("did not connect: %s", err)
	// 	return 42
	// }
	// defer conn.Close()

	// c := pb.NewEngineServiceClient(conn)
	// response, err := c.Eval(context.Background(), &pb.EvalRequest{Number: n})
	// if err != nil {
	// 	log.Fatalf("Error when calling Eval: %s", err)
	// }
	// log.Printf("Response from grunt: %s", response.Number)
	// return response.Number
}

func (s *server) Eval(ctx context.Context, in *pb.EvalRequest) (*pb.EvalResponse, error) {
	log.Printf("Eval service")
	return &pb.EvalResponse{Number: grunt(in.Number) + 1}, nil
}

func main() {
	log.Print("Hello there")

	cert, err := tls.LoadX509KeyPair(filepath.Join(baseDir, clientCert),
		filepath.Join(baseDir, clientKey))
	if err != nil {
		log.Fatalf("Failed to get certificate")
	}

	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile(filepath.Join(baseDir, caCert))
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
