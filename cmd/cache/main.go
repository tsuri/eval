package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	a "eval/pkg/actions"
	"eval/pkg/agraph"
	"eval/pkg/db"
	"eval/pkg/grpc/client"
	"eval/pkg/grpc/server"
	"eval/pkg/sizeof"
	"eval/pkg/types"

	pbaction "eval/proto/action"
	pbagraph "eval/proto/agraph"
	pbasync "eval/proto/async_service"
	pbbuilder "eval/proto/builder"
	pbcache "eval/proto/cache"
	pbrunner "eval/proto/runner"

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

	//	"k8s.io/utils/strings/slices"
	"golang.org/x/exp/slices"
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

// func (s *serverContext) CacheExperiment(ctx context.Context) {
// 	if err := s.cache.Set(&cache.Item{
// 		Ctx:   ctx,
// 		Key:   "image.build",
// 		Value: &Object{Str: "bar", Num: 42},
// 		TTL:   time.Hour,
// 	}); err != nil {
// 		panic(err)
// 	}

// 	var wanted Object
// 	if err := s.cache.Get(ctx, "image.build", &wanted); err == nil {
// 		fmt.Println(wanted)
// 	}
// }

var downstreamOperation = make(map[string]string)
var waitingOn = make(map[string]string)

type cacheValue struct {
	Action    *pbaction.Action
	Operation string
}

var cacheContent *CacheBackend

type CacheBackend struct {
	data map[string][]cacheValue

	// The last we know for an upstream operation
	upstreamOperation map[string]pbasync.Operation
}

func NewCacheBackend() *CacheBackend {
	return &CacheBackend{
		data:              make(map[string][]cacheValue),
		upstreamOperation: make(map[string]pbasync.Operation),
	}
}

func Compatible(new, old *pbaction.BuildImageConfig) bool {
	fmt.Printf("new: %v\nold: %v\n", new.BazelTargets, old.BazelTargets)
	for _, t := range new.BazelTargets {
		if !slices.Contains(old.BazelTargets, t) {
			return false
		}
	}
	return true
}

func (c *CacheBackend) Get(s *serverContext, action *pbaction.Action) (string, error) {
	digest, err := a.ActionDigest(action)
	if err != nil {
		return "", err
	}

	buildConfig := pbaction.BuildImageConfig{}
	_ = action.Config.UnmarshalTo(&buildConfig)

	if cv, present := c.data[digest]; present {
		s.log.Info().Str("CACHE", fmt.Sprintf("%v", cv)).Msg("****************")
		for _, c := range cv {
			cachedBuildConfig := pbaction.BuildImageConfig{}
			if err := c.Action.Config.UnmarshalTo(&cachedBuildConfig); err == nil {
				if Compatible(&buildConfig, &cachedBuildConfig) {
					return c.Operation, nil
				}
			}
		}
		s.log.Info().Str("op", cv[0].Operation).Msg("previous operation")
	}

	return "", fmt.Errorf("no cached operation")
}

func (c *CacheBackend) Put(digest, operation string, action *pbaction.Action) {
	c.data[digest] = append(c.data[digest], cacheValue{
		Action:    action,
		Operation: operation,
	})
}

func init() {
	cacheContent = NewCacheBackend()
}

func GetAction(agraph *pbagraph.AGraph, fullValuePath string) (*pbaction.Action, error) {
	valueNameComponents := strings.Split(fullValuePath, ".")
	if agraph.Name != valueNameComponents[0] {
		return nil, errors.New("wrong action graph")
	}

	value := strings.Join(valueNameComponents[1:], ".")

	log.Printf("AGRAPH: %s %v", value, agraph.Actions)
	log.Printf("=== %v ===", agraph.Actions[value])
	for k, v := range agraph.Actions {
		log.Printf("   %s: %v", k, v)
	}
	if a, present := agraph.Actions[value]; present {
		return a, nil
	}

	return nil, fmt.Errorf("GetAction")
}

// func getResponse[T any](m *pbasync.Operation) (*T, error) {
// 	r := m.GetResponse()
// 	if r == nil {
// 		return nil, errors.New("failed to get operation response")
// 	}

// 	out := &T{}
// 	if err := r.UnmarshalTo(out); err {
// 		return nil, err
// 	}

// 	return out, nil
// }

