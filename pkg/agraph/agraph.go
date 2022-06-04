package agraph

import (
	"eval/pkg/actions"
	pbaction "eval/proto/action"
	pbagraph "eval/proto/agraph"
	"fmt"
	"log"
)

//
func EssentialActions(agraph *pbagraph.AGraph, value string) *pbagraph.AGraph {
	return agraph
}

func Execute(agraph *pbagraph.AGraph) {
	// we shoud topological sort here (not really for runner, but doesn't hurt either)
	for _, a := range agraph.Actions {
		log.Printf("kind: %s", a.Kind)
		digest, err := actions.ActionDigest(a)
		if err != nil {
			log.Printf("Error in digest: %v", err)
		}
		log.Printf("DIGEST: %v", digest)
	}
}

func ImageGraph() *pbagraph.AGraph {
	return &pbagraph.AGraph{
		Name: "image", // maybe name is not needed
		Actions: map[string]*pbaction.Action{
			"build": actions.NewBuildImageAction(),
		},
	}
}

func KnownActionGraphs() map[string]*pbagraph.AGraph {
	actionGraphs := make(map[string]*pbagraph.AGraph)

	actionGraphs["image"] = ImageGraph()

	return actionGraphs
}

func Dump(ag *pbagraph.AGraph) {
	for k, v := range ag.Actions {
		inputs := ""
		for _, i := range v.Inputs {
			inputs += " " + i.Name
		}
		fmt.Printf("\n%s: %s\n", k, inputs)
		for _, o := range v.Outputs {
			fmt.Printf("    %s\n", o.Name)
		}
	}
	fmt.Printf("\n\n")
}
