package cmd

import (
	"context"
	"errors"
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
	pbagraph "eval/proto/agraph"
	pbasync "eval/proto/async_service"
	pbcontext "eval/proto/context"
	pbengine "eval/proto/engine"
	pbtypes "eval/proto/types"

	"github.com/alexeyco/simpletable"
	"github.com/gosuri/uitable"
	"github.com/kyokomi/emoji"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
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

var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "connect to a previous evaluation",
	Long:  `you can get the results of a prior evaluation, waiting for it if needed`,
	Args:  cobra.ExactArgs(1),
	Run:   attachCmdImpl,
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

func dumpFields(object proto.Message) {
	fmt.Println()
	fields := object.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fmt.Printf("Field: %v\n", fields.Get(i))
	}

	fmt.Println("--")
	m := object.ProtoReflect()
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fmt.Printf("Field: %v\n\tValue: %v\n", fd.FullName, v)
		opts := fd.Options().(*descriptorpb.FieldOptions)
		fmt.Printf("Options: %v\n", opts)
		return true
	})
	fmt.Println("^^")
}

// func decode(a *anypb.Any, o *proto.Message) error {
// 	err := a.UnmarshalTo(*o)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func GenerateCompletions() []string {
	dumpFields(&pbaction.CommitPoint{})
	dumpFields(&pbaction.BuildImageConfig{})
	dumpFields(&pbaction.Action{})
	fmt.Println("------------------")
	dumpFields(agraph.KnownActionGraphs()["image"].Actions["build"].Config)
	// bc := pbaction.BuildImageConfig{}
	// if err := decode(agraph.KnownActionGraphs()["image"].Actions["build"].Config, &bc); err != nil {
	// 	fmt.Printf("Cannot decode: %v", err)
	// }
	fmt.Println("------------------")

	// fields := (&pbaction.BuildImageConfig{}).ProtoReflect().Descriptor().Fields()
	// for i := 0; i < fields.Len(); i++ {
	// 	fmt.Printf("Field: %v\n", fields.Get(i))
	// }

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
	rootCmd.AddCommand(attachCmd)
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

func waitCompletion(ctx context.Context, operationName string) {
	var conn *grpc.ClientConn
	conn, err := client.NewConnection("engine.eval.net:443",
		filepath.Join(baseDir, caCert),
		filepath.Join(baseDir, clientCert),
		filepath.Join(baseDir, clientKey))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	evalOperations := pbasync.NewOperationsClient(conn)
	operation, err := evalOperations.GetOperation(ctx,
		&pbasync.GetOperationRequest{
			Name: operationName,
		})
	if err != nil {
		fmt.Printf("get operation error: %v", err)
	}

	first := true
	for !operation.Done {
		if first {
			first = false
			emoji.Printf("Hold my :beer:\n\n")
		}
		operation, err = evalOperations.GetOperation(ctx,
			&pbasync.GetOperationRequest{
				Name: operationName,
			})

		time.Sleep(5000 * time.Millisecond)
	}

	response := new(pbengine.EvalResponse)
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

	fmt.Println("-------------------------------")
	atable := simpletable.New()
	atable.SetStyle(simpletable.StyleCompactLite)
	atable.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignCenter, Text: "Field"},
			{Align: simpletable.AlignCenter, Text: "Value"},
		},
	}
	fmt.Println(atable.String())
	fmt.Println("-------------------------------")
}

func attachCmdImpl(cmd *cobra.Command, args []string) {
	fmt.Printf("Attaching to %s", args[0])
}

func valueGraphBuckets(values []string) map[string][]string {
	uniqueValues := unique(values)
	result := make(map[string][]string)

	for _, v := range uniqueValues {
		vl := strings.Split(v, ".")
		graphName := vl[0]
		if valueList, present := result[graphName]; present {
			valueList = append(valueList, v)
		} else {
			result[graphName] = []string{v}
		}
	}
	fmt.Printf("RESULT: %v\n", result)
	return result
}

type valueRequest struct {
	g  *pbagraph.AGraph
	vl []string
}

