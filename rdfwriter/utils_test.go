package rdfwriter

import (
	"github.com/RishabhBhatnagar/gordf/rdfloader/parser"
	"reflect"
	"testing"
)

func max(n1, n2 int) int {
	if n1 > n2 {
		return n1
	}
	return n2
}

// returns a slice of n blank nodes.
// n > 0
func getNBlankNodes(n int) (blankNodes []*parser.Node) {
	blankNodes = make([]*parser.Node, max(0, n))
	// first blank nodes start with N1
	blankNodeGetter := parser.BlankNodeGetter{}
	for i := 0; i < n; i++ {
		newBlankNode := blankNodeGetter.Get()
		blankNodes[i] = &newBlankNode
	}
	return
}

func Test_getAdjacencyList(t *testing.T) {
	// TestCase 1
	// empty list of triples should have empty map as an output
	adjList, recoveryDS := getAdjacencyList([]*parser.Triple{})
	if len(adjList) > 0 || len(recoveryDS) > 0 {
		t.Errorf("empty input is having non-empty output. Adjacency List: %v, recoveryDS: %v", adjList, recoveryDS)
	}

	// TestCase 2
	// modelling a simple graph depicted as follows:
	//              (N1)
	//       (N0) ---------> (N2)
	//        |
	//   (N3) |
	//        |
	//        v
	//       (N4)
	//
	// Triples for the above graph will be:
	//   1. N0 -> N1 -> N2
	//   2. N0 -> N3 -> N4
	blankNodes := getNBlankNodes(5)
	triples := []*parser.Triple{
		{
			Subject:   blankNodes[0],
			Predicate: blankNodes[1],
			Object:    blankNodes[2],
		}, {
			Subject:   blankNodes[0],
			Predicate: blankNodes[3],
			Object:    blankNodes[4],
		},
	}
	adjList, _ = getAdjacencyList(triples)
	// adjList must have exactly 3 keys (N0, N2, N4)
	if len(adjList) != 3 {
		t.Errorf("adjacency list for the given graph should've only one key. Found %v keys", len(adjList))
	}
	if nChildren := len(adjList[blankNodes[0]]); nChildren != 2 {
		t.Errorf("Node 0 should've exactly 2 children. Found %v children", nChildren)
	}
	// there aren't any neighbors for other nodes.
	for i := 1; i < len(blankNodes); i++ {
		if nChildren := len(adjList[blankNodes[i]]); nChildren > 0 {
			t.Errorf("N%v should have no neighbors. Found %v neighbors", i+1, nChildren)
		}
	}
}

func TestTopologicalSortTriples(t *testing.T) {
	nodes := getNBlankNodes(5)

	// TestCase 1: only a single triple in the list
	// The graph is as follows:
	//        (N1)
	// (N0) --------> (N2)
	triples := []*parser.Triple{
		{nodes[0], nodes[1], nodes[2]},
	}
	// but it doesn't exist in the keys of the map.
	sortedTriples, err := TopologicalSortTriples(triples)
	if err != nil {
		t.Errorf("unexpected parsing a single triple list. Error: %v", err)
	}
	if len(sortedTriples) != len(triples) {
		t.Errorf("sorted triples doesn't have a proper dimension. Expected %v triples, found %v triples", len(triples), len(sortedTriples))
	}

	// TestCase 2: another valid test-case where the input is a cyclic graph.
	/*
					 (N0)
				   /  ^   \
		    (N1) /    |    \ (N2)
				 \    |    /
		           v  |  v
		            (N3)
		    Triples:
				1. N0 -> N1 -> N3
				2. N0 -> N2 -> N3
				3. N3 -> N4 -> N0     // couldn't show N4 in the graph.
	*/
	nodes = getNBlankNodes(5)
	triples = []*parser.Triple{
		{Subject: nodes[0], Predicate: nodes[1], Object: nodes[3]},
		{Subject: nodes[0], Predicate: nodes[2], Object: nodes[3]},
		{Subject: nodes[3], Predicate: nodes[4], Object: nodes[0]},
	}
	sortedTriples, err = TopologicalSortTriples(triples)

	// since we have a cycle here, we can expect two configurations.
	expectedTriples := []*parser.Triple{
		{Subject: nodes[3], Predicate: nodes[4], Object: nodes[0]},
		{Subject: nodes[0], Predicate: nodes[1], Object: nodes[3]},
		{Subject: nodes[0], Predicate: nodes[2], Object: nodes[3]},
	}
	anotherConfig := []*parser.Triple{
		{Subject: nodes[0], Predicate: nodes[1], Object: nodes[3]},
		{Subject: nodes[0], Predicate: nodes[2], Object: nodes[3]},
		{Subject: nodes[3], Predicate: nodes[4], Object: nodes[0]},
	}
	if !reflect.DeepEqual(sortedTriples, expectedTriples) && !reflect.DeepEqual(sortedTriples, anotherConfig) {
		t.Errorf("sorted triples are not in correct order")
	}
}

