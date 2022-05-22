package agraph

import (
	pbagraph "eval/proto/agraph"
	"log"
)

//
func EssentialActions(agraph *pbagraph.AGraph, value string) *pbagraph.AGraph {
	return agraph
}

func Execute(agraph *pbagraph.AGraph) {
	// we shoud topological sort here (not really for runner, but doesn't hurt either)
	for _, action := range agraph.Actions {
		log.Printf("kind: %s", action.Kind)
	}
}
