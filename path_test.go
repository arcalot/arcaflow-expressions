package expressions_test

import (
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
)

func TestPathTree_UnpackDataPaths(t *testing.T) {
	pathTree := expressions.PathTree{
		PathItem: "foo",
		NodeType: expressions.DataRootNode,
		Subtrees: []*expressions.PathTree{
			{
				"bar",
				expressions.AccessNode,
				nil,
			},
			{
				"baz",
				expressions.AccessNode,
				[]*expressions.PathTree{
					{
						0,
						expressions.KeyNode,
						[]*expressions.PathTree{
							{
								"a",
								expressions.AccessNode,
								nil,
							},
						},
					},
				},
			},
		},
	}
	noKeyRequirements := expressions.UnpackRequirements{
		IncludeDataRootPaths:     true,
		IncludeFunctionRootPaths: false,
		StopAtTerminals:          false,
		IncludeKeys:              false,
	}
	withKeyRequirements := expressions.UnpackRequirements{
		IncludeDataRootPaths:     true,
		IncludeFunctionRootPaths: false,
		StopAtTerminals:          false,
		IncludeKeys:              true,
	}
	noDataRootRequirements := expressions.UnpackRequirements{
		IncludeDataRootPaths:     false,
		IncludeFunctionRootPaths: false,
		StopAtTerminals:          false,
		IncludeKeys:              true,
	}
	pathsWithoutKeys := pathTree.Unpack(noKeyRequirements)
	pathsWithKeys := pathTree.Unpack(withKeyRequirements)
	pathsWithoutDataRoot := pathTree.Unpack(noDataRootRequirements)

	assert.Equals(t, len(pathsWithoutKeys), 2)
	assert.Equals(t, len(pathsWithKeys), 2)
	assert.Equals(t, len(pathsWithoutDataRoot), 0)
	assert.Equals(t, pathsWithoutKeys[0].String(), "foo.bar")
	assert.Equals(t, pathsWithKeys[0].String(), "foo.bar")
	assert.Equals(t, pathsWithoutKeys[1].String(), "foo.baz.a")
	assert.Equals(t, pathsWithKeys[1].String(), "foo.baz.0.a")
}

func TestPathTree_UnpackFuncPaths(t *testing.T) {
	pathTree := expressions.PathTree{
		PathItem: "someFunction",
		NodeType: expressions.FunctionNode,
		Subtrees: []*expressions.PathTree{
			{
				"foo",
				expressions.AccessNode,
				nil,
			},
		},
	}

	noDataRootRequirements := expressions.UnpackRequirements{
		IncludeDataRootPaths:     false,
		IncludeFunctionRootPaths: true,
		StopAtTerminals:          false,
		IncludeKeys:              true,
	}
	noFunctionsRootRequirements := expressions.UnpackRequirements{
		IncludeDataRootPaths:     true,
		IncludeFunctionRootPaths: false,
		StopAtTerminals:          false,
		IncludeKeys:              true,
	}

	pathsWithoutDataRoot := pathTree.Unpack(noDataRootRequirements)
	pathsWithoutFunctionsRoot := pathTree.Unpack(noFunctionsRootRequirements)
	assert.Equals(t, len(pathsWithoutFunctionsRoot), 0)
	assert.Equals(t, len(pathsWithoutDataRoot), 1)
	assert.Equals(t, pathsWithoutDataRoot[0].String(), "someFunction.foo")
}
