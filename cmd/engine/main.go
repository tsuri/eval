package main

import (
	"context"
	"log"

	"eval/pkg/grpc/client"
	"eval/pkg/grpc/server"

	pbbuilder "eval/proto/builder"
	pbeval "eval/proto/engine"
	pbgrunt "eval/proto/grunt"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	port = "0.0.0.0:50051"
)

type serverContext struct {
	log *zerolog.Logger
	v   *viper.Viper
}

func buildImage(in *pbeval.BuildRequest) {
	// ok, a filure to connect here doesn' return error
	conn, err := client.Connect("eval-builder.eval.svc.cluster.local:50051")
	if err != nil {
		log.Fatalf("did not connect")
	}
	defer conn.Close()

	client := pbbuilder.NewBuilderServiceClient(conn)

	requester := pbbuilder.Requester{
		UserName: "USER",
		HostName: "HOSTNAME",
	}

	response, err := client.Build(context.Background(), &pbbuilder.BuildRequest{
		Requester: &requester,
		CommitSHA: in.CommitSHA,
		Branch:    in.Branch,
	})
	if err != nil {
		log.Printf("bad answer from builder")
	} else {
		log.Printf("response %v", response.Response)
	}
}

func grunt(n int64) int64 {
	conn, err := client.Connect("eval-grunt.eval.svc.cluster.local:50051")
	defer conn.Close()

	client := pbgrunt.NewEngineServiceClient(conn)

	response, err := client.Eval(context.Background(), &pbgrunt.EvalRequest{Number: n})
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}

	return response.Number*2 + 1
}

func (s *serverContext) Eval(ctx context.Context, in *pbeval.EvalRequest) (*pbeval.EvalResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Retrieving metadata is failed")
	}
	ruser := "unknown"
	rpass := "unknown"
	user, ok := md["user"]
	if ok {
		ruser = user[0]
	}
	pass, ok := md["pass"]
	if ok {
		rpass = pass[0]
	}
	s.log.Info().Str("user", ruser).Str("pass", rpass).Msg("Metadata")
	return &pbeval.EvalResponse{Number: grunt(in.Number) + 1}, nil
}

func (s *serverContext) Build(ctx context.Context, in *pbeval.BuildRequest) (*pbeval.BuildResponse, error) {
	s.log.Info().Msg("Let's see this one")
	buildImage(in)
	return &pbeval.BuildResponse{Response: "done"}, nil
}

func serviceRegister(server server.Server) func(*grpc.Server) {
	return func(s *grpc.Server) {
		context := serverContext{}
		context.log = server.Logger()
		context.v = server.Config()

		pbeval.RegisterEngineServiceServer(s, &context)
		reflection.Register(s)
	}
}

func main() {
	server := server.Build(port)
	server.RegisterService(serviceRegister(server))
	server.Start()
}
