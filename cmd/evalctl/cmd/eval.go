package cmd

import (
	"context"
	"log"
	"path/filepath"
	"strconv"

	"eval/pkg/grpc/client"
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

// evalctl eval/show infra.image --branch --sha --target
// from infra.image -> action graph
// from action-graph -> action config
//
// not here, but in more MP graphs it is nice to have configs that are inherited from level to level
// for instance in ab_comparison.a_training.snippet.sfl one could control the image used for SFL or inherit a
// setting from any of the levels above.

func evalCmdImpl(cmd *cobra.Command, args []string) {
	n, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		log.Fatalf("bad argument: %s", err)
	}

	var conn *grpc.ClientConn
	conn, err = client.NewConnection("engine.eval.net:443",
		filepath.Join(baseDir, caCert),
		filepath.Join(baseDir, clientCert),
		filepath.Join(baseDir, clientKey))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()
	engine := pbEngine.NewEngineServiceClient(conn)

	ctx := client.WithRequesterInfo(context.Background())
	response, err := engine.Eval(ctx, &pbEngine.EvalRequest{Number: n})
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}
	log.Printf("Response from server: %s", response.Number)
}
