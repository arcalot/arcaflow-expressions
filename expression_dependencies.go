package expressions

import (
	"fmt"
	"go.flow.arcalot.io/expressions/internal/ast"
	"go.flow.arcalot.io/pluginsdk/schema"
)

// dependencyContext holds the root data for a dependency evaluation in an expression. This is useful so that we
// don't need to pass the root type, path, and workflow context along with each function call.
type dependencyContext struct {
	rootType        schema.Type
	rootPath        PathTree
	workflowContext map[string][]byte
	functions       map[string]schema.Function
}

func (c *dependencyContext) rootDependencies(
	node ast.Node,
) (schema.Type, *PathTree, []*PathTree, error) {
	newRoot := c.rootPath
	resolvedType, chainablePath, dependencies, err := c.dependencies(node, c.rootType, &newRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	// Non-chainable types do not have a resultant dependency.
	if chainablePath != nil {
		dependencies = append(dependencies, &newRoot)
	}
	return resolvedType, chainablePath, dependencies, nil
}

// dependencies evaluates an AST node for possible dependencies. It adds items to the specified path tree and returns
// it. You can use this to build a list of value paths that make up the dependencies of this expression. Furthermore,
// you can also use this function to evaluate the type the resolved expression's value will have.
//
// Arguments:
// - node: The root node of the tree of sub-tree to evaluate.
// - currentType: The schema, which specifies the values and their types that can be referenced.
// - path: A reference to the PathTree to the current node, which will have sub-trees added to it.
// Returns:
//   - schema.Type: The schema for the value.
//   - *PathTree, the chainable path to the subtree node that can be built upon.
//   - []*PathTree, the path to the values this node depends on in the input schema. Empty if it's a literal.
//   - error: An error, if encountered.
func (c *dependencyContext) dependencies(
	node ast.Node,
	currentType schema.Type,
	path *PathTree,
) (schema.Type, *PathTree, []*PathTree, error) {
	switch n := node.(type) {
	case *ast.DotNotation:
		return c.dotNotationDependencies(n, currentType, path)
	case *ast.BracketAccessor:
		return c.bracketAccessorDependencies(n, currentType, path)
	case *ast.Identifier:
		return c.identifierDependencies(n, currentType, path)
	case *ast.StringLiteral:
		return schema.NewStringSchema(nil, nil, nil), nil, []*PathTree{}, nil
	case *ast.IntLiteral:
		return schema.NewIntSchema(nil, nil, nil), nil, []*PathTree{}, nil
	case *ast.FunctionCall:
		return c.functionDependencies(n)
	default:
		return nil, nil, nil, fmt.Errorf("unsupported AST node type: %T", n)
	}
}

// Note: A function itself doesn't have a path dependency, but its args could.
// Therefore, it cannot be chained, so it doesn't return that type.
func (c *dependencyContext) functionDependencies(node *ast.FunctionCall) (schema.Type, *PathTree, []*PathTree, error) {
	// Get the types and dependencies of all parameters.
	functionSchema, found := c.functions[node.FuncIdentifier.IdentifierName]
	if !found {
		return nil, nil, nil, fmt.Errorf("could not find function '%s'", node.FuncIdentifier.IdentifierName)
	}
	paramTypes := functionSchema.Parameters()
	// Validate param count
	if node.ArgumentInputs.NumChildren() != len(paramTypes) {
		return nil, nil, nil, fmt.Errorf("invalid call to function '%s'. Expected %d args, got %d args. Function schema: %s",
			functionSchema.ID(), len(paramTypes), node.ArgumentInputs.NumChildren(), functionSchema.String())
	}
	// Types need to be saved to validate argument types with parameter types, which are also needed to get the output type.
	// Dependencies need to also be added to the PathTree
	dependencies := make([]*PathTree, 0)
	// Save arg types for passing into output function
	argTypes := make([]schema.Type, 0)
	for i := 0; i < len(node.ArgumentInputs.Arguments); i++ {
		arg := node.ArgumentInputs.Arguments[i]
		argType, _, argDependencies, err := c.rootDependencies(arg)
		if err != nil {
			return nil, nil, nil, err
		}
		// Validate type compatibility with function's schema
		paramType := paramTypes[i]
		if err := paramType.ValidateCompatibility(argType); err != nil {
			return nil, nil, nil, fmt.Errorf("error while validating arg/param type compatibility for function '%s' at 0-index %d (%w). Function schema: %s",
				functionSchema.ID(), i, err, functionSchema.String())
		}
		argTypes = append(argTypes, argType)
		// Add dependency to the path tree
		dependencies = append(dependencies, argDependencies...)
	}
	// Now get the type from the function output
	outputType, _, err := functionSchema.Output(argTypes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error while getting return type (%w)", err)
	}
	return outputType, nil, dependencies, nil
}

// dotNotationDependencies resolves dependencies of a DotNotation node.
//
// The dot notation is when item.item is encountered. We simply traverse the AST in order, left to right,
// nothing specific to do.
func (c *dependencyContext) dotNotationDependencies(
	node *ast.DotNotation,
	currentType schema.Type,
	path *PathTree,
) (schema.Type, *PathTree, []*PathTree, error) {
	// Theoretical scenario, the left is a function call. It's dependencies are all that matter.
	// Alternative scenario, the left is an instance of access to the main data structure. In this case, it is its
	// own dependency.
	// Start with the left access.
	leftType, leftChainablePath, leftDependencies, err := c.dependencies(node.LeftAccessibleNode, currentType, path)
	if err != nil {
		return nil, nil, nil, err
	}
	// Right dependencies, using left type.
	rightType, rightChainablePath, rightDependencies, err := c.dependencies(node.RightAccessIdentifier, leftType, leftChainablePath)
	if err != nil {
		return nil, nil, nil, err
	}
	// If the left isn't chainable, we use include its dependencies.
	finalDependencies := append(rightDependencies, leftDependencies...)
	return rightType, rightChainablePath, finalDependencies, nil
}

// bracketAccessorDependencies resolves dependencies for a BracketAccessor node,
// resolving the left type, as well as the value in the brackets, to find the result.
func (c *dependencyContext) bracketAccessorDependencies(
	node *ast.BracketAccessor,
	currentType schema.Type,
	path *PathTree,
) (schema.Type, *PathTree, []*PathTree, error) {
	// A bracket accessor is when an item[item] is encountered. Here we need to evaluate the left tree as usual, then
	// use the right tree according to its type. This is either a literal (e.g. a string), or it is a subexpression.
	// Literals will call dependenciesMapKey, while subexpressions need to be evaluated on their own on the root
	// type.
	leftType, leftPath, leftDependencies, err := c.dependencies(node.LeftNode, currentType, path)
	if err != nil {
		return nil, nil, nil, err
	}

	// Evaluate the subexpression
	keyType, _, keyDependencies, err := c.rootDependencies(node.RightExpression)
	if err != nil {
		return nil, nil, nil, err
	}
	mergedDependencies := append(leftDependencies, keyDependencies...)
	var typeResult schema.Type
	var chainablePath *PathTree
	var currentDependencies []*PathTree
	switch leftType.TypeID() {
	case schema.TypeIDMap:
		typeResult, chainablePath, currentDependencies, err = c.bracketSubExprMapDependencies(keyType, leftType, leftPath)
	case schema.TypeIDList:
		typeResult, chainablePath, currentDependencies, err = c.bracketListDependencies(keyType, leftType, leftPath)
	case schema.TypeIDAny:
		typeResult, chainablePath, currentDependencies, err = schema.NewAnySchema(), leftPath, []*PathTree{}, nil
	default:
		// We don't support subexpressions to pick a property on an object type since that would result in
		// unpredictable behavior and runtime errors. Furthermore, we would not be able to perform type
		// evaluation.
		return nil, nil, nil, fmt.Errorf("subexpressions are only supported on map, list, and any types, %s given", currentType.TypeID())
	}
	// For literals, add extraneous data.
	chainablePath = c.addExtraneous(node.RightExpression, chainablePath)
	mergedDependencies = append(mergedDependencies, currentDependencies...)
	return typeResult, chainablePath, mergedDependencies, err
}

// bracketSubExprMapDependencies is used to resolve dependencies when a bracket accessor has a subexpression,
// with the left type being a map. So format `map[sub-expression]`
func (c *dependencyContext) bracketSubExprMapDependencies(
	keyType schema.Type,
	leftType schema.Type,
	leftPath *PathTree,
) (schema.Type, *PathTree, []*PathTree, error) {
	// For maps, we try to compare the type of the map key with the resulting type of the subexpression to
	// make sure that there are no runtime type failures. The user may need to add type conversion functions
	// to their expressions to convert an integer to a string, for example.
	mapType := leftType.(schema.UntypedMap)
	if keyType.TypeID() != mapType.Keys().TypeID() {
		return nil, nil, nil, fmt.Errorf("subexpression evaluates to type '%s' for a map, '%s' expected", keyType.TypeID(), mapType.Keys().TypeID())
	}
	return mapType.Values(), leftPath, []*PathTree{}, nil
}

// bracketListDependencies is used to resolve dependencies when a bracket accessor has a subexpression,
// with the left type being a list.
func (c *dependencyContext) bracketListDependencies(
	keyType schema.Type,
	leftType schema.Type,
	path *PathTree,
) (schema.Type, *PathTree, []*PathTree, error) {
	// Lists have integer indexes, so we try to make sure that the subexpression is yielding an int or
	// int-like type. This will have the best chance of not resulting in a runtime error.
	list := leftType.(schema.UntypedList)
	switch keyType.TypeID() {
	case schema.TypeIDInt:
	default:
		return nil, nil, nil, fmt.Errorf("subexpressions resulted in a %s type for a list key, integer expected", keyType.TypeID())
	}
	return list.Items(), path, []*PathTree{}, nil
}

func (c *dependencyContext) addExtraneous(node ast.Node, path *PathTree) *PathTree {
	// If literal, include that in the extraneous info, only if path present.
	if path != nil { // Do this check first, to skip the literal check when possible.
		if literalValue, isLiteral := node.(ast.ValueLiteral); isLiteral {
			pathItem := &PathTree{
				PathItem:     literalValue.Value(),
				IsExtraneous: true,
				Subtrees:     nil,
			}
			if path != nil {
				path.Subtrees = append(path.Subtrees, pathItem)
			}
			return pathItem
		}
	}
	return path
}

func (c *dependencyContext) identifierDependencies(
	node *ast.Identifier,
	currentType schema.Type,
	path *PathTree,
) (schema.Type, *PathTree, []*PathTree, error) {
	switch node.IdentifierName {
	case "$":
		// This identifier means the root of the expression.
		// Validate that the given path is actually at the root.
		if path.PathItem != "$" {
			return nil, nil, nil, fmt.Errorf("root access chained after non-root")
		}
		return c.rootType, path, []*PathTree{}, nil
	default:
		// This case is the item.item type expression, where the right item is the "identifier" in question.
		return dependenciesAccessObject(currentType, node.IdentifierName, path)
	}
}

// dependenciesAccessKnownKey this function reads the object on the left to determine
// the type of the property referenced.
func dependenciesAccessObject(
	leftType schema.Type,
	identifier string,
	path *PathTree,
) (schema.Type, *PathTree, []*PathTree, error) {
	switch leftType.TypeID() {
	case schema.TypeIDScope, schema.TypeIDRef, schema.TypeIDObject:
		// Object-likes always have field names (strings) as keys, so we need to convert the passed value to a string.
		// 99% of the time these are going to be strings anyway.
		currentObject := leftType.(schema.Object)
		properties := currentObject.Properties()
		property, ok := properties[identifier]
		if !ok {
			return nil, nil, nil, fmt.Errorf("object %s does not have a property named %q", currentObject.ID(), identifier)
		}
		pathItem := &PathTree{
			PathItem:     identifier,
			IsExtraneous: false,
			Subtrees:     nil,
		}
		if path != nil {
			path.Subtrees = append(path.Subtrees, pathItem)
		}
		return property.Type(), pathItem, []*PathTree{}, nil
	case schema.TypeIDAny:
		// We're accessing an unvalidated field, because the type is 'any.
		// So mark the subtree as extraneous.
		pathItem := &PathTree{
			PathItem:     identifier,
			IsExtraneous: true,
			Subtrees:     nil,
		}
		if path != nil {
			path.Subtrees = append(path.Subtrees, pathItem)
		}
		return schema.NewAnySchema(), pathItem, []*PathTree{}, nil
	default:
		return nil, nil, nil,
			fmt.Errorf("cannot evaluate expression identifier %s on data type %s", identifier, leftType.TypeID())
	}
}
