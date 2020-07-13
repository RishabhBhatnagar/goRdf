package rdfwriter

import (
	"fmt"
	"github.com/RishabhBhatnagar/gordf/rdfloader/parser"
	"github.com/RishabhBhatnagar/gordf/uri"
	"strings"
)

// returns an adjacency list from a list of triples
// Params:
//   triples: might be unordered
// Output:
//    adjList: adjacency list which maps subject to object for each triple
//    recoveryDS: subject to triple mapping that will help retrieve the triples after sorting the Subject: Object pairs.
func getAdjacencyList(triples []*parser.Triple) (adjList map[*parser.Node][]*parser.Node, recoveryDS map[*parser.Node][]*parser.Triple) {
	// triples are analogous to the edges of a graph.
	// For a (Subject, Predicate, Object) triple,
	// it forms a directed edge from Subject to Object
	// Graphically,
	//                          predicate
	//             (Subject) ---------------> (Object)

	// initialising the adjacency list:
	adjList = make(map[*parser.Node][]*parser.Node)
	recoveryDS = make(map[*parser.Node][]*parser.Triple)
	for _, triple := range triples {
		// create a new entry in the adjList if the key is not already seen.
		if adjList[triple.Subject] == nil {
			adjList[triple.Subject] = []*parser.Node{}
			recoveryDS[triple.Subject] = []*parser.Triple{}
		}

		// the key is already seen and we can directly append the child
		adjList[triple.Subject] = append(adjList[triple.Subject], triple.Object)
		recoveryDS[triple.Subject] = append(recoveryDS[triple.Subject], triple)

		// ensure that there is a key entry for all the children.
		if adjList[triple.Object] == nil {
			adjList[triple.Object] = []*parser.Node{}
			recoveryDS[triple.Object] = []*parser.Triple{}
		}
	}
	return adjList, recoveryDS
}

// same as dfs function. Just that after each every neighbor of the node is visited, it is appended in a queue.
// Params:
//     node: Current node to perform dfs on.
//     lastIdx: index where a new node should be added in the resultList
//     visited: if visited[node] is true, we've already serviced the node before.
//     resultList: list of all the nodes after topological sorting.
func topologicalSortHelper(node *parser.Node, lastIndex *int, adjList map[*parser.Node][]*parser.Node, visited *map[*parser.Node]bool, resultList *[]*parser.Node) (err error) {
	if node == nil {
		return
	}

	// checking if the node exist in the graph
	_, exists := adjList[node]
	if !exists {
		return fmt.Errorf("node%v doesn't exist in the graph", *node)
	}
	if (*visited)[node] {
		// this node is already visited.
		// the program enters here when the graph has at least one cycle..
		return
	}

	// marking current node as visited
	(*visited)[node] = true

	// visiting all the neighbors of the node and it's children recursively
	for _, neighbor := range adjList[node] {
		// recurse neighbor only if and only if it is not visited yet.
		if !(*visited)[neighbor] {
			err = topologicalSortHelper(neighbor, lastIndex, adjList, visited, resultList)
			if err != nil {
				return err
			}
		}
	}

	if *lastIndex >= len(adjList) {
		// there is at least one node which is a neighbor of some node
		// whose entry doesn't exist in the adjList
		return fmt.Errorf("found more nodes than the number of keys in the adjacency list")
	}

	// appending from left to right to get a reverse sorted output
	(*resultList)[*lastIndex] = node
	*lastIndex++
	return nil
}

// A wrapper function to initialize the data structures required by the
// topological sort algorithm. It provides an interface to directly get the
// sorted triples without knowing the internal variables required for sorting.
// Note: it sorts in reverse order.
// Params:
//   adjList   : adjacency list: a map with key as the node and value as a
//  			 list of it's neighbor nodes.
// Assumes: all the nodes in the graph are present in the adjList keys.
func topologicalSort(adjList map[*parser.Node][]*parser.Node) ([]*parser.Node, error) {
	// variable declaration
	numberNodes := len(adjList)
	resultList := make([]*parser.Node, numberNodes) //  this will be returned
	visited := make(map[*parser.Node]bool, numberNodes)
	lastIndex := 0

	// iterate through nodes and perform a dfs starting from that node.
	for node := range adjList {
		if !visited[node] {
			err := topologicalSortHelper(node, &lastIndex, adjList, &visited, &resultList)
			if err != nil {
				return resultList, err
			}
		}
	}
	return resultList, nil
}

// Interface for user to provide a list of triples and get the
// sorted one as the output
func TopologicalSortTriples(triples []*parser.Triple) (sortedTriples []*parser.Triple, err error) {
	adjList, recoveryDS := getAdjacencyList(triples)
	sortedNodes, err := topologicalSort(adjList)
	if err != nil {
		return sortedTriples, fmt.Errorf("error sorting the triples: %v", err)
	}

	// initialized a slice
	sortedTriples = make([]*parser.Triple, len(triples))

	i := 0
	for _, subjectNode := range sortedNodes {
		// append all the triples associated with the subjectNode
		for _, triple := range recoveryDS[subjectNode] {
			if i > len(triples) {
				// redundant check. there is no way user might reach here.
				return sortedTriples, fmt.Errorf("overflow error. more triples than expected found after sorting")
			}
			sortedTriples[i] = triple
			i++
		}
	}
	return sortedTriples, nil
}

func DisjointSet(triples []*parser.Triple) map[*parser.Node]*parser.Node {
	parent := make(map[*parser.Node]*parser.Node)
	for _, triple := range triples {
		parent[triple.Object] = triple.Subject
		if _, exists := parent[triple.Subject]; !exists {
			parent[triple.Subject] = nil
		}
	}
	return parent
}

// a schemaDefinition is a dictionary which maps the abbreviation defined in the root tag.
// for example: if the root tag is =>
//      <rdf:RDF
//		    xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"/>
// the schemaDefinition will contain:
//    {"rdf": "http://www.w3.org/1999/02/22-rdf-syntax-ns#"}
// this function will output a reverse map that is:
//    {"http://www.w3.org/1999/02/22-rdf-syntax-ns#": "rdf"}
func invertSchemaDefinition(schemaDefinition map[string]uri.URIRef) map[string]string {
	invertedMap := make(map[string]string)
	for abbreviation := range schemaDefinition {
		_uri := schemaDefinition[abbreviation]
		invertedMap[strings.Trim(_uri.String(), "#")] = abbreviation
	}
	return invertedMap
}

// return true if the target is in the given list
func any(target string, list []string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
