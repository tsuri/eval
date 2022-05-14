package main

import (
	"context"
	"fmt"
	"time"

	"eval/pkg/db"
	"eval/pkg/grpc/server"

	pbasync "eval/proto/async_service"
	pbcache "eval/proto/cache"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/anypb"
	"gorm.io/gorm"
)

const (
	port = "0.0.0.0:50051"
)

type serverContext struct {
	log *zerolog.Logger
	v   *viper.Viper
	db  *gorm.DB
}

type CacheInfo struct {
}

type Object struct {
	Str string
	Num int
}

func (s *serverContext) Get(ctx context.Context, in *pbcache.GetRequest) (*pbasync.Operation, error) {
	s.log.Info().Str("evaluation", in.Evaluation).Msg("Cache Get")

	for _, value := range in.Values {
		s.log.Info().Str("value", value).Msg("Value request")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "redis.eval.svc.cluster.local:6379",
	})

	mycache := cache.New(&cache.Options{
		Redis: redisClient,
	})

	if err := mycache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   "FOO",
		Value: &Object{Str: "bar", Num: 42},
		TTL:   time.Hour,
	}); err != nil {
		panic(err)
	}

	var wanted Object
	if err := mycache.Get(ctx, "FOO", &wanted); err == nil {
		fmt.Println(wanted)
	}

	cacheGetResponse := pbcache.GetResponse{}
	result, err := anypb.New(&cacheGetResponse)
	if err != nil {
		panic(err)
	}
	return &pbasync.Operation{
		Name:   "something",
		Done:   true,
		Result: &pbasync.Operation_Response{result},
	}, nil

}

func serviceRegister(server server.Server) func(*grpc.Server) {
	return func(s *grpc.Server) {
		context := serverContext{}
		context.log = server.Logger()
		context.v = server.Config()
		context.db, _ = db.NewDB("cache", &CacheInfo{})

		pbcache.RegisterCacheServiceServer(s, &context)
		reflection.Register(s)
	}
}

func main() {
	server := server.Build(port)
	server.RegisterService(serviceRegister(server))
	server.Start()
}
