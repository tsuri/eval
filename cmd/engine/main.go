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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	port = "0.0.0.0:50051"
)

const (
	CertChain  = "ca.crt"
	ServerCert = "tls.crt"
	ServerKey  = "tls.key"
)

type server struct{}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func grunt(n int64) int64 {
	var conn *grpc.ClientConn
	log.Printf("Asking grunt")
	return n + 1
	conn, err := grpc.Dial(
		"eval-grunt.default.svc.cluster.local:10051",
		grpc.WithInsecure())
	if err != nil {
		//		log.Fatalf("did not connect: %s", err)
		return 42
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
	log.Printf("Eval service")
	return &pb.EvalResponse{Number: grunt(in.Number) + 1}, nil
}

func main() {
	log.Print("Hello there")

	// d1 := []byte("hello\ngo\n")
	// err := os.WriteFile("/data/hello.txt", d1, 0644)
	// check(err)
	// log.Print("Done")

	// fileBytes, err := ioutil.ReadFile("/data/hello.txt")
	// check(err)
	// fileString := string(fileBytes)
	// log.Print("------------\n")
	// log.Print(fileString)
	// log.Print("------------\n")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	tc := getTlsConfig()
	serverOption := grpc.Creds(credentials.NewTLS(tc))

	s := grpc.NewServer(serverOption)
	pb.RegisterEngineServiceServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func getTlsConfig() *tls.Config {
	certificate := getCertificate()
	certPool := getCertPool()
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}
	return tlsConfig
}

func getCertPool() *x509.CertPool {
	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile(filepath.Join("/app", "Certs", CertChain))
	if err != nil {
		log.Fatalf("failed to read certificates chain: %s", err)
	}
	ok := certPool.AppendCertsFromPEM(bs)
	if !ok {
		log.Fatalf("failed to append certs")
	}
	return certPool
}

func getCertificate() tls.Certificate {
	crt := filepath.Join("/app", "Certs", ServerCert)
	key := filepath.Join("/app", "Certs", ServerKey)
	certificate, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		log.Fatalf("Failed to get certificate")
	}
	return certificate
}

func isInvalidCertificate(ctx context.Context) (error, bool) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		err := status.Error(codes.Unauthenticated, "no peer found")
		return err, true
	}
	tlsAuth, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		err := status.Error(codes.Unauthenticated, "unexpected peer transport credentials")
		return err, true
	}
	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
		err := status.Error(codes.Unauthenticated, "could not verify peer certificate")
		return err, true
	}
	// Check subject common name against configured username
	if !contains(tlsAuth.State.VerifiedChains[0][0].Subject.CommonName) {
		log.Printf("Here failed authenication")
		err := status.Error(codes.Unauthenticated, "invalid subject common name : Unauthenticated Client")
		return err, true
	}
	return nil, false
}

func contains(e string) bool {
	var validClients = []string{"node-grpc-client1", "node-grpc-client2", "node-grpc-client3"}
	for _, a := range validClients {
		if a == e {
			return true
		}
	}
	return false
}
