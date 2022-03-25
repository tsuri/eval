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

	// only for testing
	pb "eval/proto/engine"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	fmt.Println("EVAL")
	log.Printf("Grunt of %v is %v, no?", in.Number, 1000+in.Number)
	return &pb.EvalResponse{Number: in.Number + 1000}, nil
}

func main() {
	log.Print("Hello there, this is a grunt squad")

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
