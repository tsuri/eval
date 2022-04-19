package cmd

import (
	"context"
	"eval/pkg/grpc/client"
	pbEngine "eval/proto/engine"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Image a docker image",
	Long:  "Image",
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a docker image",
	Long:  "Build",
	Args:  cobra.ExactArgs(1),
	Run:   buildCmdImpl,
}

func init() {
	imageCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(imageCmd)
}

func buildCmdImpl(cmd *cobra.Command, args []string) {
	var conn *grpc.ClientConn
	conn, err := client.NewConnection("engine.eval.net:443",
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
	response, err := client.Build(context.Background(), &pbEngine.BuildRequest{
		Requester: &requester,
		CommitSHA: "something",
		Branch:    "branch",
	})
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}
	log.Printf("Response from server: %s", response.Response)

}
