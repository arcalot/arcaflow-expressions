package expressions_test

import (
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
)

func pathStrExtractor(value expressions.Path) string {
	return value.String()
}

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
		ExcludeDataRootPaths:     false,
		ExcludeFunctionRootPaths: true,
		IncludeKeys:              false,
	}
	withKeyRequirements := expressions.UnpackRequirements{
		ExcludeDataRootPaths:     false,
		ExcludeFunctionRootPaths: true,
		IncludeKeys:              true,
	}
	noDataRootRequirements := expressions.UnpackRequirements{
		ExcludeDataRootPaths:     true,
		ExcludeFunctionRootPaths: true,
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
		ExcludeDataRootPaths:     true,
		ExcludeFunctionRootPaths: false,
	}
	noFunctionsRootRequirements := expressions.UnpackRequirements{
		ExcludeDataRootPaths:     false,
		ExcludeFunctionRootPaths: true,
	}

	pathsWithoutDataRoot := pathTree.Unpack(noDataRootRequirements)
	pathsWithoutFunctionsRoot := pathTree.Unpack(noFunctionsRootRequirements)
	assert.Equals(t, len(pathsWithoutFunctionsRoot), 0)
	assert.Equals(t, len(pathsWithoutDataRoot), 1)
	assert.Equals(t, pathsWithoutDataRoot[0].String(), "someFunction.foo")
}

func TestPathTree_UnpackNoSubtrees(t *testing.T) {
	funcPathTree := expressions.PathTree{
		PathItem: "someFunction",
		NodeType: expressions.FunctionNode,
		Subtrees: []*expressions.PathTree{},
	}
	rootPathTree := expressions.PathTree{
		PathItem: "$",
		NodeType: expressions.DataRootNode,
		Subtrees: []*expressions.PathTree{},
	}
	nonRootPathTree := expressions.PathTree{
		PathItem: "a",
		NodeType: expressions.AccessNode,
		Subtrees: []*expressions.PathTree{},
	}

	defaultRequirements := expressions.UnpackRequirements{}

	funcTreePaths := funcPathTree.Unpack(defaultRequirements)
	assert.Equals(t, len(funcTreePaths), 1)
	assert.Equals(t, funcTreePaths[0].String(), "someFunction")
	rootTreePaths := rootPathTree.Unpack(defaultRequirements)
	assert.Equals(t, len(rootTreePaths), 1)
	assert.Equals(t, rootTreePaths[0].String(), "$")
	nonRootTreePaths := nonRootPathTree.Unpack(defaultRequirements)
	assert.Equals(t, len(nonRootTreePaths), 1)
	assert.Equals(t, nonRootTreePaths[0].String(), "a")
}

func TestPathTree_UnpackLeafKeys(t *testing.T) {
	// This test keeps it simple, with a key in the end (leaf), so we ensure that it's
	// kept only when IncludeTrees is true.
	pathTree := expressions.PathTree{
		PathItem: "$",
		NodeType: expressions.DataRootNode,
		Subtrees: []*expressions.PathTree{
			{
				PathItem: "a",
				NodeType: expressions.KeyNode,
				Subtrees: nil,
			},
		},
	}

	defaultRequirements := expressions.UnpackRequirements{}
	withKeysRequirements := expressions.UnpackRequirements{
		IncludeKeys: true,
	}

	noKeyTreePaths := pathTree.Unpack(defaultRequirements)
	assert.Equals(t, len(noKeyTreePaths), 1)
	assert.Equals(t, noKeyTreePaths[0].String(), "$")
	withKeyTreePaths := pathTree.Unpack(withKeysRequirements)
	assert.Equals(t, len(withKeyTreePaths), 1)
	assert.Equals(t, withKeyTreePaths[0].String(), "$.a")
}

func TestPathTree_UnpackMiddleKeys(t *testing.T) {
	// This tests when a key in the middle level of the tree, with a non-key leaf after it.
	pathTree := expressions.PathTree{
		PathItem: "$",
		NodeType: expressions.DataRootNode,
		Subtrees: []*expressions.PathTree{
			{
				PathItem: "a",
				NodeType: expressions.KeyNode,
				Subtrees: []*expressions.PathTree{
					{
						PathItem: "b",
						NodeType: expressions.AccessNode,
						Subtrees: nil,
					},
				},
			},
		},
	}

	defaultRequirements := expressions.UnpackRequirements{}
	withKeysRequirements := expressions.UnpackRequirements{
		IncludeKeys: true,
	}

	noKeyTreePaths := pathTree.Unpack(defaultRequirements)
	assert.Equals(t, len(noKeyTreePaths), 1)
	assert.Equals(t, noKeyTreePaths[0].String(), "$.b")
	withKeyTreePaths := pathTree.Unpack(withKeysRequirements)
	assert.Equals(t, len(withKeyTreePaths), 1)
	assert.Equals(t, withKeyTreePaths[0].String(), "$.a.b")
}