func createJob(ctx context.Context, actions *pbagraph.AGraph) {
	log.Printf("createJob")
	conn, err := client.Connect("eval-runner.eval.svc.cluster.local:50051")
	if err != nil {
		log.Fatalf("did not connect")
	}
	defer conn.Close()

	runnerClient := pbrunner.NewRunnerServiceClient(conn)
	_, err = runnerClient.CreateJob(ctx, &pbrunner.CreateJobRequest{})
	if err != nil {
		log.Printf("Error in create job")
	}
}

func (s *serverContext) Get(ctx context.Context, in *pbcache.GetRequest) (*pbasync.Operation, error) {
	s.log.Info().Int64("proto size", sizeof.DeepSize(in.Context)).Msg("Size")

	actions := agraph.EssentialActions(in.Context.Actions, in.Value)

	s.log.Info().Str("value", in.Value).Msg("get cache")

	mainAction, err := GetAction(actions, in.Value)
	if err != nil {
		return nil, err
	}

	//	createJob(ctx, in.Context.Actions)

	if !in.SkipCaching {
		cachedOperation, err := cacheContent.Get(s, mainAction)
		if err == nil {
			if lastResult, present := cacheContent.upstreamOperation[cachedOperation]; present && lastResult.Done {
				buildResponse := new(pbbuilder.BuildResponse)
				if err := lastResult.GetResponse().UnmarshalTo(buildResponse); err != nil {
					log.Fatalf("4 Cannot unmarhshal result")
				}

				buildInfo := map[string]string{
					"image_name":   buildResponse.ImageName,
					"image_tag":    buildResponse.ImageTag,
					"image_digest": buildResponse.ImageDigest,
				}

				log.Printf("%v", buildInfo)
				var response *anypb.Any
				response, err = anypb.New(&pbcache.GetResponse{
					//					Value: types.StringScalar(buildResponse.Response),
					Value: types.StringDictionary(buildInfo),
				})
				if err != nil {
					panic(err)
				}
				return &pbasync.Operation{
					Name:   cachedOperation,
					Done:   true,
					Result: &pbasync.Operation_Response{response},
				}, nil

				return &lastResult, nil
			}

			// we should return a new operation; and we shouldn't return 'done' inconditionally as
			// the operation could be ongoing
			result, err := anypb.New(&pbcache.GetResponse{})
			if err != nil {
				panic(err)
			}
			return &pbasync.Operation{
				Name:   cachedOperation,
				Done:   true,
				Result: &pbasync.Operation_Response{result},
			}, nil
		}
	}

	digest, err := a.ActionDigest(mainAction)
	if err != nil {
		return nil, err
	}

	agraph.Execute(actions)

	buildImageConfig := new(pbaction.BuildImageConfig)
	actions.Actions["build"].Config.UnmarshalTo(buildImageConfig)
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

	cacheContent.Put(digest, id.String(), actions.Actions["build"])

	// cacheContent[digest] = append(cacheContent[digest], cacheValue{
	// 	Action:    actions.Actions[0],
	// 	Operation: id.String(),
	// })

	return &pbasync.Operation{
		Name:   id.String(),
		Done:   operation.Done,
		Result: &pbasync.Operation_Response{result},
	}, nil
}

func (s *serverContext) GetOperation(ctx context.Context, in *pbasync.GetOperationRequest) (*pbasync.Operation, error) {
	if lastResult, present := cacheContent.upstreamOperation[in.Name]; present && lastResult.Done {
		return &lastResult, nil
	}
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

		if err != nil {
			log.Fatal("Failure to get operation from upstream")
		}

		cacheContent.upstreamOperation[in.Name] = *operation

		buildResponse := new(pbbuilder.BuildResponse)
		if operation.Done {
			if err := operation.GetResponse().UnmarshalTo(buildResponse); err != nil {
				log.Fatal("Cannot unmarhshal result from builder")
			}
		}

		buildInfo := map[string]string{
			"image_name":   buildResponse.ImageName,
			"image_tag":    buildResponse.ImageTag,
			"image_digest": buildResponse.ImageDigest,
		}

		log.Printf("%v", buildInfo)
		var response *anypb.Any
		response, err = anypb.New(&pbcache.GetResponse{
			//					Value: types.StringScalar(buildResponse.Response),
			Value: types.StringDictionary(buildInfo),
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
