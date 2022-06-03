package agraph

import (
	"eval/pkg/actions"
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

func KnownActions() *pbagraph.AGraph {
	actions := pbagraph.AGraph{}

	return &actions
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