func TestPathTree_UnpackL2Excluded(t *testing.T) {
	pathTree := expressions.PathTree{
		PathItem: "$",
		NodeType: expressions.DataRootNode,
		Subtrees: []*expressions.PathTree{
			{
				PathItem: "a",
				NodeType: expressions.PastTerminalNode,
				Subtrees: nil,
			},
			{
				PathItem: "b",
				NodeType: expressions.PastTerminalNode,
				Subtrees: nil,
			},
		},
	}
	noPastTerminalRequirements := expressions.UnpackRequirements{
		StopAtTerminals: true,
	}

	treePaths := pathTree.Unpack(noPastTerminalRequirements)
	assert.Equals(t, len(treePaths), 1)
	assert.Equals(t, treePaths[0].String(), "$")
}

func TestPathTree_UnpackL3Excluded(t *testing.T) {
	pathTree := expressions.PathTree{
		PathItem: "$",
		NodeType: expressions.DataRootNode,
		Subtrees: []*expressions.PathTree{
			{
				PathItem: "l2-a",
				NodeType: expressions.AccessNode,
				Subtrees: []*expressions.PathTree{
					{
						PathItem: "l3",
						NodeType: expressions.PastTerminalNode,
						Subtrees: nil,
					},
				},
			},
			{
				PathItem: "l2-b",
				NodeType: expressions.AccessNode,
				Subtrees: []*expressions.PathTree{
					{
						PathItem: "l3",
						NodeType: expressions.PastTerminalNode,
						Subtrees: nil,
					},
				},
			},
		},
	}
	noPastTerminalRequirements := expressions.UnpackRequirements{
		StopAtTerminals: true,
	}

	treePaths := pathTree.Unpack(noPastTerminalRequirements)
	assert.Equals(t, len(treePaths), 2)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.l2-a", treePaths)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.l2-b", treePaths)
}

func TestPathTree_UnpackManySubtrees(t *testing.T) {
	pathTree := expressions.PathTree{
		PathItem: "$",
		NodeType: expressions.DataRootNode,
		Subtrees: []*expressions.PathTree{
			{
				PathItem: "l2-a",
				NodeType: expressions.AccessNode,
				Subtrees: []*expressions.PathTree{
					{
						PathItem: "l3-a",
						NodeType: expressions.AccessNode,
						Subtrees: nil,
					},
					{
						PathItem: "l3-b",
						NodeType: expressions.AccessNode,
						Subtrees: nil,
					},
				},
			},
			{
				PathItem: "l2-b",
				NodeType: expressions.AccessNode,
				Subtrees: []*expressions.PathTree{
					{
						PathItem: "l3-a",
						NodeType: expressions.AccessNode,
						Subtrees: []*expressions.PathTree{
							{
								PathItem: "l4-a",
								NodeType: expressions.AccessNode,
								Subtrees: nil,
							},
							{
								PathItem: "l4-b",
								NodeType: expressions.AccessNode,
								Subtrees: []*expressions.PathTree{
									{
										PathItem: "l5-a",
										NodeType: expressions.AccessNode,
										Subtrees: nil,
									},
									{
										PathItem: "l5-b",
										NodeType: expressions.AccessNode,
										Subtrees: nil,
									},
								},
							},
						},
					},
					{
						PathItem: "l3-b",
						NodeType: expressions.AccessNode,
						Subtrees: nil,
					},
				},
			},
		},
	}
	defaultRequirements := expressions.UnpackRequirements{}

	funcTreePaths := pathTree.Unpack(defaultRequirements)
	assert.Equals(t, len(funcTreePaths), 6)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.l2-a.l3-a", funcTreePaths)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.l2-a.l3-b", funcTreePaths)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.l2-b.l3-a.l4-a", funcTreePaths)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.l2-b.l3-a.l4-b.l5-a", funcTreePaths)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.l2-b.l3-a.l4-b.l5-b", funcTreePaths)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.l2-b.l3-b", funcTreePaths)
}

func TestPathTree_ErrorUnpackInvalidType(t *testing.T) {
	// The NodeType is an alias of String, so submit an invalid one to ensure
	// that it panics.
	pathTree := expressions.PathTree{
		PathItem: "$",
		NodeType: "abc-invalid",
		Subtrees: nil,
	}

	defaultRequirements := expressions.UnpackRequirements{}

	assert.Panics(t, func() {
		_ = pathTree.Unpack(defaultRequirements)
	})
}
