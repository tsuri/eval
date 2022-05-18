package cmd

import (
	"log"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
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
	// conn, err := client.NewConnection("engine.eval.net:443",
	// 	filepath.Join(baseDir, caCert),
	// 	filepath.Join(baseDir, clientCert),
	// 	filepath.Join(baseDir, clientKey))
	// if err != nil {
	// 	log.Fatalf("did not connect: %s", err)
	// }
	// defer conn.Close()
	// engine := pbEngine.NewEngineServiceClient(conn)

	// targets, err := cmd.Flags().GetStringArray("target")
	// if err != nil {
	// 	log.Fatalf("Bad argument, target %v", err)
	// }
	// log.Printf("TARGETS: %v\n", targets)

	// branch, err := cmd.Flags().GetString("branch")
	// if err != nil {
	// 	log.Fatalf("Bad argument, branch")
	// }
	// log.Printf("BRANCH: %v\n", branch)

	// commit, err := cmd.Flags().GetString("commit")
	// if err != nil {
	// 	log.Fatalf("Bad argument, commit")
	// }
	// log.Printf("COMMIT: %v\n", commit)

	// repo, err := git.PlainOpen("/home/mav/eval")
	// w, err := repo.Worktree()
	// if err != nil {
	// 	log.Fatalf("Cannot get worktree")
	// }
	// status, err := w.Status()
	// if err != nil {
	// 	log.Fatalf("Cannot get status")
	// }
	// log.Printf("STATUS: %v\n", status)
	// if status.IsClean() {
	// 	log.Println("Clean workspace")
	// } else {
	// 	log.Println("Dirty workspace")
	// }

	// ctx := client.WithRequesterInfo(context.Background())
	// response, err := engine.Build(ctx, &pbEngine.BuildRequest{
	// 	CommitSHA: commit,
	// 	Branch:    branch,
	// 	Target:    targets,
	// })
	// if err != nil {
	// 	log.Fatalf("Error when calling Build: %s", err)
	// }
	// log.Printf("Response from server: %s", response.Response)
	// log.Printf("Built image %s:%s", response.ImageName, response.ImageTag)
}
