package expressions

import (
	"fmt"
	"strings"
)

// Path describes the path needed to take to reach an item. Items can either be strings or integers.
type Path []any

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
	// Extra values that do not contribute to the validated path, like within `any` values, or map indexes
	IsExtraneous bool
	Subtrees     []*PathTree
}

// Unpack unpacks the path tree into a list of paths.
func (p PathTree) Unpack(includeExtraneous bool) []Path {
	if !includeExtraneous && p.IsExtraneous {
		// This one is extraneous, so exit without any paths.
		return []Path{}
	}
	var result []Path

	for _, subtree := range p.Subtrees {
		for _, subtreeResult := range subtree.Unpack(includeExtraneous) {
			// First, this path item
			currentPathNodes := []any{p.PathItem}
			// Second, add the subtrees
			currentPathNodes = append(currentPathNodes, subtreeResult...)
			result = append(result, currentPathNodes)
		}
	}
	if len(result) == 0 { // Happens when there are only extraneous subtrees
		result = append(result, []any{p.PathItem})
	}
	return result
}
