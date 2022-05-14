package cmd

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"eval/pkg/actions"
	"eval/pkg/grpc/client"
	pbAction "eval/proto/action"
	pbAsyncService "eval/proto/async_service"
	pbContext "eval/proto/context"
	pbEngine "eval/proto/engine"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const (
	baseDir    = "/data/eval/certificates"
	caCert     = "evalCA.crt"
	clientCert = "evalctl.crt"
	clientKey  = "evalctl.key"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "causes the evaluation of a graph",
	Long:  `Something deeper here.`,
	Args:  cobra.ExactArgs(1),
	Run:   evalCmdImpl,
}

func init() {
	rootCmd.AddCommand(evalCmd)
}

func evalBuildImage(branch string, commitSHA string) {
}

// evalctl eval/show infra.image --branch --sha --target
// from infra.image -> action graph
// from action-graph -> action config
//
// not here, but in more MP graphs it is nice to have configs that are inherited from level to level
// for instance in ab_comparison.a_training.snippet.sfl one could control the image used for SFL or inherit a
// setting from any of the levels above.

func evalCmdImpl(cmd *cobra.Command, args []string) {
	var conn *grpc.ClientConn
	conn, err := client.NewConnection("engine.eval.net:443",
		filepath.Join(baseDir, caCert),
		filepath.Join(baseDir, clientCert),
		filepath.Join(baseDir, clientKey))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()
	engine := pbEngine.NewEngineServiceClient(conn)

	ctx := client.WithRequesterInfo(context.Background())

	buildImageConfig := pbAction.BuildImageConfig{
		ImageName:    "eval",
		ImageTag:     "latest",
		BaseImage:    "debian:buster",
		BazelTargets: []string{"//test:test", "//test:runner"},
		CommitPoint: &pbAction.CommitPoint{
			Branch:    "main",
			CommitSha: "c32b7e6cbac753c54ffa8c78687feae7eae1711c",
		},
	}

	actionGraph := actions.AGraphBuildImage(&buildImageConfig)
	request := pbEngine.EvalRequest{
		Context: &pbContext.Context{
			Actions: actionGraph,
		},
		Values: []string{"image.build"},
	}

	log.Printf("Launching evaluation")
	operation, err := engine.Eval(ctx, &request)
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}

	evalOperations := pbAsyncService.NewOperationsClient(conn)
	for !operation.Done {
		log.Printf("Waiting...\n")
		operation, err = evalOperations.GetOperation(ctx, &pbAsyncService.GetOperationRequest{Name: "foo"})

		time.Sleep(500 * time.Millisecond)
	}

	response := new(pbEngine.EvalResponse)
	if err := operation.GetResponse().UnmarshalTo(response); err != nil {
		log.Fatal("Cannot unmarhshal result")
	}
	log.Printf("Response from server: %s", response.Number)
}
