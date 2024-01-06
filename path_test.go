package expressions_test

import (
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
)

func TestPathTree_Unpack(t *testing.T) {
	pathTree := expressions.PathTree{
		PathItem:   "foo",
		Extraneous: []any{"a"},
		Subtrees: []*expressions.PathTree{
			{
				"bar",
				[]any{},
				nil,
			},
			{
				"baz",
				[]any{0},
				nil,
			},
		},
	}
	paths := pathTree.Unpack(false)
	pathsWithExtranous := pathTree.Unpack(true)

	assert.Equals(t, len(paths), 2)
	assert.Equals(t, len(pathsWithExtranous), 2)
	assert.Equals(t, paths[0].String(), "foo.bar")
	assert.Equals(t, pathsWithExtranous[0].String(), "foo.a.bar")
	assert.Equals(t, paths[1].String(), "foo.baz")
	assert.Equals(t, pathsWithExtranous[1].String(), "foo.a.baz.0")
}
