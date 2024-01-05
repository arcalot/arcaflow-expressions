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
	PathItem any
	Subtrees []*PathTree
}

// Unpack unpacks the path tree into a list of paths.
func (p PathTree) Unpack() []Path {
	if len(p.Subtrees) > 0 {
		var result []Path
		for _, subtree := range p.Subtrees {
			for _, subtreeResult := range subtree.Unpack() {
				currentPathNodes := append([]any{p.PathItem}, subtreeResult...)
				result = append(result, currentPathNodes)
			}
		}
		return result
	}

	return []Path{{p.PathItem}}
}
