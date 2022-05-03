package cmd

import (
	"context"
	"eval/pkg/grpc/client"
	pbEngine "eval/proto/engine"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/go-git/go-git/v5"
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
	Args:  cobra.ExactArgs(0),
	Run:   buildCmdImpl,
}

func init() {
	repo, err := git.PlainOpen("/home/mav/eval")
	if err != nil {
		log.Fatalf("Not in  git workspace")
	}
	ref, err := repo.Head()
	if err != nil {
		log.Fatalf("Cannot get HEAD")
	}

	buildCmd.PersistentFlags().StringP("commit", "c", ref.Hash().String(), "Commit SHA used for building targets")
	buildCmd.PersistentFlags().StringP("branch", "b", ref.Name().String(), "Branch used for building targets")
	buildCmd.PersistentFlags().StringArrayP("target", "t", []string{}, "Bazel targets to be included in the image")
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

	targets, err := cmd.Flags().GetStringArray("target")
	if err != nil {
		log.Fatalf("Bad argument, target %v", err)
	}
	log.Printf("TARGETS: %v\n", targets)

	branch, err := cmd.Flags().GetString("branch")
	if err != nil {
		log.Fatalf("Bad argument, branch")
	}
	log.Printf("BRANCH: %v\n", branch)

	commit, err := cmd.Flags().GetString("commit")
	if err != nil {
		log.Fatalf("Bad argument, commit")
	}
	log.Printf("COMMIT: %v\n", commit)

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

	repo, err := git.PlainOpen("/home/mav/eval")
	// if err != nil {
	// 	log.Fatalf("Not in  git workspace")
	// }
	// ref, err := repo.Head()
	// if err != nil {
	// 	log.Fatalf("Cannot get HEAD")
	// }
	// log.Printf("REF: %v\n", ref)
	// log.Println("REF hash: ", ref.Hash())
	// log.Println("REF name: ", ref.Name())
	// log.Println("REF target: ", ref.Target())
	w, err := repo.Worktree()
	if err != nil {
		log.Fatalf("Cnnot get worktree")
	}
	status, err := w.Status()
	if err != nil {
		log.Fatalf("Cannot get status")
	}
	log.Printf("STATUS: %v\n", status)
	if status.IsClean() {
		log.Println("Clean workspace")
	} else {
		log.Println("Dirty workspace")
	}

	requester := pbEngine.Requester{
		UserName: user.Username,
		HostName: hostname,
	}
	response, err := client.Build(context.Background(), &pbEngine.BuildRequest{
		Requester: &requester,
		CommitSHA: commit,
		Branch:    branch,
		Target:    targets,
	})
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}
	log.Printf("Response from server: %s", response.Response)

}
