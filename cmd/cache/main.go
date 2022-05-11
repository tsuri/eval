package main

import (
	"context"

	"eval/pkg/grpc/server"

	pbcache "eval/proto/cache"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = "0.0.0.0:50051"
)

type serverContext struct {
	log *zerolog.Logger
	v   *viper.Viper
}

func (s *serverContext) Get(ctx context.Context, in *pbcache.GetRequest) (*pbcache.GetResponse, error) {
	s.log.Info().Str("evaluation", in.Evaluation).Msg("Cache Get")
	return &pbcache.GetResponse{}, nil
}

func serviceRegister(server server.Server) func(*grpc.Server) {
	return func(s *grpc.Server) {
		context := serverContext{}
		context.log = server.Logger()
		context.v = server.Config()

		pbcache.RegisterCacheServiceServer(s, &context)
		reflection.Register(s)
	}
}

func main() {
	server := server.Build(port)
	server.RegisterService(serviceRegister(server))
	server.Start()
}
