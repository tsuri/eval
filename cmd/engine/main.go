package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"eval/pkg/actions"
	"eval/pkg/db"
	"eval/pkg/grpc/client"
	"eval/pkg/grpc/server"
	"eval/pkg/types"

	pbaction "eval/proto/action"
	pbasync "eval/proto/async_service"
	pbcache "eval/proto/cache"
	pbeval "eval/proto/engine"
	pbtypes "eval/proto/types"

	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/anypb"

	"gorm.io/gorm"
)

const (
	port = "0.0.0.0:50051"
)

type serverContext struct {
	log             *zerolog.Logger
	v               *viper.Viper
	db              *gorm.DB
	cache           pbcache.CacheServiceClient
	cacheOperations pbasync.OperationsClient
}

// func buildImage(ctx context.Context, in *pbeval.BuildRequest) *pbeval.BuildResponse {
// 	// ok, a filure to connect here doesn' return error
// 	conn, err := client.Connect("eval-builder.eval.svc.cluster.local:50051")
// 	if err != nil {
// 		log.Fatalf("did not connect")
// 	}
// 	defer conn.Close()
// 	client := pbbuilder.NewBuilderServiceClient(conn)

// 	md, _ := metadata.FromIncomingContext(ctx)
// 	log.Printf("BUILD METADATA: %v", md["user"][0])

// 	ctx = metadata.NewOutgoingContext(ctx, md)

// 	response, err := client.Build(ctx, &pbbuilder.BuildRequest{
// 		CommitSHA: in.CommitSHA,
// 		Branch:    in.Branch,
// 		Target:    in.Target,
// 	})
// 	if err != nil {
// 		log.Printf("bad answer from builder")
// 	} else {
// 		log.Printf("response %v", response)
// 	}
// 	return &pbeval.BuildResponse{
// 		ImageName: response.ImageName,
// 		ImageTag:  response.ImageTag,
// 		Response:  response.Response,
// 	}
// }

func get(evalID string) *pbcache.GetResponse {
	conn, err := client.Connect("eval-cache.eval.svc.cluster.local:50051")
	defer conn.Close()

	cache := pbcache.NewCacheServiceClient(conn)

	operation, err := cache.Get(context.Background(), &pbcache.GetRequest{Evaluation: evalID})
	if err != nil {
		log.Fatalf("Error when calling Get: %s", err)
	}

	response := new(pbcache.GetResponse)
	if err := operation.GetResponse().UnmarshalTo(response); err != nil {
		log.Fatal("Cannot unmarhshal result")
	}

	return response
}

type Operation struct {
	EvaluationID uuid.UUID `gorm:"type:uuid;"` // `gorm:"type:uuid;primary_key;"`
	DependentID  uuid.UUID
	Value        string
}

type Evaluation struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time

	User     string
	Hostname string

	Operations []Operation `gorm:"ForeignKey:ID`
}

// func (e *Evaluation) BeforeCreate(tx *gorm.DB) error {
// 	uuid, err := uuid.NewV4()
// 	if err != nil {
// 		return err
// 	}
// 	tx.Statement.SetColumn("ID", uuid)
// 	return nil
// }

func (s *serverContext) newEvaluation(ctx context.Context, id uuid.UUID, operations map[string]*pbasync.Operation) error {
	user := "unknown"
	hostname := "unknown"
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		user = md["user"][0]
		hostname = md["hostname"][0]
	}

	// not sure why just assigning Operations here dosen't work. When I try only
	// one entry is inserted in the Operations table. So we explicitely insert all in the loop below
	// With that the query we use later works fine
	evaluation := Evaluation{
		ID:         id,
		User:       user,
		Hostname:   hostname,
		Operations: nil, //operations,
	}

	s.log.Info().Str("operations", fmt.Sprintf("%v", operations)).Msg("CREATE")
	for valueName, o := range operations {
		s.db.Create(&Operation{
			EvaluationID: id,
			DependentID:  uuid.Must(uuid.FromString(o.Name)),
			Value:        valueName,
		})
	}

	result := s.db.Create(&evaluation)

	if result.Error != nil && result.RowsAffected != 1 {
		s.log.Err(result.Error).Msg("Error occurred while creating a new evaluation")
		return result.Error
	}
	return nil
}

