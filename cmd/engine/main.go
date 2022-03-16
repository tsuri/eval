package main

import (
	"context"
	"log"
	"net"

	pb "eval/proto/engine"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":50051"
)

type server struct{}

func (s *server) Eval(ctx context.Context, in *pb.EvalRequest) (*pb.EvalResponse, error) {
	return &pb.EvalResponse{Number: in.Number + 1}, nil
}

func main() {
	log.Print("Hello there")
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterEngineServiceServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
