package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"path/filepath"

	pb "eval/proto/engine"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	port     = ":50051"
	baseDir  = "/data/eval/C"
	certFile = "engine.eval.net.crt"
	keyFile  = "engine.eval.net.key"
)

type server struct{}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func (s *server) Eval(ctx context.Context, in *pb.EvalRequest) (*pb.EvalResponse, error) {
	log.Printf("Eval service")
	return &pb.EvalResponse{Number: in.Number + 1}, nil
}

func main() {
	log.Print("Server here")

	cert, err := tls.LoadX509KeyPair(filepath.Join(baseDir, certFile),
		filepath.Join(baseDir, keyFile))
	if err != nil {
		log.Fatalf("Failed to get certificate")
	}

	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
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
