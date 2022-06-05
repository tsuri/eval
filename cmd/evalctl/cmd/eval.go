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

	"eval/pkg/agraph"
	"eval/pkg/git"
	"eval/pkg/grpc/client"
	pbaction "eval/proto/action"
	pbasync "eval/proto/async_service"
	pbcontext "eval/proto/context"
	pbengine "eval/proto/engine"
	pbtypes "eval/proto/types"

	"github.com/gosuri/uitable"
	"github.com/kyokomi/emoji"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
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

var EvalOperation *pbasync.Operation

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

func getWithDefault(vars map[string]string, varName string, defaultValue string) string {
	if value, present := vars[varName]; present {
		return value
	}
	return defaultValue
}

func createBuildImageConfig(substitutionMap map[string]string) *pbaction.BuildImageConfig {

	// This should probably go to a substitution validation and transformation
	// were we also check that these are valid targets (syntactically) and maybe even that
	// they exists at the requested commit SHA.
	var bazelTargets = []string{"//actions/wrapper:wrapper"}
	if bazelTargetsString, present := substitutionMap["image.build.bazel_targets"]; present {
		bazelTargets = unique(append(bazelTargets, strings.Split(bazelTargetsString, " ")...))
		sort.Strings(bazelTargets)
	}

	bazelTargetsString := getWithDefault(substitutionMap, "image.build.bazel_targets", "//actions/wrapper")
	bazelTargets = unique(append(bazelTargets, strings.Split(bazelTargetsString, " ")...))
	sort.Strings(bazelTargets)

	branch := getWithDefault(substitutionMap, "image.build.commit_point.branch", "main")
	commit := getWithDefault(substitutionMap, "image.build.commit_point.commit_sha", "golden")
	// // TODO by default we should use "golden" in the cluster repo
	// var commit string = "dcc35cc6d501d0b966ff89d589754ca5f31cb429"
	// if commitString, present := substitutionMap["image.build.commit_point.commit_sha"]; present {
	// 	commit = commitString
	// }

	buildImageConfig := pbaction.BuildImageConfig{
		ImageName:    "eval",
		ImageTag:     "latest",
		BaseImage:    "debian:buster",
		BazelTargets: bazelTargets,
		CommitPoint: &pbaction.CommitPoint{
			Branch:    branch,
			CommitSha: commit,
		},
	}

	return &buildImageConfig
}

func ppScalar(v *pbtypes.ScalarValue) string {
	switch x := v.Value.(type) {
	case *pbtypes.ScalarValue_S:
		return x.S
	case *pbtypes.ScalarValue_B:
		if x.B {
			return "true"
		} else {
			return "false"
		}
	case nil:
		return "null"
	default:
		return fmt.Sprintf("unknown type: %v", v)
	}
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
	engine := pbengine.NewEngineServiceClient(conn)

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

	// tags, err := git.GetTags(workspaceTop)
	// if err != nil {
	// 	fmt.Printf("Cannot get tags: %v", err)
	// }
	// fmt.Printf("Tags: %v", tags)

	//	buildImageConfig := createBuildImageConfig(substitutionMap)

	knownActionGraphs := agraph.KnownActionGraphs()

	if bazelTargets, present := substitutionMap["image.build.bazel_targets"]; present {
		config := knownActionGraphs["image"].Actions["build"].Config
		//		fmt.Printf(">>> %T: %v\n", config, config)

		buildConfig := pbaction.BuildImageConfig{}
		if err = config.UnmarshalTo(&buildConfig); err == nil {
			buildConfig.BazelTargets = strings.Split(bazelTargets, " ")

			//			fmt.Printf(">>> %T: %v\n", buildConfig, buildConfig)
			c, err := anypb.New(&buildConfig)
			if err != nil {
				fmt.Println("error")
			}
			//			fmt.Printf(">>> %T: %v", c, c)
			knownActionGraphs["image"].Actions["build"].Config = c
		}
	}

	// TODO here we want to suppoer multiple values. In this case we would have to send in
	// a minimal set of graphs.
	fullValuePath := wantedValues[0]
	graphName := strings.Split(fullValuePath, ".")[0]

	actionGraph, present := knownActionGraphs[graphName]
	if !present {
		log.Fatalf("Unknown action graph: %v", graphName)
	}

	request := pbengine.EvalRequest{
		SkipCaching: skipCaching,
		Context: &pbcontext.Context{
			Actions: actionGraph,
			Substitutions: []*pbcontext.Substitution{
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

	operation, err := engine.Eval(ctx, &request)
	if err != nil {
		log.Fatalf("Error when calling Eval: %s", err)
	}
	EvalOperation = operation

	response := new(pbengine.EvalResponse)
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

	if !operation.Done {
		emoji.Printf("Hold my :beer:\n\n")
	}

	evalOperations := pbasync.NewOperationsClient(conn)
	for !operation.Done {
		operation, err = evalOperations.GetOperation(ctx,
			&pbasync.GetOperationRequest{
				Name: operation.Name,
			})

		time.Sleep(5000 * time.Millisecond)
	}

	if err := operation.GetResponse().UnmarshalTo(response); err != nil {
		log.Fatal("Cannot unmarhshal result")
	}

	emoji.Printf(":magic_wand: Evaluation %s\n\n", operation.Name)

	for k, v := range response.Values {
		// TODO we should chose whether o have k on  aseparate line depending on its length
		//		fmt.Println(k)
		table := uitable.New()
		table.MaxColWidth = 78
		for i, el := range v.Fields {
			if i != 0 {
				k = ""
			}
			table.AddRow(k, el.Name, ppScalar(el.Value))
		}
		fmt.Println(table)
	}
}