func Test_topologicalSort(t *testing.T) {
	nodes := getNBlankNodes(5)

	// TestCase 1: invalid adjacency matrix with nodes which not in
	// the keys but exists in the children lists.
	// The graph is as follows:
	//        (N1)
	// (N0) --------> (N2)
	adjList := map[*parser.Node][]*parser.Node{
		nodes[0]: {nodes[2]},
	} // here, nodes[2] is child of nodes[0]
	// but it doesn't exist in the keys of the map.
	_, err := topologicalSort(adjList)
	if err == nil {
		t.Error("expected an error reporting \"extra nodes found\"")
	}

	// TestCase 2: Valid case
	adjList = map[*parser.Node][]*parser.Node{
		nodes[0]: {nodes[2]},
		nodes[2]: {},
	}
	sortedNodes, err := topologicalSort(adjList)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expectedOutput := []*parser.Node{nodes[2], nodes[0]}
	if !reflect.DeepEqual(sortedNodes, expectedOutput) {
		t.Errorf("nodes are not sorted correctly. \nExpected: %v, \nFound: %v", sortedNodes, expectedOutput)
	}
}

func Test_topologicalSortHelper(t *testing.T) {
	// declaring satellite field required by the function
	var lastIndex int
	var adjList map[*parser.Node][]*parser.Node
	var visited map[*parser.Node]bool
	var resultList []*parser.Node

	reinitializeSatellites := func(nNodes int) {
		// nNodes := number of nodes.
		lastIndex = 0
		visited = make(map[*parser.Node]bool, nNodes)
		resultList = make([]*parser.Node, nNodes)
	}

	/*
		Graph that we will be using for all the testcases in this function:
		 It is a simple three staged input with a single source and sink pair.

								 (N1)
		                (N0) ------------> (N2)
						 |				    |
						 |				    |
			 		 (N3)|				    |(N6)
						 |                  |
						 v                  v
						(N4) ------------> (N7)
								(N5)
			Triples that exists in the above graph:
			1. N0 -> N1 -> N2
			2. N2 -> N6 -> N7
			3. N0 -> N3 -> N4
			4. N3 -> N5 -> N7
	*/
	numberNodes := 8
	nodes := getNBlankNodes(numberNodes)
	triples := []*parser.Triple{
		{Subject: nodes[0], Predicate: nodes[1], Object: nodes[2]},
		{Subject: nodes[2], Predicate: nodes[6], Object: nodes[7]},
		{Subject: nodes[0], Predicate: nodes[3], Object: nodes[4]},
		{Subject: nodes[3], Predicate: nodes[5], Object: nodes[7]},
	}
	adjList, _ = getAdjacencyList(triples)

	// TestCase 1: trying to traverse on a node which doesn't exist in the graph.
	// function should raise an error.
	reinitializeSatellites(numberNodes)
	inexistentNode := parser.Node{NodeType: parser.BLANK, ID: "sample node"}
	err := topologicalSortHelper(&inexistentNode, &lastIndex, adjList, &visited, &resultList)
	if err == nil {
		t.Errorf("inexistent node should've raised an error")
	}

	// TestCase 2: traversing node with no children
	reinitializeSatellites(numberNodes)
	err = topologicalSortHelper(nodes[7], &lastIndex, adjList, &visited, &resultList)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// if a node without a child is traversed, the number of nodes added to the
	// resultList should be exactly 1. That is the node itself.
	if lastIndex != 1 {
		t.Errorf("Expected exactly 1 node in the result list. Found %v nodes", lastIndex)
	}

	// TestCase 3: traversing a node with 1 child
	// if we are traversing node N2, N7 is the only child.
	reinitializeSatellites(numberNodes)
	_ = topologicalSortHelper(nodes[2], &lastIndex, adjList, &visited, &resultList)
	if lastIndex != 2 {
		t.Errorf("Expected exactly 1 node in the result list. Found %v nodes", lastIndex)
	}
	// since N7 is the child of N2,
	// resultList must  have N7 before N2.
	if resultList[0] != nodes[7] || resultList[1] != nodes[2] {
		t.Error("resultList is not set properly")
	}

	// TestCase 4: Final Case when all the nodes will be traversed.
	reinitializeSatellites(numberNodes)
	_ = topologicalSortHelper(nodes[0], &lastIndex, adjList, &visited, &resultList)
	if lastIndex != 4 {
		t.Errorf("after parsing all nodes, resultList must've 4 nodes. Found %v nodes", lastIndex)
	}
	// after traversing all the nodes, (N0) should be the last node
	// to be added to the list and (N7) should be the first node.
	if resultList[0] != nodes[7] || resultList[lastIndex-1] != nodes[0] {
		t.Error("order of resultList if not correct")
	}
}
