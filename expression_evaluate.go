package expressions

import (
	"fmt"
	"go.flow.arcalot.io/pluginsdk/schema"
	"math"
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
		return c.evaluateFuncCall(n)
	case *ast.BinaryOperation:
		return c.evaluateBinaryOperation(n, data)
	case *ast.UnaryOperation:
		return c.evaluateUnaryOperation(n, data)
	default:
		return nil, fmt.Errorf("unsupported node type: %T", n)
	}
}

type SupportedNumber interface {
	int64 | float64
}

func evalNumericalOperation[T SupportedNumber](a, b T, op ast.MathOperationType) (any, error) {
	var aAsAny any = a
	switch op {
	case ast.Invalid:
		return nil, fmt.Errorf("attempted to perform invalid operation")
	case ast.Add:
		return a + b, nil
	case ast.Subtract:
		return a - b, nil
	case ast.Multiply:
		return a * b, nil
	case ast.Divide:
		return a / b, nil
	case ast.Modulus:
		switch typedA := aAsAny.(type) {
		case int64:
			return typedA % int64(b), nil
		case float64:
			return math.Mod(float64(a), float64(b)), nil
		}
		return nil, fmt.Errorf("unsupported type for modulus: %T", a)
	case ast.Power:
		return T(math.Pow(float64(a), float64(b))), nil
	case ast.Equals:
		return a == b, nil
	case ast.NotEquals:
		return a != b, nil
	case ast.GreaterThan:
		return a > b, nil
	case ast.LessThan:
		return a < b, nil
	case ast.GreaterThanEquals:
		return a >= b, nil
	case ast.LessThanEquals:
		return a <= b, nil
	case ast.And, ast.Or:
		return nil, fmt.Errorf("attempted logical operation %s on numeric input %T", op, a)
	default:
		return nil, fmt.Errorf("numeric eval missing case for logical operation %s", op)
	}
}
func evalBooleanOperation(a, b bool, op ast.MathOperationType) (any, error) {
	switch op {
	case ast.Invalid:
		return nil, fmt.Errorf("attempted to perform invalid operation")
	case ast.Power, ast.Modulus, ast.Divide, ast.Multiply, ast.Subtract, ast.Add:
		return nil, fmt.Errorf("attempted to perform math operation '%s' on boolean", op)
	case ast.GreaterThan, ast.LessThan, ast.GreaterThanEquals, ast.LessThanEquals:
		return nil, fmt.Errorf("attempted to perform invalid operation '%s' on boolean", op)
	case ast.Equals:
		return a == b, nil
	case ast.NotEquals:
		return a != b, nil
	case ast.And:
		return a && b, nil
	case ast.Or:
		return a || b, nil
	default:
		return nil, fmt.Errorf("numeric eval missing case for logical operation %s", op)
	}
}

func evalStringOperation(a, b string, op ast.MathOperationType) (any, error) {
	switch op {
	case ast.Invalid:
		return nil, fmt.Errorf("attempted to perform invalid operation")
	case ast.Add:
		// Concatenate
		return a + b, nil
	case ast.Subtract, ast.Multiply, ast.Divide, ast.Modulus, ast.Power:
		return nil, fmt.Errorf("strings do not support operator '%s'", op)
	case ast.Equals:
		return a == b, nil
	case ast.NotEquals:
		return a != b, nil
	case ast.GreaterThan:
		return a > b, nil
	case ast.LessThan:
		return a < b, nil
	case ast.GreaterThanEquals:
		return a >= b, nil
	case ast.LessThanEquals:
		return a <= b, nil
	case ast.And, ast.Or:
		return nil, fmt.Errorf("attempted logical operation %s on string input", op)
	default:
		return nil, fmt.Errorf("string eval missing case for logical operation %s", op)
	}
}

func (c evaluateContext) evaluateBinaryOperation(node *ast.BinaryOperation, data any) (any, error) {
	leftEval, err := c.evaluate(node.Left(), data)
	if err != nil {
		return nil, err
	}
	rightEval, err := c.evaluate(node.Right(), data)
	if err != nil {
		return nil, err
	}
	rightType := reflect.TypeOf(rightEval)
	leftType := reflect.TypeOf(leftEval)
	if rightType != leftType {
		return nil, fmt.Errorf("left type '%s' and right type '%s' of binary operation '%s' do not match",
			leftType, rightType, node.Operation)
	}

	switch left := leftEval.(type) {
	case int64:
		return evalNumericalOperation(left, rightEval.(int64), node.Operation)
	case float64:
		return evalNumericalOperation(left, rightEval.(float64), node.Operation)
	case string:
		return evalStringOperation(left, rightEval.(string), node.Operation)
	case bool:
		return evalBooleanOperation(left, rightEval.(bool), node.Operation)
	default:
		return nil, fmt.Errorf("unsupported type to perform binary operation on: %T", left)
	}
}

func (c evaluateContext) evaluateUnaryOperation(node *ast.UnaryOperation, data any) (any, error) {
	rightEval, err := c.evaluate(node.RightNode, data)
	if err != nil {
		return nil, err
	}
	// Currently we only support negation with unary operators
	if node.LeftOperation != ast.Subtract {
		return nil, fmt.Errorf("only negation is supported with unary evaluation at the moment. Got '%s'", node.LeftOperation)
	}
	switch left := rightEval.(type) {
	case int64:
		return left * -1, nil
	case float64:
		return left * -1.0, nil
	default:
		return nil, fmt.Errorf("unsupported type to perform unary operation on: %T", left)
	}
}

func (c evaluateContext) evaluateFuncCall(node *ast.FunctionCall) (any, error) {
	funcID := node.FuncIdentifier
	functionSchema, found := c.functions[funcID.String()]
	if !found {
		return nil, fmt.Errorf("function with ID '%s' not found", funcID)
	}
	// Evaluate args
	evaluatedArgs, err := c.evaluateParameters(node.ArgumentInputs)
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
}

func (c evaluateContext) evaluateParameters(node *ast.ArgumentList) ([]any, error) {
	// A value for each argument
	result := make([]any, node.NumChildren())
	for i := 0; i < node.NumChildren(); i++ {
		arg := node.Arguments[i]
		var err error
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
		asInt64, isInt64 := mapKey.(int64)
		if !isInt64 {
			return nil, fmt.Errorf("unsupported slice index type '%T', expected int64", mapKey)
		}
		sliceIndex := int(asInt64)
		if int64(sliceIndex) != asInt64 {
			return nil, fmt.Errorf("int64 %d specified is too large for a slice index on the current system", asInt64)
		}
		sliceLen := dataVal.Len()
		if sliceLen <= sliceIndex {
			return nil, fmt.Errorf("index %d is larger than the list items length (%d)", sliceIndex, sliceLen)
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
