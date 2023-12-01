package expressions

import (
	"fmt"
	"go.flow.arcalot.io/pluginsdk/schema"
	"reflect"

	"go.flow.arcalot.io/expressions/internal/ast"
)

// evaluateContext holds the root data and context for a value evaluation in an expression. This is useful so that we
// don't need to pass the data, root data, and workflow context along with each function call.
type evaluateContext struct {
	rootData        any
	functions       map[string]schema.CallableFunction
	workflowContext map[string][]byte
}

// evaluate evaluates the passed  node on a set of data consisting of primitive types. It must also have access
// to the root data to evaluate subexpressions, as well as the workflow context to pull in additional files. It will
// return the evaluated data.
func (c evaluateContext) evaluate(node ast.Node, data any) (any, error) {
	// First checks for any literal type, since it's generic.
	if literal, isLiteral := node.(ast.ValueLiteral); isLiteral {
		return literal.Value(), nil
	}
	// Checks non-generic types.
	switch n := node.(type) {
	case *ast.DotNotation:
		return c.evaluateDotNotation(n, data)
	case *ast.BracketAccessor:
		return c.evaluateBracketAccessor(n, data)
	case *ast.Identifier:
		return c.evaluateIdentifier(n, data)
	case *ast.FunctionCall:
		return c.evaluateFuncCall(n, data)
	default:
		return nil, fmt.Errorf("unsupported node type: %T", n)
	}
}

func (c evaluateContext) evaluateFuncCall(node *ast.FunctionCall, data any) (any, error) {
	funcID := node.FuncIdentifier
	functionSchema, found := c.functions[funcID.String()]
	if found {
		// Evaluate args
		evaluatedArgs, err := c.evaluateParameters(node.ParameterInputs, data)
		if err != nil {
			return nil, err
		}
		expectedArgs := len(functionSchema.Parameters())
		gotArgs := len(evaluatedArgs)
		if gotArgs != expectedArgs {
			return nil, fmt.Errorf(
				"function '%s' called with incorrect number of arguments. Expected %d, got %d",
				funcID, expectedArgs, gotArgs)
		}
		return functionSchema.Call(evaluatedArgs)
	} else {
		return nil, fmt.Errorf("function with ID '%s' not found", funcID)
	}
}

func (c evaluateContext) evaluateParameters(node *ast.ArgumentList, _ any) ([]any, error) {
	// A value for each argument
	result := make([]any, node.NumChildren())
	for i := 0; i < node.NumChildren(); i++ {
		arg, err := node.GetChild(i)
		if err != nil {
			return nil, err
		}
		result[i], err = c.evaluate(arg, c.rootData)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// Evaluates a DotNotation node.
//
// The dot notation is an item.item expression part, where we simply need to evaluate the
// left and right subtrees in order.
func (c evaluateContext) evaluateDotNotation(node *ast.DotNotation, data any) (any, error) {
	leftResult, err := c.evaluate(node.LeftAccessibleNode, data)
	if err != nil {
		return nil, err
	}
	return c.evaluate(node.RightAccessIdentifier, leftResult)
}

// Evaluates a MapAccessor node, which is a more advanced version of dot notation
//
// The map accessor is an item[item] expression part, where we evaluate the left subtree first, then the right
// subtree to obtain the map key. Finally, the map key is used to look up the resulting data.
func (c evaluateContext) evaluateBracketAccessor(node *ast.BracketAccessor, data any) (any, error) {
	// First evaluate the value to the left of the [], since we're accessing a value in it.
	leftResult, err := c.evaluate(node.LeftNode, data)
	if err != nil {
		return nil, err
	}
	// Next, evaluates the item inside the brackets. Can be any valid literal or something that evaluates into a value.
	mapKey, err := c.evaluate(node.RightExpression, leftResult)
	if err != nil {
		return nil, err
	}
	return evaluateMapAccess(data, mapKey)
}

// Evaluates an identifier
// Identifiers are items in dot notation.
func (c evaluateContext) evaluateIdentifier(node *ast.Identifier, data any) (any, error) {
	switch node.IdentifierName {
	case "$":
		// $ is the root node of the data structure.
		return c.rootData, nil
	default:
		// Otherwise, it's a normal accessor key, which we evaluate like a map key.
		return evaluateMapAccess(data, node.IdentifierName)
	}
}

// evaluateMapKey is a helper function for evaluate that extracts an item in maps, lists, or object-likes when an
// identifier or map accessor is encountered.
func evaluateMapAccess(data any, mapKey any) (any, error) {
	dataVal := reflect.ValueOf(data)
	switch dataVal.Kind() {
	case reflect.Map:
		// In case of a map, we simply look up the value passed.
		indexValue := dataVal.MapIndex(reflect.ValueOf(mapKey))
		if !indexValue.IsValid() {
			return nil, fmt.Errorf("map key %v not found", mapKey)
		}
		return indexValue.Interface(), nil
	case reflect.Slice:
		// In case of slices we want integers. The user is responsible for converting the type to an integer themselves.
		var sliceIndex int
		switch t := mapKey.(type) {
		case int:
			sliceIndex = t
		case int64:
			sliceIndex = int(t)
		default:
			return nil, fmt.Errorf("unsupported map key type: %T", mapKey)
		}
		sliceLen := dataVal.Len()
		if sliceLen <= sliceIndex {
			return nil, fmt.Errorf("index %d is larger than the list items (%d)", sliceIndex, sliceLen)
		}
		indexValue := dataVal.Index(sliceIndex)
		return indexValue.Interface(), nil
	default:
		return nil, fmt.Errorf(
			"cannot evaluate identifier %v on a %s",
			mapKey,
			dataVal.Kind().String(),
		)
	}
}
