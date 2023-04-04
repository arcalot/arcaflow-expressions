package expressions

import (
	"fmt"
	"reflect"

	"go.flow.arcalot.io/expressions/internal/ast"
)

// evaluate evaluates the passed  node on a set of data consisting of primitive types. It must also have access
// to the root data to evaluate subexpressions, as well as the workflow context to pull in additional files. It will
// return the evaluated data.
func evaluate(node ast.Node, data any, rootData any, workflowContext map[string][]byte) (any, error) {
	switch n := node.(type) {
	case *ast.DotNotation:
		return evaluateDotNotation(n, data, rootData, workflowContext)
	case *ast.MapAccessor:
		return evaluateMapAccessor(n, data, rootData, workflowContext)
	case *ast.Key:
		return evaluateKey(n, data, rootData, workflowContext)
	case *ast.Identifier:
		return evaluateIdentifier(n, data, rootData)
	default:
		return nil, fmt.Errorf("unsupported  node type: %T", n)
	}
}

// Evaluates a DotNotation node.
//
// The dot notation is an item.item expression part, where we simply need to evaluate the
// left and right subtrees in order.
func evaluateDotNotation(node *ast.DotNotation, data any, rootData any, workflowContext map[string][]byte) (any, error) {
	leftResult, err := evaluate(node.LeftAccessibleNode, data, rootData, workflowContext)
	if err != nil {
		return nil, err
	}
	return evaluate(node.RightAccessIdentifier, leftResult, rootData, workflowContext)
}

// Evaluates a MapAccessor node, which is a more advanced version of dot notation
//
// The map accessor is an item[item] expression part, where we evaluate the left subtree first, then the right
// subtree to obtain the map key. Finally, the map key is used to look up the resulting data.
func evaluateMapAccessor(node *ast.MapAccessor, data any, rootData any, workflowContext map[string][]byte) (any, error) {
	// First evaluate the value to the left of the [], since we're accessing a value in it.
	leftResult, err := evaluate(node.LeftNode, data, rootData, workflowContext)
	if err != nil {
		return nil, err
	}
	// Next, evaluates the item inside the brackets. Can be any valid literal or something that evaluates into a value.
	mapKey, err := evaluate(&node.RightKey, leftResult, rootData, workflowContext)
	if err != nil {
		return nil, err
	}
	return evaluateMapAccess(data, mapKey)
}

// Evaluates a key, which is the item looked up in a map access
//
// A map access has the form `item[itemkey]`, and the key can be either a literal (e.g. string) or a
// subexpression, which needs to be evaluated in its own right.
func evaluateKey(node *ast.Key, data any, rootData any, workflowContext map[string][]byte) (any, error) {
	switch {
	case node.Literal != nil:
		return node.Literal.Value(), nil
	case node.SubExpression != nil:
		return evaluate(node.SubExpression, data, rootData, workflowContext)
	default:
		return nil, fmt.Errorf("bug: neither literal, nor subexpression are set on key")
	}
}

// Evaluates an identifier
// Identifiers are items in dot notation.
func evaluateIdentifier(node *ast.Identifier, data any, rootData any) (any, error) {
	switch node.IdentifierName {
	case "$":
		// $ is the root node of the data structure.
		return rootData, nil
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
