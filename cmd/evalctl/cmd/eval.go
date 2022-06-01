package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"eval/pkg/actions"
	"eval/pkg/git"
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

var workspaceTop string

var evalCmd = &cobra.Command{
	Use:               "eval",
	Short:             "causes the evaluation of a graph",
	Long:              `evalctl controls task graph evaluations.`,
	Args:              cobra.ExactArgs(1),
	Run:               evalCmdImpl,
	ValidArgsFunction: completeTargets,
}

func completeTargets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return []string{"foo", "bar"}, cobra.ShellCompDirectiveNoFileComp
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
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

var completions []string

var substitutionMap = make(map[string]string)

var skipCaching bool

func GenerateCompletions() []string {
	return []string{
		"compare.baseline.train.features.generate",
		"compare.baseline.train.features.generate.images",
		"compare.baseline.train.features.process",
		"compare.baseline.train.features.process.out",
		"compare.baseline.train.features.aggregate",
		"compare.baseline.train.model_train",
		"compare.baseline.analyze",
		"compare.exp.train.features.generate",
		"compare.exp.train.features.process",
		"compare.exp.train.features.aggregate",
		"compare.exp.train.model_train",
		"compare.exp.analyze",
		"compare.summarize",
		"image.build"}
}

func init() {
	rootCmd.AddCommand(evalCmd)
	evalCmd.PersistentFlags().VarP(
		enumflag.New(&executionEngine, "engine", ExecutionEngineIds, enumflag.EnumCaseInsensitive),
		"engine", "e",
		"execution engine; can be 'cloud' or 'local'")
	evalCmd.PersistentFlags().StringToStringVar(&substitutionMap, "with", nil, "some more docs")
	evalCmd.PersistentFlags().BoolVarP(&skipCaching, "no-cache", "x", false, "bypass the cache")
	completions = GenerateCompletions()

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	workspaceTop = dirname + "/eval"
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

func unique(s []string) []string {
	inResult := make(map[string]bool)
	var result []string
	for _, str := range s {
		if _, ok := inResult[str]; !ok {
			inResult[str] = true
			result = append(result, str)
		}
	}
	return result
}

func createBuildImageConfig(substitutionMap map[string]string) *pbAction.BuildImageConfig {

	// This should probably go to a substitution validation and transformation
	// were we also check that these are valid targets (syntactically) and maybe even that
	// they exists at the requested commit SHA.
	var bazelTargets = []string{"//actions/wrapper:wrapper"}
	if bazelTargetsString, present := substitutionMap["image.build.bazel_targets"]; present {
		bazelTargets = unique(append(bazelTargets, strings.Split(bazelTargetsString, " ")...))
		sort.Strings(bazelTargets)
	}

	var branch string = "main"
	if branchString, present := substitutionMap["image.build.commit_point.branch"]; present {
		branch = branchString
	}

	// TODO by default we should use "golden" in the cluster repo
	var commit string = "dcc35cc6d501d0b966ff89d589754ca5f31cb429"
	if commitString, present := substitutionMap["image.build.commit_point.commit_sha"]; present {
		commit = commitString
	}

	buildImageConfig := pbAction.BuildImageConfig{
		ImageName:    "eval",
		ImageTag:     "latest",
		BaseImage:    "debian:buster",
		BazelTargets: bazelTargets,
		CommitPoint: &pbAction.CommitPoint{
			Branch:    branch,
			CommitSha: commit,
		},
	}

	return &buildImageConfig
}

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

	// $WORK
	status, err := git.WorkspaceStatus(workspaceTop)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	wantedValues := args

	// we should do this check only when a commit sha is not passed (for build, so it is tricky; the
	// reason we can do it is that the build action doesn't depend on the commit sha, only the
	// image build. In that case all we need to check that the desired sha is available in repo.
	// But in general, we always need to make this check if the graph contains 'dev' as an image
	// anywhere.
	if !status.IsClean() {
		// TODO: we should wait for an answer when not ok
		emoji.Printf(":pile_of_poo: Do you really want to ignore:\n")
		// TODO: format workspace status nicely
		fmt.Printf("%s\n", status.String())
	}

	// TODO make top of workspace a constant. Better, see is there's a way to derive it
	// automatically
	workspace_branch, workspace_commit_sha, err := git.GetHead(workspaceTop)
	if err != nil {
		log.Fatalf("Cannot get workspace head references")
	}

	buildImageConfig := createBuildImageConfig(substitutionMap)

	actionGraph := actions.AGraphBuildImage(buildImageConfig)
	request := pbEngine.EvalRequest{
		SkipCaching: skipCaching,
		Context: &pbContext.Context{
			Actions: actionGraph,
			Substitutions: []*pbContext.Substitution{
				{
					Variable: "workspace_branch",
					Value:    workspace_branch,
				},
				{
					Variable: "workspace_commit_sha",
					Value:    workspace_commit_sha,
				},
			},
		},
		Values: wantedValues,
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

	// // iterate by sorted keys
	// for _, valueName := range valueNames {
	// 	fmt.Printf("%s: %v", valueName, values[valueName])
	// }

	if !operation.Done {
		emoji.Printf("Hold my :beer:\n\n")
	} else {
		emoji.Printf(":magic_wand: here you are\n\n")
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