func (s *serverContext) Eval(ctx context.Context, in *pbeval.EvalRequest) (*pbasync.Operation, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	s.log.Info().Str("Evaluation", id.String()).Msg("-------")
	operations := make(map[string]*pbasync.Operation)
	done := true
	for _, value := range in.Values {
		s.log.Info().Str("value", value).Msg("Request")
		operation, err := s.cache.Get(ctx, &pbcache.GetRequest{
			SkipCaching: in.SkipCaching,
			Evaluation:  id.String(),
			Context:     in.Context,
			Value:       value,
		})
		if err != nil {
			s.log.Err(err).Msg("Failure to talk to cache")
			return nil, err
		}
		done = done && operation.Done
		log.Println("Unmarshaling")
		if operation.Done {
			response := new(pbcache.GetResponse)
			protoResponse := operation.GetResponse()
			if protoResponse != nil {
				if err := protoResponse.UnmarshalTo(response); err != nil {
					log.Fatalf("1 Cannot unmarhshal result for %s", value)
				}
			} else {
				log.Println("nil response in operation")
			}
		}

		s.log.Info().Str("dependent", operation.Name).Msg("Cache op ID")
		operations[value] = operation
		response := new(pbcache.GetResponse)
		if err := operations[value].GetResponse().UnmarshalTo(response); err != nil {
			log.Fatalf("2 Cannot unmarhshal result for %s", value)
		}

	}

	s.log.Info().Str("OPERATIONS", fmt.Sprintf("%v", operations)).Msg("Dependent Operations")
	err = s.newEvaluation(ctx, id, operations) //uuid.Must(uuid.FromString(operation.Name)))
	if err != nil {
		return nil, err
	}

	values := make(map[string]*pbtypes.TypedValue)
	if done {
		for _, value := range in.Values {
			response := new(pbcache.GetResponse)
			if operation, present := operations[value]; present {
				if err := operation.GetResponse().UnmarshalTo(response); err != nil {
					log.Fatalf("3 Cannot unmarhshal result for %s", value)
				}
				values[value] = response.Value //types.StringScalar("xvalue 1")
			}
		}
	}

	var response *anypb.Any
	response, err = anypb.New(&pbeval.EvalResponse{
		Values: values,
	})
	if err != nil {
		panic(err)
	}

	return &pbasync.Operation{
		Name:   id.String(),
		Done:   done,
		Result: &pbasync.Operation_Response{response},
	}, nil

	var evaluation Evaluation
	//	s.db.Debug().Joins("Operations").First(&evaluation, "ID = ?", id.String())
	s.db.Debug().Preload("Operations").First(&evaluation, "ID = ?", id.String())
	s.log.Info().Str("THING", fmt.Sprintf("%v", evaluation)).Msg("XXXXXXXXXXXXXXXXXX")

	s.log.Info().Str("graph name", in.Context.Actions.Name).Msg("eval")
	a := in.Context.Actions.Actions[0]
	digest, err := actions.ActionDigest(a)
	if err != nil {
		s.log.Err(err).Msg("Error computing digest")
	} else {
		s.log.Info().Str("digest", digest).Msg("action digest")
	}

	// conn, err := client.Connect("eval-cache.eval.svc.cluster.local:50051")
	// if err != nil {
	// 	log.Fatalf("did not connect: %s", err)
	// }
	// defer conn.Close()

	// cache := pbcache.NewCacheServiceClient(conn)

	// evalOperation := Operation{
	// 	EvaluationID: id,
	// 	DependentID:
	// }
	// result := s.db.Create(&evalOperation)
	// // why this condition?
	// if result.Error != nil && result.RowsAffected != 1 {
	// 	s.log.Err(result.Error).Msg("Error occurred while creating a new evaluation")
	// 	return nil, result.Error
	// }

	buildImageConfig := new(pbaction.BuildImageConfig)
	in.Context.Actions.Actions[0].Config.UnmarshalTo(buildImageConfig)
	s.log.Info().Str("Image Name", buildImageConfig.ImageName).Msg("BuildConfig")

	xvalue := types.StringScalar("a nice string value")
	s.log.Info().Str("value", fmt.Sprintf("%v", xvalue)).Msg("VALUE")

	response, err = anypb.New(&pbeval.EvalResponse{
		// for now we hardcode the value name. Will be the values in the request
		Values: map[string]*pbtypes.TypedValue{"result": xvalue},
	})
	if err != nil {
		panic(err)
	}
	return &pbasync.Operation{
		Name:   id.String(),
		Done:   false,
		Result: &pbasync.Operation_Response{response},
	}, nil
}

func (s *serverContext) GetOperation(ctx context.Context, in *pbasync.GetOperationRequest) (*pbasync.Operation, error) {
	var evaluation Evaluation
	//	s.db.Debug().Joins("Operations").First(&evaluation, "ID = ?", id.String())
	s.db.Debug().Preload("Operations").First(&evaluation, "ID = ?", in.Name)

	done := true
	results := make(map[string]*pbtypes.TypedValue)
	for _, o := range evaluation.Operations {
		s.log.Info().Str("OP", o.DependentID.String()).Msg("Cache Operation")
		operation, err := s.cacheOperations.GetOperation(ctx, &pbasync.GetOperationRequest{
			Name: o.DependentID.String(),
		})
		if operation.Done {
			response := new(pbcache.GetResponse)
			if err := operation.GetResponse().UnmarshalTo(response); err != nil {
				log.Fatal("Cannot unmarhshal result")
			}
			results[o.Value] = response.Value
		}
		s.log.Err(err).Msg("GetOperation")
		s.log.Info().Str("operation", fmt.Sprintf("%v", operation)).Msg("GetOperation")

		done = done && operation.Done
	}

	var response *anypb.Any
	response, err := anypb.New(&pbeval.EvalResponse{
		Values: results,
	})
	if err != nil {
		panic(err)
	}

	return &pbasync.Operation{
		Name:   in.Name,
		Done:   done,
		Result: &pbasync.Operation_Response{response},
	}, nil
}

func serviceRegister(server server.Server) func(*grpc.Server) {
	return func(s *grpc.Server) {
		context := serverContext{}
		context.log = server.Logger()
		context.v = server.Config()
		context.db, _ = db.NewDB("engine", &Evaluation{}, &Operation{})

		conn, err := client.Connect("eval-cache.eval.svc.cluster.local:50051")
		if err != nil {
			log.Fatalf("did not connect: %s", err)
		}
		//		defer conn.Close()
		context.cache = pbcache.NewCacheServiceClient(conn)
		context.cacheOperations = pbasync.NewOperationsClient(conn)

		pbeval.RegisterEngineServiceServer(s, &context)
		pbasync.RegisterOperationsServer(s, &context)
		reflection.Register(s)
	}
}

func echoString(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello")
}

func main() {
	go func() {
		http.HandleFunc("/", echoString)

		http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hi")
		})

		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	server := server.Build(port)
	server.RegisterService(serviceRegister(server))
	server.Start()
}
