package agraph

import (
	"eval/pkg/actions"
	pbagraph "eval/proto/agraph"
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
