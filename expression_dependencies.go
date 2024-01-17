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

type dependencyResult struct {
	resolvedType   schema.Type // The type resolved for the node specified.
	chainablePath  *PathTree   // The chainable path for accessing this value, or fields within it. The current leaf node.
	rootPathResult *PathTree   // The root path tree, if known.
	completedPaths []*PathTree // Completed dependency paths.
}

func (d *dependencyResult) addCompletedDependencies(newCompletedPaths []*PathTree) {
	d.completedPaths = append(d.completedPaths, newCompletedPaths...)
}

func (d *dependencyResult) addCompletedDependency(newCompletedPath *PathTree) {
	d.completedPaths = append(d.completedPaths, newCompletedPath)
}

// rootDependencies evaluates the dependencies with a new root, and populates the completedPaths field with the new root.
func (c *dependencyContext) rootDependencies(
	node ast.Node,
) (*dependencyResult, error) {
	newRoot := c.rootPath
	result, err := c.dependencies(node, c.rootType, &newRoot)
	if err != nil {
		return nil, err
	}
	// Currently, literals wouldn't give a path.
	if result.rootPathResult != nil {
		result.addCompletedDependency(result.rootPathResult)
	}
	return result, nil
}

// dependencies evaluates an AST node for possible dependencies. It adds items to the specified path tree and returns
// it. You can use this to build a list of value paths that make up the dependencies of this expression. Furthermore,
// you can also use this function to evaluate the type the resolved expression's value will have.
//
// Arguments:
//   - node: The AST node to resolve the dependencies for.
//   - currentType: The schema, which specifies the values and their types that can be referenced for the current node.
//   - path: A reference to the PathTree to the current node, which will have sub-trees added to it.
//
// Returns:
//   - *dependencyResult: A reference to the struct that contains the chainable path, the completed paths, and the type.
//   - error: An error, if encountered.
func (c *dependencyContext) dependencies(
	node ast.Node,
	currentType schema.Type,
	path *PathTree,
) (*dependencyResult, error) {
	switch n := node.(type) {
	case *ast.DotNotation:
		return c.dotNotationDependencies(n, currentType, path)
	case *ast.BracketAccessor:
		return c.bracketAccessorDependencies(n, currentType, path)
	case *ast.Identifier:
		return c.identifierDependencies(n, currentType, path)
	case *ast.StringLiteral:
		return &dependencyResult{resolvedType: schema.NewStringSchema(nil, nil, nil)}, nil
	case *ast.IntLiteral:
		return &dependencyResult{resolvedType: schema.NewIntSchema(nil, nil, nil)}, nil
	case *ast.FunctionCall:
		return c.functionDependencies(n)
	default:
		return nil, fmt.Errorf("unsupported AST node type: %T", n)
	}
}

func (c *dependencyContext) functionDependencies(node *ast.FunctionCall) (*dependencyResult, error) {
	// Get the types and dependencies of all parameters.
	functionSchema, found := c.functions[node.FuncIdentifier.IdentifierName]
	if !found {
		return nil, fmt.Errorf("could not find function '%s'", node.FuncIdentifier.IdentifierName)
	}
	paramTypes := functionSchema.Parameters()
	// Validate param count
	if node.ArgumentInputs.NumChildren() != len(paramTypes) {
		return nil, fmt.Errorf("invalid call to function '%s'. Expected %d args, got %d args. Function schema: %s",
			functionSchema.ID(), len(paramTypes), node.ArgumentInputs.NumChildren(), functionSchema.String())
	}
	// Types need to be saved to validate argument types with parameter types, which are also needed to get the output type.
	// Dependencies need to also be added to the PathTree
	dependencies := make([]*PathTree, 0)
	// Save arg types for passing into output function
	argTypes := make([]schema.Type, 0)
	for i := 0; i < len(node.ArgumentInputs.Arguments); i++ {
		arg := node.ArgumentInputs.Arguments[i]
		argResult, err := c.rootDependencies(arg)
		if err != nil {
			return nil, err
		}
		// Validate type compatibility with function's schema
		paramType := paramTypes[i]
		if err := paramType.ValidateCompatibility(argResult.resolvedType); err != nil {
			return nil, fmt.Errorf("error while validating arg/param type compatibility for function '%s' at 0-index %d (%w). Function schema: %s",
				functionSchema.ID(), i, err, functionSchema.String())
		}
		argTypes = append(argTypes, argResult.resolvedType)
		// Add dependency to the path tree
		dependencies = append(dependencies, argResult.completedPaths...)
	}
	// Now get the type from the function output
	outputType, _, err := functionSchema.Output(argTypes)
	if err != nil {
		return nil, fmt.Errorf("error while getting return type (%w)", err)
	}
	// Create the chainable path and root dependency node for the function
	functionRootPath := &PathTree{
		PathItem: node.FuncIdentifier.IdentifierName,
		NodeType: FunctionNode,
		Subtrees: nil,
	}
	return &dependencyResult{
		resolvedType:   outputType,
		chainablePath:  functionRootPath,
		rootPathResult: functionRootPath,
		completedPaths: dependencies,
	}, nil
}

