package graph

import gonumGraph "gonum.org/v1/gonum/graph/simple"

type EGraph struct {
	gonumGraph.DirectedGraph
}

type ENode struct {
	id   int64
	name string
}

func (n ENode) ID() int64 {
	return n.id
}

func NewNode(id int64) ENode {
	return ENode{
		id:   id,
		name: "",
	}
}

// type EEdge struct {
// 	//	gonumGraph.simple.Node
// 	F ENode
// 	T ENode
// }

// func (e EEdge) To() gonumGraph.Node {
// 	return e.T
// }

// func (e EEdge) From() gonumGraph.Node {
// 	return e.F
// }

func NewEGraph() EGraph {
	return EGraph{}
}
