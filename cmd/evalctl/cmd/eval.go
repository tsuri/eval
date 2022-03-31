package cmd

import (
	"context"
	"log"
	"os"
	"os/user"
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
	client := pbEngine.NewEngineServiceClient(conn)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("cannot get hostname: %s", err)
	}
	if os.Geteuid() == 0 {
		log.Fatal("cannot execute as root")
	}
	user, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get username: %s", err)
	}
	requester := pbEngine.Requester{
		UserName: user.Username,
		HostName: hostname,
	}
	response, err := client.Eval(context.Background(), &pbEngine.EvalRequest{Number: n, Requester: &requester})
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}
	log.Printf("Response from server: %s", response.Number)
}
