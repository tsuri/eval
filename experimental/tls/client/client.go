package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "eval/proto/engine"
)

const (
	endpoint = "engine.eval.net:50051"
	baseDir  = "/data/eval/C"
	certFile = "engine.eval.net.crt"
)

func main() {
	fmt.Printf("Client")

	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		log.Fatalf("bad endpoint err: %v", err)
	}

	creds, err := credentials.NewClientTLSFromFile(filepath.Join(baseDir, certFile), host)
	if err != nil {
		log.Fatalf("failed to load credentials: %v", err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewEngineServiceClient(conn)

	response, err := client.Eval(context.Background(), &pb.EvalRequest{Number: 42})
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}
	log.Printf("Response from server: %s", response.Number)

}
