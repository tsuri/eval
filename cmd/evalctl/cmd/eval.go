package cmd

import (
	"context"
	"log"
	"path/filepath"
	"strconv"

	"eval/pkg/grpc/client"
	pb "eval/proto/engine"

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
	Run: func(cmd *cobra.Command, args []string) {
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
		client := pb.NewEngineServiceClient(conn)
		response, err := client.Eval(context.Background(), &pb.EvalRequest{Number: n})
		if err != nil {
			log.Fatalf("Error when calling Eval: %s", err)
		}
		log.Printf("Response from server: %s", response.Number)

	},
}

func init() {
	rootCmd.AddCommand(evalCmd)
}
