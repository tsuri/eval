package server

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"

	grpczerolog "github.com/philip-bui/grpc-zerolog"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	baseDir     = "/app/Certs"
	caCert      = "ca.crt"
	clientCert  = "tls.crt"
	clientKey   = "tls.key"
	configFile  = "/app/config/config.yaml"
	varLogLevel = "log.level"
)

// Context
type serverContext struct {
	log      *zerolog.Logger
	v        *viper.Viper
	server   *grpc.Server
	listener *net.Listener
}

// The Server interface defines gRPC server
type Server interface {
	RegisterService(reg func(*grpc.Server))
	Start()
}

func (s *serverContext) RegisterService(reg func(*grpc.Server)) {
	reg(s.server)
}

func (s *serverContext) Start() {
	log.Println("Start new server")
	// TODO serve in a go routine and support grace termination
	if err := s.server.Serve(*s.listener); err != nil {
		// TODO real logger
		log.Fatalf("error serving GRPC traffic")
	}
}

func Build(port string) Server {
	opts := []grpc.ServerOption{
		grpcCredentials(),
		grpczerolog.UnaryInterceptor(),
	}
	server := grpc.NewServer(opts...)

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	return &serverContext{
		server:   server,
		listener: &listener,
	}
}

func grpcCredentials() grpc.ServerOption {
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

	return grpc.Creds(
		credentials.NewTLS(&tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{cert},
			ClientCAs:    certPool}))
}
