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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "eval/proto/engine"
)

const (
	endpoint   = "engine.eval.net:50051"
	baseDir    = "/data/eval/C"
	certFile   = "evalctl.crt"
	keyFile    = "evalctl.key"
	caCertFile = "evalCA.crt"
)

func main() {
	fmt.Printf("Client")

	cert, err := tls.LoadX509KeyPair(filepath.Join(baseDir, certFile),
		filepath.Join(baseDir, keyFile))
	if err != nil {
		log.Fatalf("failed to get certificate")
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

	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		log.Fatalf("bad endpoint err: %v", err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(
			&tls.Config{
				ServerName:   host,
				Certificates: []tls.Certificate{cert},
				RootCAs:      certPool,
			},
		)),
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
