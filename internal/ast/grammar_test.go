package ast

import (
	"errors"
	"strings"
	"testing"

	"go.arcalot.io/assert"
)

func TestIdentifierParser(t *testing.T) {
	identifierName := "abc"

	// Create parser
	p, err := InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	identifierResult, err := p.parseIdentifier()

	assert.NoError(t, err)
	assert.Equals(t, identifierName, identifierResult.IdentifierName)

	// No tokens left, so should error out

	_, err = p.parseIdentifier()
	assert.Error(t, err)
}

func TestIdentifierParserInvalidToken(t *testing.T) {
	identifierName := "["

	// Create parser
	p, err := InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	_, err = p.parseIdentifier()

	assert.Error(t, err)
}

// Test proper map access.
func TestMapAccessParser(t *testing.T) {
	expression := "[0]['a']"

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	mapResult, err := p.parseBracketAccess(&Identifier{IdentifierName: "a"})

	assert.NoError(t, err)
	assert.Equals[Node](t, mapResult.RightExpression, &IntLiteral{IntValue: int64(0)})
	assert.InstanceOf[ValueLiteral](t, mapResult.RightExpression)
	assert.Equals(t, mapResult.RightExpression.(ValueLiteral).Value().(int64), int64(0))

	mapResult, err = p.parseBracketAccess(&Identifier{IdentifierName: "a"})

	assert.NoError(t, err)
	assert.Equals[Node](t, mapResult.RightExpression, &StringLiteral{StrValue: "a"})
	assert.InstanceOf[ValueLiteral](t, mapResult.RightExpression)
	assert.Equals(t, mapResult.RightExpression.(ValueLiteral).Value(), "a")

	// Test left and right functions
	assert.Equals(t, mapResult.Right(), mapResult.RightExpression)
	assert.Equals(t, mapResult.Left(), mapResult.LeftNode)

	// No tokens left, so should error out

	_, err = p.parseIdentifier()
	assert.Error(t, err)
}

// Test invalid param.
func TestParseBracketAccessInvalidParam(t *testing.T) {
	identifierName := "[0]"

	// Create parser
	p, err := InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	_, err = p.parseBracketAccess(nil)

	assert.Error(t, err)
}

// Test invalid input.
func TestParseBracketAccessInvalidPrefixToken(t *testing.T) {
	identifierName := "]0]"

	// Create parser
	p, err := InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	_, err = p.parseBracketAccess(&Identifier{IdentifierName: "a"})

	assert.Error(t, err)
}

func TestParseBracketAccessInvalidKey(t *testing.T) {
	identifierName := "[0["

	// Create parser
	p, err := InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	_, err = p.parseBracketAccess(&Identifier{IdentifierName: "a"})

	assert.Error(t, err)
}

// Test cases that test the entire parser.
func TestRootVar(t *testing.T) {
	expression := "$.test"

	// $.test
	root := &DotNotation{}
	root.LeftAccessibleNode = &Identifier{IdentifierName: "$"}
	root.RightAccessIdentifier = &Identifier{IdentifierName: "test"}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*DotNotation)
	if !ok {
		t.Fatalf("Output is not of type *DotNotation")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestRootStrLiteral(t *testing.T) {
	expression := `"test"`

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)
	assert.InstanceOf[*StringLiteral](t, parsedResult)
	assert.Equals(t, parsedResult.(*StringLiteral).StrValue, "test")
}

