package main

import (
	"context"
	"fmt"
	"log"
	"time"

	a "eval/pkg/actions"
	"eval/pkg/agraph"
	"eval/pkg/db"
	"eval/pkg/grpc/client"
	"eval/pkg/grpc/server"
	"eval/pkg/types"

	pbaction "eval/proto/action"
	pbagraph "eval/proto/agraph"
	pbasync "eval/proto/async_service"
	pbbuilder "eval/proto/builder"
	pbcache "eval/proto/cache"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
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

func buildImage(ctx context.Context, config *pbaction.BuildImageConfig) (*pbasync.Operation, error) {
	// ok, a filure to connect here doesn' return error
	conn, err := client.Connect("eval-builder.eval.svc.cluster.local:50051")
	if err != nil {
		log.Fatalf("did not connect")
	}
	defer conn.Close()

	client := pbbuilder.NewBuilderServiceClient(conn)

	// md, _ := metadata.FromIncomingContext(ctx)
	// log.Printf("BUILD METADATA: %v", md["user"][0])

	// ctx = metadata.NewOutgoingContext(ctx, md)

	response, err := client.Build(ctx, &pbbuilder.BuildRequest{
		CommitSHA: config.CommitPoint.CommitSha,
		Branch:    config.CommitPoint.Branch,
		Target:    config.BazelTargets,
	})
	if err != nil {
		log.Printf("bad answer from builder")
		return nil, err
	} else {
		log.Printf("response %v", response)
		return response, nil
	}
}

type serverContext struct {
	log   *zerolog.Logger
	v     *viper.Viper
	db    *gorm.DB
	cache *cache.Cache
}

type CacheInfo struct {
}

type Object struct {
	Str string
	Num int
}

func (s *serverContext) CacheExperiment(ctx context.Context) {
	if err := s.cache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   "image.build",
		Value: &Object{Str: "bar", Num: 42},
		TTL:   time.Hour,
	}); err != nil {
		panic(err)
	}

	var wanted Object
	if err := s.cache.Get(ctx, "image.build", &wanted); err == nil {
		fmt.Println(wanted)
	}
}

var downstreamOperation = make(map[string]string)

type cacheValue struct {
	Action    *pbaction.Action
	Operation string
}

var cacheContent = make(map[string][]cacheValue)

func GetAction(agraph *pbagraph.AGraph, value string) (*pbaction.Action, error) {
	if value == "image.build" && agraph.Name == "image" && agraph.Actions[0].Name == "image.build" {
		return agraph.Actions[0], nil
	}

	return nil, fmt.Errorf("GetAction")
}

func (s *serverContext) Get(ctx context.Context, in *pbcache.GetRequest) (*pbasync.Operation, error) {
	actions := agraph.EssentialActions(in.Context.Actions, in.Value)

	mainAction, err := GetAction(actions, in.Value)
	if err != nil {
		return nil, err
	}
	digest, err := a.ActionDigest(mainAction)
	if err != nil {
		return nil, err
	}

	if cv, ok := cacheContent[digest]; ok {
		s.log.Info().Str("op", cv[0].Operation).Msg("previous operation")
	}

	agraph.Execute(actions)

	buildImageConfig := new(pbaction.BuildImageConfig)
	actions.Actions[0].Config.UnmarshalTo(buildImageConfig)
	s.log.Info().Str("Image Name", buildImageConfig.ImageName).Msg("BuildConfig")

	operation, err := buildImage(ctx, buildImageConfig)

	// here we should populate a map of results
	cacheGetResponse := pbcache.GetResponse{}

	result, err := anypb.New(&cacheGetResponse)
	if err != nil {
		panic(err)
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	downstreamOperation[id.String()] = operation.Name

	cacheContent[digest] = []cacheValue{{
		Action:    actions.Actions[0],
		Operation: id.String(),
	}}

	return &pbasync.Operation{
		Name:   id.String(),
		Done:   operation.Done,
		Result: &pbasync.Operation_Response{result},
	}, nil
}

func (s *serverContext) GetOperation(ctx context.Context, in *pbasync.GetOperationRequest) (*pbasync.Operation, error) {

	// ok, a filure to connect here doesn' return error
	conn, err := client.Connect("eval-builder.eval.svc.cluster.local:50051")
	if err != nil {
		log.Fatalf("did not connect")
	}
	defer conn.Close()

	client := pbasync.NewOperationsClient(conn)

	if operationID, ok := downstreamOperation[in.Name]; ok {
		operation, err := client.GetOperation(ctx,
			&pbasync.GetOperationRequest{
				Name: operationID,
			})

		var response *anypb.Any
		response, err = anypb.New(&pbcache.GetResponse{
			Value: types.StringScalar("xvalue from cache"),
		})
		if err != nil {
			panic(err)
		}
		return &pbasync.Operation{
			Name:   in.Name,
			Done:   operation.Done,
			Result: &pbasync.Operation_Response{response},
		}, nil
	}

	/// should be an error here, but I don't remember how
	return &pbasync.Operation{
		Name: in.Name,
		Done: true,
	}, nil
}

func serviceRegister(server server.Server) func(*grpc.Server) {
	return func(s *grpc.Server) {
		context := serverContext{}
		context.log = server.Logger()
		context.v = server.Config()
		context.db, _ = db.NewDB("cache", &CacheInfo{})

		redisClient := redis.NewClient(&redis.Options{
			Addr: "redis.eval.svc.cluster.local:6379",
		})

		context.cache = cache.New(&cache.Options{
			Redis: redisClient,
		})

		pbcache.RegisterCacheServiceServer(s, &context)
		pbasync.RegisterOperationsServer(s, &context)
		reflection.Register(s)
	}
}

func main() {
	server := server.Build(port)
	server.RegisterService(serviceRegister(server))
	server.Start()
}
