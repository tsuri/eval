package graph_test

import (
	"math"
	"testing"

	egraph "eval/pkg/graph"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/testgraph"
)

func directedBuilder(nodes []graph.Node, edges []testgraph.WeightedLine, _, _ float64) (g graph.Graph, n []graph.Node, e []testgraph.Edge, s, a float64, ok bool) {
	//	return nil, nil, nil, math.NaN(), math.NaN(), false

	seen := make(map[graph.Node]bool)
	dg := simple.NewDirectedGraph()

	for _, n := range nodes {
		seen[n] = true
		dg.AddNode(egraph.NewNode(n.id))
	}
	for _, edge := range edges {
		if edge.From().ID() == edge.To().ID() {
			continue
		}
		f := egraph.NewNode(edge.From().ID())
		if f == nil {
			f = edge.From()
		}
		t := egraph.NewNode(edge.To().ID())
		if t == nil {
			t = edge.To()
		}
		ce := simple.Edge{F: *f, T: *t}
		seen[ce.F] = true
		seen[ce.T] = true
		e = append(e, ce)
		dg.SetEdge(ce)
	}
	if len(e) == 0 && len(edges) != 0 {
		return nil, nil, nil, math.NaN(), math.NaN(), false
	}
	if len(seen) != 0 {
		n = make([]graph.Node, 0, len(seen))
	}
	for sn := range seen {
		n = append(n, sn)
	}
	return dg, n, e, math.NaN(), math.NaN(), true
}

func TestDirected(t *testing.T) {
	usesEmpty := true
	reversesEdges := true

	t.Run("EdgeExistence", func(t *testing.T) {
		testgraph.EdgeExistence(t, directedBuilder, true)
	})
	t.Run("NodeExistence", func(t *testing.T) {
		testgraph.NodeExistence(t, directedBuilder)
	})
	t.Run("ReturnAdjacentNodes", func(t *testing.T) {
		testgraph.ReturnAdjacentNodes(t, directedBuilder, usesEmpty, reversesEdges)
	})
	t.Run("ReturnAllEdges", func(t *testing.T) {
		testgraph.ReturnAllEdges(t, directedBuilder, usesEmpty)
	})
	t.Run("ReturnAllNodes", func(t *testing.T) {
		testgraph.ReturnAllNodes(t, directedBuilder, usesEmpty)
	})
	t.Run("ReturnEdgeSlice", func(t *testing.T) {
		testgraph.ReturnEdgeSlice(t, directedBuilder, usesEmpty)
	})
	t.Run("ReturnNodeSlice", func(t *testing.T) {
		testgraph.ReturnNodeSlice(t, directedBuilder, usesEmpty)
	})

	// t.Run("AddNodes", func(t *testing.T) {
	// 	testgraph.AddNodes(t, simple.NewDirectedGraph(), 100)
	// })
	// t.Run("AddArbitraryNodes", func(t *testing.T) {
	// 	testgraph.AddArbitraryNodes(t,
	// 		simple.NewDirectedGraph(),
	// 		testgraph.NewRandomNodes(100, 1, func(id int64) graph.Node { return simple.Node(id) }),
	// 	)
	// })
	// t.Run("RemoveNodes", func(t *testing.T) {
	// 	g := simple.NewDirectedGraph()
	// 	it := testgraph.NewRandomNodes(100, 1, func(id int64) graph.Node { return simple.Node(id) })
	// 	for it.Next() {
	// 		g.AddNode(it.Node())
	// 	}
	// 	it.Reset()
	// 	rnd := rand.New(rand.NewSource(1))
	// 	for it.Next() {
	// 		u := it.Node()
	// 		d := rnd.Intn(5)
	// 		vit := g.Nodes()
	// 		for d >= 0 && vit.Next() {
	// 			v := vit.Node()
	// 			if v.ID() == u.ID() {
	// 				continue
	// 			}
	// 			d--
	// 			g.SetEdge(g.NewEdge(u, v))
	// 		}
	// 	}
	// 	testgraph.RemoveNodes(t, g)
	// })
	// t.Run("AddEdges", func(t *testing.T) {
	// 	testgraph.AddEdges(t, 100,
	// 		simple.NewDirectedGraph(),
	// 		func(id int64) graph.Node { return simple.Node(id) },
	// 		false, // Cannot set self-loops.
	// 		true,  // Can update nodes.
	// 	)
	// })
	// t.Run("NoLoopAddEdges", func(t *testing.T) {
	// 	testgraph.NoLoopAddEdges(t, 100,
	// 		simple.NewDirectedGraph(),
	// 		func(id int64) graph.Node { return simple.Node(id) },
	// 	)
	// })
	// t.Run("RemoveEdges", func(t *testing.T) {
	// 	g := simple.NewDirectedGraph()
	// 	it := testgraph.NewRandomNodes(100, 1, func(id int64) graph.Node { return simple.Node(id) })
	// 	for it.Next() {
	// 		g.AddNode(it.Node())
	// 	}
	// 	it.Reset()
	// 	rnd := rand.New(rand.NewSource(1))
	// 	for it.Next() {
	// 		u := it.Node()
	// 		d := rnd.Intn(5)
	// 		vit := g.Nodes()
	// 		for d >= 0 && vit.Next() {
	// 			v := vit.Node()
	// 			if v.ID() == u.ID() {
	// 				continue
	// 			}
	// 			d--
	// 			g.SetEdge(g.NewEdge(u, v))
	// 		}
	// 	}
	// 	testgraph.RemoveEdges(t, g, g.Edges())
	// })
}
