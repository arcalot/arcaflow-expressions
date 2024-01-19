package expressions

import (
	"fmt"
	"strings"
)

// Path describes the path needed to take to reach an item. Items can either be strings or integers.
type Path []any

type PathNodeType string

const (
	// DataRootNode is a node describing the root data path '$'
	// The valid PathItem for this node type is '$'
	DataRootNode PathNodeType = "root"
	// FunctionNode is a node describing a function access. It is a type of root node.
	// The valid PathItems for this node type are function name strings.
	FunctionNode PathNodeType = "function"
	// AccessNode is accessing a field within the object preceding it.
	// Valid PathItems for this node type are object field names strings.
	AccessNode PathNodeType = "access"
	// KeyNode means that it's a node that describes the access to the node preceding it. It could be describing
	// an integer literal index, or a string literal map key. This info is useful in the cases where the index or
	// map key can change the dependency.
	// Valid PathItems for this node type are integers for indexes, or strings or integers for map accesses.
	KeyNode PathNodeType = "key"
	// PastTerminalNode refers to a value access that is past a terminal value. An any type is a terminal value.
	// This node type means it's trying to access a value within an any type.
	// Valid PathItems for this node type are object field names strings.
	PastTerminalNode PathNodeType = "past-terminal"
)

// String returns the dot-concatenated string version of the path as an Arcaflow-expression.
func (p Path) String() string {
	items := make([]string, len(p))
	for i, item := range p {
		items[i] = fmt.Sprintf("%v", item)
	}

	return strings.Join(items, ".")
}

// PathTree holds multiple paths in a branching fashion.
type PathTree struct {
	// The value at the part of the tree
	PathItem any
	NodeType PathNodeType
	Subtrees []*PathTree
}

// Unpack unpacks the path tree into a list of paths.
func (p PathTree) Unpack(requirements UnpackRequirements) []Path {
	if requirements.shouldStop(p.NodeType) {
		return []Path{}
	}
	var result []Path

	for _, subtree := range p.Subtrees {
		for _, subtreeResult := range subtree.Unpack(requirements) {
			currentPathNodes := make([]any, 0)
			// First, this path item, if not skipping it
			if !requirements.shouldSkip(p.NodeType) {
				currentPathNodes = append(currentPathNodes, p.PathItem)
			}
			// Second, add the subtrees
			currentPathNodes = append(currentPathNodes, subtreeResult...)
			result = append(result, currentPathNodes)
		}
	}
	
	// An empty result happens when either there are zero subtrees, or the
	// subtrees are excluded based on the current requirements.
	// Return the current path if the current path node should be an included
	// leaf node. Skipped nodes should not.
	if len(result) == 0 && !requirements.shouldSkip(p.NodeType) {
		return []Path{[]any{p.PathItem}}
	}

	return result
}

type UnpackRequirements struct {
	ExcludeDataRootPaths     bool // Exclude paths that start at data root
	ExcludeFunctionRootPaths bool // Exclude paths that start at a function
	StopAtTerminals          bool // Whether to stop at terminals (any types are terminals).
	IncludeKeys              bool // Whether to include the keys in the path. // Example, the 0 in `$ -> list -> 0 -> a`
}

func (r *UnpackRequirements) shouldStop(nodeType PathNodeType) bool {
	switch nodeType {
	case DataRootNode:
		return r.ExcludeDataRootPaths
	case FunctionNode:
		return r.ExcludeFunctionRootPaths
	case PastTerminalNode:
		return r.StopAtTerminals
	case AccessNode, KeyNode:
		return false
	default:
		panic(fmt.Errorf("node type %q missing in shouldStop in path", nodeType))
	}
}

func (r *UnpackRequirements) shouldSkip(nodeType PathNodeType) bool {
	return nodeType == KeyNode && !r.IncludeKeys
}
