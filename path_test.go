package expressions_test

import (
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
)

func TestPathTree_Unpack(t *testing.T) {
	pathTree := expressions.PathTree{
		PathItem: "foo",
		Subtrees: []*expressions.PathTree{
			{
				"bar",
				false,
				nil,
			},
			{
				"baz",
				false,
				[]*expressions.PathTree{
					{
						0,
						true,
						nil,
					},
				},
			},
		},
	}
	paths := pathTree.Unpack(false)
	pathsWithExtranous := pathTree.Unpack(true)

	assert.Equals(t, len(paths), 2)
	assert.Equals(t, len(pathsWithExtranous), 2)
	assert.Equals(t, paths[0].String(), "foo.bar")
	assert.Equals(t, pathsWithExtranous[0].String(), "foo.bar")
	assert.Equals(t, paths[1].String(), "foo.baz")
	assert.Equals(t, pathsWithExtranous[1].String(), "foo.baz.0")
}
