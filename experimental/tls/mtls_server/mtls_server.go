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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	port       = ":50051"
	baseDir    = "/data/eval/C"
	certFile   = "engine.eval.net.crt"
	keyFile    = "engine.eval.net.key"
	caCertFile = "evalCA.crt"
)

type server struct{}

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

	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile(filepath.Join(baseDir, caCertFile))
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