// dotNotationDependencies resolves dependencies of a DotNotation node.
//
// The dot notation is when item.item is encountered. We simply traverse the AST in order, left to right,
// nothing specific to do.
func (c *dependencyContext) dotNotationDependencies(
	node *ast.DotNotation,
	currentType schema.Type,
	path *PathTree,
) (*dependencyResult, error) {
	// Theoretical scenario, the left is a function call. It's dependencies are all that matter.
	// Alternative scenario, the left is an instance of access to the main data structure. In this case, it is its
	// own dependency.
	// Start with the left access.
	leftResult, err := c.dependencies(node.LeftAccessibleNode, currentType, path)
	if err != nil {
		return nil, err
	}
	// Right dependencies, using left type.
	rightResult, err := c.dependencies(node.RightAccessIdentifier, leftResult.resolvedType, leftResult.chainablePath)
	if err != nil {
		return nil, err
	}
	finalDependencies := append(rightResult.completedPaths, leftResult.completedPaths...)
	return &dependencyResult{
		resolvedType:   rightResult.resolvedType,
		chainablePath:  rightResult.chainablePath,
		rootPathResult: leftResult.rootPathResult,
		completedPaths: finalDependencies,
	}, nil
}

// bracketAccessorDependencies resolves dependencies for a BracketAccessor node, which is
// when item[item] is encountered.
func (c *dependencyContext) bracketAccessorDependencies(
	node *ast.BracketAccessor,
	currentType schema.Type,
	path *PathTree,
) (*dependencyResult, error) {
	// Start with the part before the []
	leftResult, err := c.dependencies(node.LeftNode, currentType, path)
	if err != nil {
		return nil, err
	}

	// Evaluate the subexpression, the part in the brackets []
	keyResult, err := c.rootDependencies(node.RightExpression)
	if err != nil {
		return nil, err
	}
	mergedDependencies := append(leftResult.completedPaths, keyResult.completedPaths...)
	var overallResult *dependencyResult
	switch leftResult.resolvedType.TypeID() {
	case schema.TypeIDMap:
		overallResult, err = c.bracketMapDependencies(leftResult, keyResult.resolvedType)
	case schema.TypeIDList:
		overallResult, err = c.bracketListDependencies(leftResult, keyResult.resolvedType)
	case schema.TypeIDAny:
		overallResult, err = &dependencyResult{
			resolvedType:   schema.NewAnySchema(),
			chainablePath:  leftResult.chainablePath,
			rootPathResult: leftResult.rootPathResult,
		}, nil
	case schema.TypeIDScope, schema.TypeIDObject, schema.TypeIDRef:
		// This is supported in JavaScript, but not this expression language. This is because objects have
		// different types for each field, meaning that the type cannot be determined at this point.
		return nil, fmt.Errorf(
			"bracket ([]) access is not supported for object/scope/ref types; please use dot notation",
		)
	default:
		return nil, fmt.Errorf(
			"bracket ([]) subexpressions are only supported on 'map', 'list', and 'any' types; %s given",
			currentType.TypeID(),
		)
	}
	// For literals, add key data.
	overallResult.chainablePath = c.addKeyNode(node.RightExpression, overallResult.chainablePath)
	overallResult.addCompletedDependencies(mergedDependencies)
	return overallResult, err
}

