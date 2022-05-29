package cmd

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"time"

	"eval/pkg/actions"
	"eval/pkg/grpc/client"
	pbAction "eval/proto/action"
	pbAsyncService "eval/proto/async_service"
	pbContext "eval/proto/context"
	pbEngine "eval/proto/engine"

	"github.com/kyokomi/emoji"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
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

type ExecutionEngine enumflag.Flag

const (
	Cloud ExecutionEngine = iota
	Local
)

var ExecutionEngineIds = map[ExecutionEngine][]string{
	Cloud: {"cloud"},
	Local: {"local"},
}

var executionEngine ExecutionEngine

func init() {
	rootCmd.AddCommand(evalCmd)
	rootCmd.PersistentFlags().VarP(
		enumflag.New(&executionEngine, "engine", ExecutionEngineIds, enumflag.EnumCaseInsensitive),
		"engine", "e",
		"execution engine; can be 'cloud' or 'local'")

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

var EvalOperation *pbAsyncService.Operation

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
		BazelTargets: []string{"//actions/wrapper:wrapper", "//actions/generate:generate"},
		//		BazelTargets: []string{"//test:runner"},
		CommitPoint: &pbAction.CommitPoint{
			Branch:    "main",
			CommitSha: "5eb87be8f3f975ea295cf1bd8fa2d3828314e493",
			//CommitSha: "af1b634eb10777ff1b2c4aded960ea1645d49653",
		},
	}

	actionGraph := actions.AGraphBuildImage(&buildImageConfig)
	request := pbEngine.EvalRequest{
		Context: &pbContext.Context{
			Actions: actionGraph,
		},
		Values: []string{"image.build"},
	}

	//	log.Printf("Launching evaluation")

	operation, err := engine.Eval(ctx, &request)
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}
	EvalOperation = operation

	//	log.Println("Got answer")

	response := new(pbEngine.EvalResponse)
	if err := operation.GetResponse().UnmarshalTo(response); err != nil {
		log.Fatal("Cannot unmarhshal result")
	}

	values := response.Values //.Fields[0].Value.GetS()
	// create slice and store keys
	valueNames := make([]string, 0, len(values))
	for v := range values {
		valueNames = append(valueNames, v)
	}

	// sort the slice by keys
	sort.Strings(valueNames)

	// iterate by sorted keys
	for _, valueName := range valueNames {
		fmt.Printf("%s: %v", valueName, values[valueName])
	}

	if !operation.Done {
		emoji.Printf("Hold my :beer:\n\n")
	}

	evalOperations := pbAsyncService.NewOperationsClient(conn)
	for !operation.Done {
		operation, err = evalOperations.GetOperation(ctx,
			&pbAsyncService.GetOperationRequest{
				Name: operation.Name,
			})

		time.Sleep(5000 * time.Millisecond)
	}

	if err := operation.GetResponse().UnmarshalTo(response); err != nil {
		log.Fatal("Cannot unmarhshal result")
	}

	for k, v := range response.Values {
		fmt.Printf("%s: %v", k, v)
	}
}
