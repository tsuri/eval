package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"

	"eval/pkg/grpc/client"

	pbeval "eval/proto/engine"
	pbgrunt "eval/proto/grunt"

	grpczerolog "github.com/philip-bui/grpc-zerolog"
	"github.com/rs/zerolog"
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

type server struct {
	log *zerolog.Logger
}

func connect(service string) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	conn, err := client.NewConnection("eval-grunt.eval.svc.cluster.local:50051",
		filepath.Join(baseDir, caCert),
		filepath.Join(baseDir, clientCert),
		filepath.Join(baseDir, clientKey))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	return conn, nil
}

func build_image() {
	conn, err := connect("eval-build.eval.svc.cluster.local:50051")
	if err != nil {
		log.Fatalf("did not connect")
	}
	defer conn.Close()

}

func grunt(n int64) int64 {
	conn, err := connect("eval-grunt.eval.svc.cluster.local:50051")
	defer conn.Close()

	client := pbgrunt.NewEngineServiceClient(conn)

	response, err := client.Eval(context.Background(), &pbgrunt.EvalRequest{Number: n})
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}

	return response.Number*2 + 1
}

func (s *server) Eval(ctx context.Context, in *pbeval.EvalRequest) (*pbeval.EvalResponse, error) {
	// s.log.Info().Msg("new logger")
	// log.Printf("Eval service")
	// log.Printf("Request from %s on host %s", in.Requester.UserName, in.Requester.HostName)
	return &pbeval.EvalResponse{Number: grunt(in.Number) + 1}, nil
}

func NewServer() *server {
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	logger.Info().Msg("Starting eval engine server")
	return &server{
		log: &logger,
	}
}

func main() {
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
		grpczerolog.UnaryInterceptor(),
		//		grpczerolog.UnaryInterceptor(),
	}

	server := NewServer()

	s := grpc.NewServer(opts...)
	pbeval.RegisterEngineServiceServer(s, server)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
