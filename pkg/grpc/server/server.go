package server

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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
	s.log.Info().Msg("Start new server")

	// graceful stop on Interrupt
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			s.log.Warn().Str("signal", sig.String()).Msg("GracefulStop triggered")
			s.server.GracefulStop()
		}
	}()

	if err := s.server.Serve(*s.listener); err != nil {
		s.log.Fatal().Msg("error serving GRPC traffic")
	}
}

func Build(port string) Server {
	//	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	opts := []grpc.ServerOption{
		grpcCredentials(),
		grpczerolog.UnaryInterceptorWithLogger(&logger),
	}
	server := grpc.NewServer(opts...)

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	return &serverContext{
		server:   server,
		listener: &listener,
		log:      &logger,
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