func TestDotNotation(t *testing.T) {
	expression := "$.parent.child"

	// level2: $.parent
	level2 := &DotNotation{}
	level2.LeftAccessibleNode = &Identifier{IdentifierName: "$"}
	level2.RightAccessIdentifier = &Identifier{IdentifierName: "parent"}
	// root: <level2>.child
	root := &DotNotation{}
	root.LeftAccessibleNode = level2
	root.RightAccessIdentifier = &Identifier{IdentifierName: "child"}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*DotNotation)
	if !ok {
		t.Fatalf("Output is not of type *DotNotation")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestMapAccess(t *testing.T) {
	expression := `$.map["key"]`

	// level2: $.map
	level2 := &DotNotation{}
	level2.LeftAccessibleNode = &Identifier{IdentifierName: "$"}
	level2.RightAccessIdentifier = &Identifier{IdentifierName: "map"}
	// root: <level2>.["key"]
	root := &BracketAccessor{}
	root.LeftNode = level2
	root.RightExpression = &StringLiteral{StrValue: "key"}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*BracketAccessor)
	if !ok {
		t.Fatalf("Output is not of type *BracketAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestDeepMapAccess(t *testing.T) {
	expression := `$.a.b[0].c["k"]`

	// level5: $.a
	level5 := &DotNotation{}
	level5.LeftAccessibleNode = &Identifier{IdentifierName: "$"}
	level5.RightAccessIdentifier = &Identifier{IdentifierName: "a"}
	// level4: <level5>.b
	level4 := &DotNotation{}
	level4.LeftAccessibleNode = level5
	level4.RightAccessIdentifier = &Identifier{IdentifierName: "b"}
	// level3: <level4>[0]
	level3 := &BracketAccessor{}
	level3.LeftNode = level4
	level3.RightExpression = &IntLiteral{IntValue: 0}
	// level2: <level3>.c
	level2 := &DotNotation{}
	level2.LeftAccessibleNode = level3
	level2.RightAccessIdentifier = &Identifier{IdentifierName: "c"}
	// root: <level2>["k"]
	root := &BracketAccessor{}
	root.LeftNode = level2
	root.RightExpression = &StringLiteral{StrValue: "k"}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*BracketAccessor)
	if !ok {
		t.Fatalf("Output is not of type *BracketAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestCompound(t *testing.T) {
	expression := `$.a.b.c["key"].d`

	// level5: $.a
	level5 := &DotNotation{}
	level5.LeftAccessibleNode = &Identifier{IdentifierName: "$"}
	level5.RightAccessIdentifier = &Identifier{IdentifierName: "a"}
	// level4: <level5>.b
	level4 := &DotNotation{}
	level4.LeftAccessibleNode = level5
	level4.RightAccessIdentifier = &Identifier{IdentifierName: "b"}
	// level3: <level4>.c
	level3 := &DotNotation{}
	level3.LeftAccessibleNode = level4
	level3.RightAccessIdentifier = &Identifier{IdentifierName: "c"}
	// level2: <level3>["key"]
	level2 := &BracketAccessor{}
	level2.LeftNode = level3
	level2.RightExpression = &StringLiteral{StrValue: "key"}
	// root: <level2>.d
	root := &DotNotation{}
	root.LeftAccessibleNode = level2
	root.RightAccessIdentifier = &Identifier{IdentifierName: "d"}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*DotNotation)
	if !ok {
		t.Fatalf("Output is not of type *DotNotation")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestAllBracketNotation(t *testing.T) {
	expression := `$["a"]["b"][0]["c"]`

	level4 := &BracketAccessor{}
	level4.LeftNode = &Identifier{"$"}
	level4.RightExpression = &StringLiteral{"a"}
	level3 := &BracketAccessor{}
	level3.LeftNode = level4
	level3.RightExpression = &StringLiteral{"b"}
	level2 := &BracketAccessor{}
	level2.LeftNode = level3
	level2.RightExpression = &IntLiteral{0}
	root := &BracketAccessor{}
	root.LeftNode = level2
	root.RightExpression = &StringLiteral{"c"}
	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*BracketAccessor)
	if !ok {
		t.Fatalf("Output is not of type *BracketAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestEmptyExpression(t *testing.T) {
	expression := ""

	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestMapWithSingleQuotes(t *testing.T) {
	expression := "$['a']"

	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	resultAsString := parsedResult.String()
	resultAsString = strings.ReplaceAll(resultAsString, "\"", "'")
	assert.Equals(t, expression, resultAsString)
}

func TestSubExpression(t *testing.T) {
	expression := "$[$.a]"

	right := &DotNotation{}
	right.LeftAccessibleNode = &Identifier{IdentifierName: "$"}
	right.RightAccessIdentifier = &Identifier{IdentifierName: "a"}
	root := &BracketAccessor{}
	root.LeftNode = &Identifier{IdentifierName: "$"}
	root.RightExpression = right

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*BracketAccessor)
	if !ok {
		t.Fatalf("Output is not of type *BracketAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestEmptyFunctionExpression(t *testing.T) {
	expression := "funcName()"

	root := &FunctionCall{
		FuncIdentifier: &Identifier{IdentifierName: "funcName"},
		ArgumentInputs: &ArgumentList{Arguments: make([]Node, 0)},
	}

	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*FunctionCall)
	if !ok {
		t.Fatalf("Output is not of type *FunctionCall")
	}
	assert.Equals(t, parsedRoot, root)
}
func TestOneArgFunctionExpression(t *testing.T) {
	expression := "funcName($.a)"

	arg1 := &DotNotation{}
	arg1.LeftAccessibleNode = &Identifier{IdentifierName: "$"}
	arg1.RightAccessIdentifier = &Identifier{IdentifierName: "a"}
	root := &FunctionCall{
		FuncIdentifier: &Identifier{IdentifierName: "funcName"},
		ArgumentInputs: &ArgumentList{Arguments: []Node{arg1}},
	}

	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*FunctionCall)
	if !ok {
		t.Fatalf("Output is not of type *FunctionCall")
	}
	assert.Equals(t, parsedRoot, root)
}
func TestMultiArgFunctionExpression(t *testing.T) {
	expression := `funcName($.a, 5, "test")`
	arg1 := &DotNotation{}
	arg1.LeftAccessibleNode = &Identifier{IdentifierName: "$"}
	arg1.RightAccessIdentifier = &Identifier{IdentifierName: "a"}
	arg2 := &IntLiteral{IntValue: 5}
	arg3 := &StringLiteral{StrValue: "test"}
	root := &FunctionCall{
		FuncIdentifier: &Identifier{IdentifierName: "funcName"},
		ArgumentInputs: &ArgumentList{Arguments: []Node{arg1, arg2, arg3}},
	}

	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*FunctionCall)
	if !ok {
		t.Fatalf("Output is not of type *FunctionCall")
	}
	assert.Equals(t, parsedRoot, root)
}
func TestChainedFunctionExpression(t *testing.T) {
	expression := "funcName().a"

	functionCall := &FunctionCall{
		FuncIdentifier: &Identifier{IdentifierName: "funcName"},
		ArgumentInputs: &ArgumentList{Arguments: make([]Node, 0)},
	}
	root := &DotNotation{
		LeftAccessibleNode:    functionCall,
		RightAccessIdentifier: &Identifier{IdentifierName: "a"},
	}

	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*DotNotation)
	if !ok {
		t.Fatalf("Output is not of type *DotNotation")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestExpressionInvalidStart(t *testing.T) {
	expression := "()"

	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidNonRoot(t *testing.T) {
	expression := "$.$"

	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidObjectAccess(t *testing.T) {
	expression := "@.a"

	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidMapAccessGrammar(t *testing.T) {
	expression := "$[)]" // Invalid due to the )

	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidDotNotationGrammar(t *testing.T) {
	expression := "$)a" // Invalid due to the )

	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidIdentifier(t *testing.T) {
	expression := "$.(" // invalid due to the (

	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpression_SimpleAdd(t *testing.T) {
	expression := "2 + 2"

	// 2 + 2 as tree
	root := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 2},
		RightNode: &IntLiteral{IntValue: 2},
		Operation: Add,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals(t, parsedResult.(*BinaryOperation), root)
}

func TestExpression_ThreeSub(t *testing.T) {
	expression := "1.0 - 2.0 - 3.0"

	// 1.0 - 2.0 - 3.0 as tree
	//     -
	//    / \
	//   -   3.0
	//  / \
	// 1.0    2.0
	level2 := &BinaryOperation{
		LeftNode:  &FloatLiteral{FloatValue: 1.0},
		RightNode: &FloatLiteral{FloatValue: 2.0},
		Operation: Subtract,
	}
	root := &BinaryOperation{
		LeftNode:  level2,
		RightNode: &FloatLiteral{FloatValue: 3.0},
		Operation: Subtract,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_MixedAddMultiplicationDivision(t *testing.T) {
	expression := "7 + 50 * 6 / 10"

	// 7 + 50 * 6 / 10 as tree
	//       +
	//      / \
	//     ÷   7
	//    / \
	//   *   10
	//  / \
	// 50  6
	level3 := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 50},
		RightNode: &IntLiteral{IntValue: 6},
		Operation: Multiply,
	}
	level2 := &BinaryOperation{
		LeftNode:  level3,
		RightNode: &IntLiteral{IntValue: 10},
		Operation: Divide,
	}
	root := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 7},
		RightNode: level2,
		Operation: Add,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_Power(t *testing.T) {
	expression := "1 ^ 4 * 3"

	// 1 ^ 4 * 3 as tree
	//     *
	//    / \
	//   ^   3
	//  / \
	// 1    4
	level2 := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 1},
		RightNode: &IntLiteral{IntValue: 4},
		Operation: Power,
	}
	root := &BinaryOperation{
		LeftNode:  level2,
		RightNode: &IntLiteral{IntValue: 3},
		Operation: Multiply,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_PowerParentheses(t *testing.T) {
	expression := "2 ^ (4 * 3)"

	// 2 ^ 4 * 3 as tree
	//     ^
	//    / \
	//   2   *
	//      / \
	//     4    3
	level2 := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 4},
		RightNode: &IntLiteral{IntValue: 3},
		Operation: Multiply,
	}
	root := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 2},
		RightNode: level2,
		Operation: Power,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_Parentheses(t *testing.T) {
	expression := "(4 + 3) * 2"

	// (4 + 3) * 2 as tree
	//     *
	//    / \
	//   +   2
	//  / \
	// 4    3
	level2 := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 4},
		RightNode: &IntLiteral{IntValue: 3},
		Operation: Add,
	}
	root := &BinaryOperation{
		LeftNode:  level2,
		RightNode: &IntLiteral{IntValue: 2},
		Operation: Multiply,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_UnaryNegative(t *testing.T) {
	expression := "5 + -5"

	// 5 + -5 as tree
	//     +
	//    / \
	//   5   -
	//       |
	//       5
	level2 := &UnaryOperation{
		LeftOperation: Subtract,
		RightNode:     &IntLiteral{IntValue: 5},
	}
	root := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 5},
		RightNode: level2,
		Operation: Add,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_MultiNegationUnary(t *testing.T) {
	expression := `---5`
	level3 := &UnaryOperation{
		LeftOperation: Subtract,
		RightNode:     &IntLiteral{IntValue: 5},
	}
	level2 := &UnaryOperation{
		LeftOperation: Subtract,
		RightNode:     level3,
	}
	root := &UnaryOperation{
		LeftOperation: Subtract,
		RightNode:     level2,
	}
	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*UnaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_MultiNegationUnaryParentheses(t *testing.T) {
	expression := `-(-5)`
	level2 := &UnaryOperation{
		LeftOperation: Subtract,
		RightNode:     &IntLiteral{IntValue: 5},
	}
	root := &UnaryOperation{
		LeftOperation: Subtract,
		RightNode:     level2,
	}
	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*UnaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_MultiNotUnary(t *testing.T) {
	expression := `!!true`
	level2 := &UnaryOperation{
		LeftOperation: Not,
		RightNode:     &BooleanLiteral{BooleanValue: true},
	}
	root := &UnaryOperation{
		LeftOperation: Not,
		RightNode:     level2,
	}
	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*UnaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

// In the binary operator grammar tests, not all operators are tested
// in every scenario because not every operator has its own code path.
// The per-operator tests are done in expression_evaluate_test.go

func TestExpression_SimpleComparison(t *testing.T) {
	expression := "2 > 2"

	// 2 > 2 as tree
	//   >
	//  / \
	// 2   2
	root := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 2},
		RightNode: &IntLiteral{IntValue: 2},
		Operation: GreaterThan,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_SimpleComparisonTwoToken(t *testing.T) {
	expression := "2 >= 2"

	// 2 >= 2 as tree
	//  >=
	//  / \
	// 2   2
	root := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 2},
		RightNode: &IntLiteral{IntValue: 2},
		Operation: GreaterThanEqualTo,
	}
	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_ErrIncorrectEquals(t *testing.T) {
	// In this test, we ensure that it properly rejects a single equals. A double equals is required.
	expression := "2 = 2"

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	_, err = p.ParseExpression()

	assert.Error(t, err)
	var grammarErr *InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []TokenID{EqualsToken})
}

func TestExpression_MixedComparisons(t *testing.T) {
	expression := "0 < 1 + 2"

	// 0 < 1 + 2 as tree
	//     <
	//    / \
	//   0   +
	//      / \
	//     1   2
	level2 := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 1},
		RightNode: &IntLiteral{IntValue: 2},
		Operation: Add,
	}
	root := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 0},
		RightNode: level2,
		Operation: LessThan,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}
func TestExpression_AndLogic(t *testing.T) {
	expression := "true && false"

	// true && false as tree
	//     &&
	//    /  \
	//  true  false
	root := &BinaryOperation{
		LeftNode:  &BooleanLiteral{BooleanValue: true},
		RightNode: &BooleanLiteral{BooleanValue: false},
		Operation: And,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	assert.Equals[Node](t, parsedResult, root)
}

func TestExpression_AllTypes(t *testing.T) {
	expression := "2 * 3 + 4 > 2 % 5 || $.test && !true"

	// 2 * 3 + 4 > 2 % 5 || $.test && !true as tree
	//                 ||
	//             /        \
	//           >            &&
	//        /     \        /    \
	//       +       %    $.test   !
	//     /  \     / \            |
	//    *    4   2   5          true
	//   / \
	//  2   3
	multiplicationNode := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 2},
		RightNode: &IntLiteral{IntValue: 3},
		Operation: Multiply,
	}
	addNode := &BinaryOperation{
		LeftNode:  multiplicationNode,
		RightNode: &IntLiteral{IntValue: 4},
		Operation: Add,
	}
	modNode := &BinaryOperation{
		LeftNode:  &IntLiteral{IntValue: 2},
		RightNode: &IntLiteral{IntValue: 5},
		Operation: Modulus,
	}
	greaterThanNode := &BinaryOperation{
		LeftNode:  addNode,
		RightNode: modNode,
		Operation: GreaterThan,
	}
	notNode := &UnaryOperation{
		LeftOperation: Not,
		RightNode:     &BooleanLiteral{BooleanValue: true},
	}
	andNode := &BinaryOperation{
		LeftNode:  &Identifier{IdentifierName: "$.test"},
		RightNode: notNode,
		Operation: And,
	}
	root := &BinaryOperation{
		LeftNode:  greaterThanNode,
		RightNode: andNode,
		Operation: Or,
	}

	// Create parser
	p, err := InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*BinaryOperation](t, parsedResult)
	// For some reason, comparing the raw results was failing falsely.
	assert.Equals(t, parsedResult.String(), root.String())
}

// Test unexpected tokens
// This is specifically targeted for places where a specific token is always expected,
// which is where .eat is called.
func TestExpression_MismatchedPair(t *testing.T) {
	bracketAccessExpr := "$.test[5)"
	// Create parser
	p, err := InitParser(bracketAccessExpr, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	_, err = p.ParseExpression()

	assert.Error(t, err)
	var grammarErr *InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []TokenID{BracketAccessDelimiterEndToken})

	funcExpr := "5 * (5 * 5]"
	// Create parser
	p, err = InitParser(funcExpr, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	_, err = p.ParseExpression()

	assert.Error(t, err)
	ok = errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []TokenID{ParenthesesEndToken})
}

func TestExpressionErrorChainLiteral(t *testing.T) {
	expression := `"a".a`
	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dot notation cannot follow a literal")
}

func TestParseArgs_badStart(t *testing.T) {
	expression := `))`
	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.advanceToken()
	assert.NoError(t, err)
	_, err = p.parseArgs()
	assert.Error(t, err)
	var grammarErr *InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []TokenID{ParenthesesStartToken})
}

func TestParseArgs_badEnd1(t *testing.T) {
	// This end will test using an open parentheses instead of a close parentheses
	// This will end up making it recurse into a second call to parseArgs, where it then expects it to be closed.
	expression := `(""(`
	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.advanceToken()
	assert.NoError(t, err)
	_, err = p.parseArgs()
	assert.Error(t, err)
	var grammarErr *InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []TokenID{ParenthesesEndToken})
}

func TestParseArgs_badEnd2(t *testing.T) {
	// This end will test a missing close parentheses.
	// This will create a nil value for nextToken that must be handled properly.
	expression := `(""`
	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.advanceToken()
	assert.NoError(t, err)
	_, err = p.parseArgs()
	assert.Error(t, err)
	var grammarErr *InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []TokenID{ParenthesesEndToken})
}

func TestParseArgs_badSeparator(t *testing.T) {
	// This end will test a missing close parentheses.
	// This will create a nil value for nextToken that must be handled properly.
	expression := `(""1`
	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.advanceToken()
	assert.NoError(t, err)
	_, err = p.parseArgs()
	assert.Error(t, err)
	var grammarErr *InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []TokenID{ListSeparatorToken, ParenthesesEndToken})
}

func TestParseArgs_badFirstToken(t *testing.T) {
	// This end will test a missing close parentheses.
	// This will create a nil value for nextToken that must be handled properly.
	expression := `1`
	p, err := InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.advanceToken()
	assert.NoError(t, err)
	_, err = p.parseArgs()
	assert.Error(t, err)
	var grammarErr *InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []TokenID{ParenthesesStartToken})
}