// bracketMapDependencies is used to resolve dependencies when a bracket accessor has a subexpression,
// with the left type being a map. So format `map[sub-expression]`
func (c *dependencyContext) bracketMapDependencies(
	leftResult *dependencyResult,
	keyType schema.Type,
) (*dependencyResult, error) {
	// For maps, we try to compare the type of the map key with the resulting type of the subexpression to
	// make sure that there are no runtime type failures. The user may need to add type conversion functions
	// to their expressions to convert an integer to a string, for example.
	mapType := leftResult.resolvedType.(schema.UntypedMap)
	if keyType.TypeID() != mapType.Keys().TypeID() {
		return nil, fmt.Errorf("subexpression evaluates to type '%s' for a map, '%s' expected", keyType.TypeID(), mapType.Keys().TypeID())
	}
	return &dependencyResult{
		resolvedType:   mapType.Values(),
		chainablePath:  leftResult.chainablePath,
		rootPathResult: leftResult.rootPathResult,
	}, nil
}

// bracketListDependencies is used to resolve dependencies when a bracket accessor has a subexpression,
// with the left type being a list.
func (c *dependencyContext) bracketListDependencies(
	leftResult *dependencyResult,
	keyType schema.Type,
) (*dependencyResult, error) {
	// Lists have integer indexes, so make sure that the subexpression is yielding an int.
	list := leftResult.resolvedType.(schema.UntypedList)
	if keyType.TypeID() != schema.TypeIDInt {
		return nil, fmt.Errorf("subexpressions resulted in a %s type for a list key, integer expected", keyType.TypeID())
	}
	return &dependencyResult{
		resolvedType:   list.Items(), // The type is the type for an individual item of the list.
		chainablePath:  leftResult.chainablePath,
		rootPathResult: leftResult.rootPathResult,
	}, nil
}

// If the key is literal, include the value in a key-type node.
// This extends the chainable path.
func (c *dependencyContext) addKeyNode(node ast.Node, path *PathTree) *PathTree {
	literalValue, isLiteral := node.(ast.ValueLiteral)
	if !isLiteral {
		return path
	}
	pathItem := &PathTree{
		PathItem: literalValue.Value(),
		NodeType: KeyNode,
		Subtrees: nil,
	}
	path.Subtrees = append(path.Subtrees, pathItem)
	return pathItem
}

func (c *dependencyContext) identifierDependencies(
	node *ast.Identifier,
	currentType schema.Type,
	path *PathTree,
) (*dependencyResult, error) {
	switch node.IdentifierName {
	case "$":
		var root *PathTree
		// If the given node is root, use it. If nil, create it.
		if path == nil {
			root = &PathTree{
				PathItem: "$",
				NodeType: DataRootNode,
				Subtrees: nil,
			}
		} else if path.NodeType == DataRootNode {
			root = path
		} else {
			return nil, fmt.Errorf("root access %q of type %q not at root", path.PathItem, path.NodeType)
		}
		// The path is validated as the root already.
		return &dependencyResult{
			resolvedType:   c.rootType,
			chainablePath:  path,
			rootPathResult: root,
		}, nil
	default:
		// This case is the item.item type expression, where the right item is the "identifier" in question.
		return dependenciesAccessObject(currentType, node.IdentifierName, path)
	}
}

// dependenciesAccessObject reads the object on the left to determine
// the type of the property referenced.
func dependenciesAccessObject(
	leftType schema.Type,
	identifier string,
	path *PathTree,
) (*dependencyResult, error) {
	switch leftType.TypeID() {
	case schema.TypeIDScope, schema.TypeIDRef, schema.TypeIDObject:
		currentObject := leftType.(schema.Object)
		properties := currentObject.Properties()
		property, ok := properties[identifier]
		if !ok {
			return nil, fmt.Errorf("object %s does not have a property named %q", currentObject.ID(), identifier)
		}
		pathItem := &PathTree{
			PathItem: identifier,
			NodeType: AccessNode,
			Subtrees: nil,
		}
		path.Subtrees = append(path.Subtrees, pathItem)
		return &dependencyResult{
			resolvedType:  property.Type(),
			chainablePath: pathItem,
			// In case the root isn't explicitly set with '$.', set the root to the current path.
			rootPathResult: path,
		}, nil
	case schema.TypeIDAny:
		// Since the left type is any (a terminal type), this access (deeper than the 'any' node) is past-terminal.
		pathItem := &PathTree{
			PathItem: identifier,
			NodeType: PastTerminalNode,
			Subtrees: nil,
		}
		path.Subtrees = append(path.Subtrees, pathItem)
		return &dependencyResult{
			resolvedType:  schema.NewAnySchema(),
			chainablePath: pathItem,
		}, nil
	default:
		return nil, fmt.Errorf("cannot evaluate expression identifier %s on data type %s", identifier, leftType.TypeID())
	}
}
