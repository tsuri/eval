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

func grunt(n int64) int64 {
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(
		"eval-grunt.default.svc.cluster.local:10051",
		grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := pb.NewEngineServiceClient(conn)
	response, err := c.Eval(context.Background(), &pb.EvalRequest{Number: n})
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}
	log.Printf("Response from grunt: %s", response.Number)
	return response.Number
}

func (s *server) Eval(ctx context.Context, in *pb.EvalRequest) (*pb.EvalResponse, error) {
	return &pb.EvalResponse{Number: grunt(in.Number) + 1}, nil
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
