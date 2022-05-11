package agraph

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/testgraph"
)

func actionGraphBuilder(nodes []graph.Node,
	edges []testgraph.WeightedLine,
	_,
	_ float64) (g graph.Graph, n []graph.Node, e []testgraph.Edge, s, a float64, ok bool) {
	ag := NewActionGraph()

	for _, n := range nodes {
		ag.AddNode(n)
	}

	return ag, nodes, []testgraph.Edge{}, math.NaN(), math.NaN(), true
}

func TestActionGraph(t *testing.T) {
	t.Run("EdgeExistence", func(t *testing.T) {
		testgraph.EdgeExistence(t, actionGraphBuilder, true)
	})

	// ag := NewActionGraph()

	// nodeCount := len(ag.Nodes())
	// if nodeCount != 0 {
	// 	t.Fatalf("Expecting empty graph, found %d", nodeCount)
	// }
}
