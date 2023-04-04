package expressions_test

import (
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
)

func TestPathTree_Unpack(t *testing.T) {
	paths := expressions.PathTree{
		"foo",
		[]*expressions.PathTree{
			{
				"bar",
				nil,
			},
			{
				"baz",
				nil,
			},
		},
	}.Unpack()

	assert.Equals(
		t,
		len(paths),
		2,
	)
	assert.Equals(
		t,
		paths[0].String(),
		"foo.bar",
	)
	assert.Equals(
		t,
		paths[1].String(),
		"foo.baz",
	)
}
