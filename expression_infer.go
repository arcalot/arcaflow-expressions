package expressions

import (
	"go.flow.arcalot.io/pluginsdk/schema"
)

func (e expression) InferSourceType(
	desiredType schema.Type,
	knownSchema schema.Type,
	workflowContext map[string][]byte,
) (schema.Type, error) {
	tree := &PathTree{
		PathItem: "$",
		Subtrees: nil,
	}
	d := &dependencyContext{
		rootType:        knownSchema,
		rootPath:        tree,
		workflowContext: workflowContext,
		deep:            true,
	}
	_, _, err := d.dependencies(e.ast, knownSchema, tree)
	if err != nil {
		return nil, err
	}
	paths := tree.Unpack()

	for _, path := range paths {
		knownSchema = hydrateAnyTypes(path, knownSchema)
	}
	return knownSchema, nil
}

func hydrateAnyTypes(path Path, rootType schema.Type) schema.Type {
	var currentType = &rootType
	for i, entry := range path {
		switch (*currentType).TypeID() {
		case schema.TypeIDAny:
		case schema.TypeIDList:
		case schema.TypeIDMap:
		case schema.ty

		}
	}
	return rootType
}