func actionGraphs(values []string, substitutionMap map[string]string) ([]valueRequest, error) {
	availableGraphs := agraph.KnownActionGraphs()

	buckets := valueGraphBuckets(values)
	fmt.Printf("buckets: %v\n", buckets)
	result := make([]valueRequest, 0, len(buckets))
	for k, vl := range buckets {
		if g, present := availableGraphs[k]; present {
			result = append(result, valueRequest{g: g, vl: vl})
		} else {
			return nil, errors.New(fmt.Sprintf("Invalid value, graph %s unkown", k))
		}
	}
	fmt.Printf("REQUESTS: %v\n", result)
	return result, nil
}

func evalCmdImpl(cmd *cobra.Command, args []string) {
	evalStart := time.Now()

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

	resolvedGraphs, err := actionGraphs(wantedValues, substitutionMap)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("------------------------")
	for _, valueRequest := range resolvedGraphs {
		agraph.Dump(valueRequest.g)
	}
	fmt.Println("------------------------")
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

	// we should get tags from the remote, not from the local repo. Somebody might have
	// changed them. For the demo this is ok.
	tags, err := git.GetTags(workspaceTop)
	if err != nil {
		fmt.Printf("Cannot get tags: %v", err)
	}
	for tag, commit := range tags {
		fmt.Printf("Tag: %s Branch: %s Sha: %s\n", tag, commit.Branch, commit.Hash)
	}
	fmt.Println("")

	//	buildImageConfig := createBuildImageConfig(substitutionMap)

	fmt.Printf("Substirutions:\n")
	for k, v := range substitutionMap {
		fmt.Printf("%s: %s\n", k, v)
	}

	knownActionGraphs := agraph.KnownActionGraphs()

	fmt.Printf("TODO: NEED TO APPLY SUBSTITUTIONS (SHA AND BRANCH)\n")

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

	if commitSha, present := substitutionMap["image.build.commit_sha"]; present {
		config := knownActionGraphs["image"].Actions["build"].Config
		//		fmt.Printf(">>> %T: %v\n", config, config)

		buildConfig := pbaction.BuildImageConfig{}
		if err = config.UnmarshalTo(&buildConfig); err == nil {
			buildConfig.CommitPoint.CommitSha = commitSha
			c, err := anypb.New(&buildConfig)
			if err != nil {
				fmt.Println("error")
			}
			knownActionGraphs["image"].Actions["build"].Config = c
		}
	}

	// TODO here we want to support multiple values. In this case we would have to send in
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
					// later will be something like:
					// Variable: "...branch=dev",
					// e.g. a variable branch with a value 'dev' will be replaced with
					// workspace branch
					Variable:     "...branch",
					ReplaceValue: "dev",
					Value:        workspace_branch,
				},
				{
					// later will be something like:
					// Variable: "...commit_sha=dev",
					// e.g. a variable commit_sha with a value 'dev' will be replaced with
					// workspace branch
					Variable:     "...commit_sha",
					ReplaceValue: "dev",
					Value:        workspace_commit_sha,
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

	//	waitCompletion(ctx, operation.Name)
	//	return

	evalOperations := pbasync.NewOperationsClient(conn)

	first := true
	for !operation.Done {
		if first {
			first = false
			emoji.Printf("Hold my :beer:\n\n")
		}
		operation, err = evalOperations.GetOperation(ctx,
			&pbasync.GetOperationRequest{
				Name: operation.Name,
			})

		time.Sleep(5000 * time.Millisecond)
	}

	if err := operation.GetResponse().UnmarshalTo(response); err != nil {
		log.Fatal("Cannot unmarhshal result")
	}

	evalDuration := time.Now().Sub(evalStart)
	emoji.Printf(":magic_wand: Evaluation %s (%s)\n\n", operation.Name, evalDuration)

	for k, v := range response.Values {
		// TODO we should chose whether o have k on  aseparate line depending on its length
		//		fmt.Println(k)
		table := uitable.New()
		table.MaxColWidth = 78
		table.AddRow("compare.baseline.train.features.generate.images")
		table.AddRow("")
		for i, el := range v.Fields {
			if i != 0 {
				k = ""
			}
			table.AddRow(k, el.Name, ppScalar(el.Value))
		}
		fmt.Println(table)
	}
}
