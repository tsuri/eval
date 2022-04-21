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

	"github.com/fsnotify/fsnotify"
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
	Logger() *zerolog.Logger
	Config() *viper.Viper
}

func (s *serverContext) RegisterService(reg func(*grpc.Server)) {
	reg(s.server)
}

func (s *serverContext) Logger() *zerolog.Logger {
	return s.log
}

func (s *serverContext) Config() *viper.Viper {
	return s.v
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

	viper := viper.New()
	// viper.SetDefault(varPathToConfig, configFile)
	viper.SetDefault(varLogLevel, "info")
	viper.SetConfigFile("/app/config/config.yaml")
	//	viper.AddConfigPath("/app/config")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatalf("failed to read configuration")
	}

	logger.Info().Str(varLogLevel, viper.GetString(varLogLevel)).Msg("Log level")
	//	logger.Info().Str("commit", binfo.BuildInfo.GitCommit).Msg("build info")

	//	viper.Debug()

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		logger.Info().Msg("Config changed")
		//	logrus.WithField("file", e.Name).Warn("Config file changed")
		//	setLogLevel(c.GetLogLevel())
	})

	return &serverContext{
		server:   server,
		listener: &listener,
		log:      &logger,
		v:        viper,
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
