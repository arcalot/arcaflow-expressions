package ast_test

import (
	"errors"
	"strings"
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions/internal/ast"
)

func TestIdentifierParser(t *testing.T) {
	identifierName := "abc"

	// Create parser
	p, err := ast.InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.AdvanceTokenForTest())

	identifierResult, err := p.ParseIdentifierForTest()

	assert.NoError(t, err)
	assert.Equals(t, identifierName, identifierResult.IdentifierName)

	// No tokens left, so should error out

	_, err = p.ParseIdentifierForTest()
	assert.Error(t, err)
}

func TestIdentifierParserInvalidToken(t *testing.T) {
	identifierName := "["

	// Create parser
	p, err := ast.InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.AdvanceTokenForTest())

	_, err = p.ParseIdentifierForTest()

	assert.Error(t, err)
}

// Test proper map access.
func TestMapAccessParser(t *testing.T) {
	expression := "[0]['a']"

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.AdvanceTokenForTest())

	mapResult, err := p.ParseBracketAccessForTest(&ast.Identifier{IdentifierName: "a"})

	assert.NoError(t, err)
	assert.Equals[ast.Node](t, mapResult.RightExpression, &ast.IntLiteral{IntValue: int64(0)})
	assert.InstanceOf[ast.ValueLiteral](t, mapResult.RightExpression)
	assert.Equals(t, mapResult.RightExpression.(ast.ValueLiteral).Value().(int64), int64(0))

	mapResult, err = p.ParseBracketAccessForTest(&ast.Identifier{IdentifierName: "a"})

	assert.NoError(t, err)
	assert.Equals[ast.Node](t, mapResult.RightExpression, &ast.StringLiteral{StrValue: "a"})
	assert.InstanceOf[ast.ValueLiteral](t, mapResult.RightExpression)
	assert.Equals(t, mapResult.RightExpression.(ast.ValueLiteral).Value(), "a")

	// Test left and right functions
	assert.Equals(t, mapResult.Right(), mapResult.RightExpression)
	assert.Equals(t, mapResult.Left(), mapResult.LeftNode)

	// No tokens left, so should error out

	_, err = p.ParseIdentifierForTest()
	assert.Error(t, err)
}

// Test invalid param.
func TestParseBracketAccessInvalidParam(t *testing.T) {
	identifierName := "[0]"

	// Create parser
	p, err := ast.InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.AdvanceTokenForTest())

	_, err = p.ParseBracketAccessForTest(nil)

	assert.Error(t, err)
}

// Test invalid input.
func TestParseBracketAccessInvalidPrefixToken(t *testing.T) {
	identifierName := "]0]"

	// Create parser
	p, err := ast.InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.AdvanceTokenForTest())

	_, err = p.ParseBracketAccessForTest(&ast.Identifier{IdentifierName: "a"})

	assert.Error(t, err)
}

func TestParseBracketAccessInvalidKey(t *testing.T) {
	identifierName := "[0["

	// Create parser
	p, err := ast.InitParser(identifierName, t.Name())

	assert.NoError(t, err)
	assert.NoError(t, p.AdvanceTokenForTest())

	_, err = p.ParseBracketAccessForTest(&ast.Identifier{IdentifierName: "a"})

	assert.Error(t, err)
}

// Test cases that test the entire parser.
func TestRootVar(t *testing.T) {
	expression := "$.test"

	// $.test
	root := &ast.DotNotation{}
	root.LeftAccessibleNode = &ast.Identifier{IdentifierName: "$"}
	root.RightAccessIdentifier = &ast.Identifier{IdentifierName: "test"}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*ast.DotNotation)
	if !ok {
		t.Fatalf("Output is not of type *DotNotation")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestRootStrLiteral(t *testing.T) {
	expression := `"test"`

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)
	assert.InstanceOf[*ast.StringLiteral](t, parsedResult)
	assert.Equals(t, parsedResult.(*ast.StringLiteral).StrValue, "test")
}

func TestDotNotation(t *testing.T) {
	expression := "$.parent.child"

	// level2: $.parent
	level2 := &ast.DotNotation{}
	level2.LeftAccessibleNode = &ast.Identifier{IdentifierName: "$"}
	level2.RightAccessIdentifier = &ast.Identifier{IdentifierName: "parent"}
	// root: <level2>.child
	root := &ast.DotNotation{}
	root.LeftAccessibleNode = level2
	root.RightAccessIdentifier = &ast.Identifier{IdentifierName: "child"}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*ast.DotNotation)
	if !ok {
		t.Fatalf("Output is not of type *DotNotation")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestMapAccess(t *testing.T) {
	expression := `$.map["key"]`

	// level2: $.map
	level2 := &ast.DotNotation{}
	level2.LeftAccessibleNode = &ast.Identifier{IdentifierName: "$"}
	level2.RightAccessIdentifier = &ast.Identifier{IdentifierName: "map"}
	// root: <level2>.["key"]
	root := &ast.BracketAccessor{}
	root.LeftNode = level2
	root.RightExpression = &ast.StringLiteral{StrValue: "key"}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*ast.BracketAccessor)
	if !ok {
		t.Fatalf("Output is not of type *BracketAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

// assertParseExpression is a helper that parses the given expression and verifies
// the result matches the expected AST node and its string representation.
func assertParseExpression(t *testing.T, expression string, expectedRoot ast.Node) {
	t.Helper()
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, expectedRoot.String())
	assert.Equals(t, parsedResult.String(), expectedRoot.String())
}

func TestDeepMapAccess(t *testing.T) {
	expression := `$.a.b[0].c["k"]`

	// level5: $.a
	level5 := &ast.DotNotation{}
	level5.LeftAccessibleNode = &ast.Identifier{IdentifierName: "$"}
	level5.RightAccessIdentifier = &ast.Identifier{IdentifierName: "a"}
	// level4: <level5>.b
	level4 := &ast.DotNotation{}
	level4.LeftAccessibleNode = level5
	level4.RightAccessIdentifier = &ast.Identifier{IdentifierName: "b"}
	// level3: <level4>[0]
	level3 := &ast.BracketAccessor{}
	level3.LeftNode = level4
	level3.RightExpression = &ast.IntLiteral{IntValue: 0}
	// level2: <level3>.c
	level2 := &ast.DotNotation{}
	level2.LeftAccessibleNode = level3
	level2.RightAccessIdentifier = &ast.Identifier{IdentifierName: "c"}
	// root: <level2>["k"]
	root := &ast.BracketAccessor{}
	root.LeftNode = level2
	root.RightExpression = &ast.StringLiteral{StrValue: "k"}

	assertParseExpression(t, expression, root)
}

func TestCompound(t *testing.T) {
	expression := `$.a.b.c["key"].d`

	// level5: $.a
	level5 := &ast.DotNotation{}
	level5.LeftAccessibleNode = &ast.Identifier{IdentifierName: "$"}
	level5.RightAccessIdentifier = &ast.Identifier{IdentifierName: "a"}
	// level4: <level5>.b
	level4 := &ast.DotNotation{}
	level4.LeftAccessibleNode = level5
	level4.RightAccessIdentifier = &ast.Identifier{IdentifierName: "b"}
	// level3: <level4>.c
	level3 := &ast.DotNotation{}
	level3.LeftAccessibleNode = level4
	level3.RightAccessIdentifier = &ast.Identifier{IdentifierName: "c"}
	// level2: <level3>["key"]
	level2 := &ast.BracketAccessor{}
	level2.LeftNode = level3
	level2.RightExpression = &ast.StringLiteral{StrValue: "key"}
	// root: <level2>.d
	root := &ast.DotNotation{}
	root.LeftAccessibleNode = level2
	root.RightAccessIdentifier = &ast.Identifier{IdentifierName: "d"}

	assertParseExpression(t, expression, root)
}

func TestAllBracketNotation(t *testing.T) {
	expression := `$["a"]["b"][0]["c"]`

	level4 := &ast.BracketAccessor{}
	level4.LeftNode = &ast.Identifier{IdentifierName: "$"}
	level4.RightExpression = &ast.StringLiteral{StrValue: "a"}
	level3 := &ast.BracketAccessor{}
	level3.LeftNode = level4
	level3.RightExpression = &ast.StringLiteral{StrValue: "b"}
	level2 := &ast.BracketAccessor{}
	level2.LeftNode = level3
	level2.RightExpression = &ast.IntLiteral{IntValue: 0}
	root := &ast.BracketAccessor{}
	root.LeftNode = level2
	root.RightExpression = &ast.StringLiteral{StrValue: "c"}
	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*ast.BracketAccessor)
	if !ok {
		t.Fatalf("Output is not of type *BracketAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestEmptyExpression(t *testing.T) {
	expression := ""

	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestMapWithSingleQuotes(t *testing.T) {
	expression := "$['a']"

	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	resultAsString := parsedResult.String()
	resultAsString = strings.ReplaceAll(resultAsString, "\"", "'")
	assert.Equals(t, expression, resultAsString)
}

func TestSubExpression(t *testing.T) {
	expression := "$[$.a]"

	right := &ast.DotNotation{}
	right.LeftAccessibleNode = &ast.Identifier{IdentifierName: "$"}
	right.RightAccessIdentifier = &ast.Identifier{IdentifierName: "a"}
	root := &ast.BracketAccessor{}
	root.LeftNode = &ast.Identifier{IdentifierName: "$"}
	root.RightExpression = right

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*ast.BracketAccessor)
	if !ok {
		t.Fatalf("Output is not of type *BracketAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestParseExpression_Error_TrailingDot(t *testing.T) {
	// Test error message for dot notation after literal.
	expression := "$.a."
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	_, err = p.ParseExpression()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `expected token "identifier"`)
}

func TestParseExpression_Error_BracketAfterLiteral(t *testing.T) {
	// Test error message for bracket access after literal
	expression := "0[0]"
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	_, err = p.ParseExpression()
	assert.Error(t, err)
	assert.Equals(t, err.Error(), `bracket access cannot follow a literal; got "[" after "0"`)
}

func TestParseExpression_Error_DotAfterLiteral(t *testing.T) {
	// Test error message for dot notation after literal.
	expression := "0 .a" // The space is needed for the tokens to be separated for the behavior we're testing.
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	_, err = p.ParseExpression()
	assert.Error(t, err)
	assert.Equals(t, err.Error(), `dot notation cannot follow a literal; got "." after "0"`)
}

func TestParseExpression_Error_ParenthesesAfterLiteral(t *testing.T) {
	// Test the error message for function call after literal.
	expression := "0(0 + 0)"
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()
	assert.Error(t, err)
	assert.Nil(t, parsedResult)
	assert.Contains(t, err.Error(), "an opening parentheses cannot follow a literal")
}

func TestEmptyFunctionExpression(t *testing.T) {
	expression := "funcName()"

	root := &ast.FunctionCall{
		FuncIdentifier: &ast.Identifier{IdentifierName: "funcName"},
		ArgumentInputs: &ast.ArgumentList{Arguments: make([]ast.Node, 0)},
	}

	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*ast.FunctionCall)
	if !ok {
		t.Fatalf("Output is not of type *FunctionCall")
	}
	assert.Equals(t, parsedRoot, root)
}
func TestOneArgFunctionExpression(t *testing.T) {
	expression := "funcName($.a)"

	arg1 := &ast.DotNotation{}
	arg1.LeftAccessibleNode = &ast.Identifier{IdentifierName: "$"}
	arg1.RightAccessIdentifier = &ast.Identifier{IdentifierName: "a"}
	root := &ast.FunctionCall{
		FuncIdentifier: &ast.Identifier{IdentifierName: "funcName"},
		ArgumentInputs: &ast.ArgumentList{Arguments: []ast.Node{arg1}},
	}

	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*ast.FunctionCall)
	if !ok {
		t.Fatalf("Output is not of type *FunctionCall")
	}
	assert.Equals(t, parsedRoot, root)
}
func TestMultiArgFunctionExpression(t *testing.T) {
	expression := `funcName($.a, 5, "test")`
	arg1 := &ast.DotNotation{}
	arg1.LeftAccessibleNode = &ast.Identifier{IdentifierName: "$"}
	arg1.RightAccessIdentifier = &ast.Identifier{IdentifierName: "a"}
	arg2 := &ast.IntLiteral{IntValue: 5}
	arg3 := &ast.StringLiteral{StrValue: "test"}
	root := &ast.FunctionCall{
		FuncIdentifier: &ast.Identifier{IdentifierName: "funcName"},
		ArgumentInputs: &ast.ArgumentList{Arguments: []ast.Node{arg1, arg2, arg3}},
	}

	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*ast.FunctionCall)
	if !ok {
		t.Fatalf("Output is not of type *FunctionCall")
	}
	assert.Equals(t, parsedRoot, root)
}
func TestChainedFunctionExpression(t *testing.T) {
	expression := "funcName().a"

	functionCall := &ast.FunctionCall{
		FuncIdentifier: &ast.Identifier{IdentifierName: "funcName"},
		ArgumentInputs: &ast.ArgumentList{Arguments: make([]ast.Node, 0)},
	}
	root := &ast.DotNotation{
		LeftAccessibleNode:    functionCall,
		RightAccessIdentifier: &ast.Identifier{IdentifierName: "a"},
	}

	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*ast.DotNotation)
	if !ok {
		t.Fatalf("Output is not of type *DotNotation")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestExpressionInvalidStart(t *testing.T) {
	expression := "()"

	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidNonRoot(t *testing.T) {
	expression := "$.$"

	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidObjectAccess(t *testing.T) {
	expression := "@.a"

	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidMapAccessGrammar(t *testing.T) {
	expression := "$[)]" // Invalid due to the )

	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidDotNotationGrammar(t *testing.T) {
	expression := "$)a" // Invalid due to the )

	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpressionInvalidIdentifier(t *testing.T) {
	expression := "$.(" // invalid due to the (

	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
}

func TestExpression_SimpleAdd(t *testing.T) {
	expression := "2 + 2"

	// 2 + 2 as tree
	root := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 2},
		RightNode: &ast.IntLiteral{IntValue: 2},
		Operation: ast.Add,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals(t, parsedResult.(*ast.BinaryOperation), root)
}

func TestExpression_ThreeSub(t *testing.T) {
	expression := "1.0 - 2.0 - 3.0"

	// 1.0 - 2.0 - 3.0 as tree
	//     -
	//    / \
	//   -   3.0
	//  / \
	// 1.0    2.0
	level2 := &ast.BinaryOperation{
		LeftNode:  &ast.FloatLiteral{FloatValue: 1.0},
		RightNode: &ast.FloatLiteral{FloatValue: 2.0},
		Operation: ast.Subtract,
	}
	root := &ast.BinaryOperation{
		LeftNode:  level2,
		RightNode: &ast.FloatLiteral{FloatValue: 3.0},
		Operation: ast.Subtract,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
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
	level3 := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 50},
		RightNode: &ast.IntLiteral{IntValue: 6},
		Operation: ast.Multiply,
	}
	level2 := &ast.BinaryOperation{
		LeftNode:  level3,
		RightNode: &ast.IntLiteral{IntValue: 10},
		Operation: ast.Divide,
	}
	root := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 7},
		RightNode: level2,
		Operation: ast.Add,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}

func TestExpression_Power(t *testing.T) {
	expression := "1 ^ 4 * 3"

	// 1 ^ 4 * 3 as tree
	//     *
	//    / \
	//   ^   3
	//  / \
	// 1    4
	level2 := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 1},
		RightNode: &ast.IntLiteral{IntValue: 4},
		Operation: ast.Power,
	}
	root := &ast.BinaryOperation{
		LeftNode:  level2,
		RightNode: &ast.IntLiteral{IntValue: 3},
		Operation: ast.Multiply,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}

func TestExpression_PowerParentheses(t *testing.T) {
	expression := "2 ^ (4 * 3)"

	// 2 ^ 4 * 3 as tree
	//     ^
	//    / \
	//   2   *
	//      / \
	//     4    3
	level2 := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 4},
		RightNode: &ast.IntLiteral{IntValue: 3},
		Operation: ast.Multiply,
	}
	root := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 2},
		RightNode: level2,
		Operation: ast.Power,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}

func TestExpression_Parentheses(t *testing.T) {
	expression := "(4 + 3) * 2"

	// (4 + 3) * 2 as tree
	//     *
	//    / \
	//   +   2
	//  / \
	// 4    3
	level2 := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 4},
		RightNode: &ast.IntLiteral{IntValue: 3},
		Operation: ast.Add,
	}
	root := &ast.BinaryOperation{
		LeftNode:  level2,
		RightNode: &ast.IntLiteral{IntValue: 2},
		Operation: ast.Multiply,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}

func TestExpression_UnaryNegative(t *testing.T) {
	expression := "5 + -5"

	// 5 + -5 as tree
	//     +
	//    / \
	//   5   -
	//       |
	//       5
	level2 := &ast.UnaryOperation{
		LeftOperation: ast.Subtract,
		RightNode:     &ast.IntLiteral{IntValue: 5},
	}
	root := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 5},
		RightNode: level2,
		Operation: ast.Add,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}

func TestExpression_MultiNegationUnary(t *testing.T) {
	expression := `---5`
	level3 := &ast.UnaryOperation{
		LeftOperation: ast.Subtract,
		RightNode:     &ast.IntLiteral{IntValue: 5},
	}
	level2 := &ast.UnaryOperation{
		LeftOperation: ast.Subtract,
		RightNode:     level3,
	}
	root := &ast.UnaryOperation{
		LeftOperation: ast.Subtract,
		RightNode:     level2,
	}
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.UnaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}

func TestExpression_MultiNegationUnaryParentheses(t *testing.T) {
	expression := `-(-5)`
	level2 := &ast.UnaryOperation{
		LeftOperation: ast.Subtract,
		RightNode:     &ast.IntLiteral{IntValue: 5},
	}
	root := &ast.UnaryOperation{
		LeftOperation: ast.Subtract,
		RightNode:     level2,
	}
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.UnaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}

func TestExpression_MultiNotUnary(t *testing.T) {
	expression := `!!true`
	level2 := &ast.UnaryOperation{
		LeftOperation: ast.Not,
		RightNode:     &ast.BooleanLiteral{BooleanValue: true},
	}
	root := &ast.UnaryOperation{
		LeftOperation: ast.Not,
		RightNode:     level2,
	}
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.UnaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
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
	root := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 2},
		RightNode: &ast.IntLiteral{IntValue: 2},
		Operation: ast.GreaterThan,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}

func TestExpression_SimpleComparisonTwoToken(t *testing.T) {
	expression := "2 >= 2"

	// 2 >= 2 as tree
	//  >=
	//  / \
	// 2   2
	root := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 2},
		RightNode: &ast.IntLiteral{IntValue: 2},
		Operation: ast.GreaterThanEqualTo,
	}
	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}

func TestExpression_ErrIncorrectEquals(t *testing.T) {
	// In this test, we ensure that it properly rejects a single equals. A double equals is required.
	expression := "2 = 2"

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	_, err = p.ParseExpression()

	assert.Error(t, err)
	var grammarErr *ast.InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []ast.TokenID{ast.EqualsToken})
}

func TestExpression_MixedComparisons(t *testing.T) {
	expression := "0 < 1 + 2"

	// 0 < 1 + 2 as tree
	//     <
	//    / \
	//   0   +
	//      / \
	//     1   2
	level2 := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 1},
		RightNode: &ast.IntLiteral{IntValue: 2},
		Operation: ast.Add,
	}
	root := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 0},
		RightNode: level2,
		Operation: ast.LessThan,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
}
func TestExpression_AndLogic(t *testing.T) {
	expression := "true && false"

	// true && false as tree
	//     &&
	//    /  \
	//  true  false
	root := &ast.BinaryOperation{
		LeftNode:  &ast.BooleanLiteral{BooleanValue: true},
		RightNode: &ast.BooleanLiteral{BooleanValue: false},
		Operation: ast.And,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	assert.Equals[ast.Node](t, parsedResult, root)
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
	multiplicationNode := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 2},
		RightNode: &ast.IntLiteral{IntValue: 3},
		Operation: ast.Multiply,
	}
	addNode := &ast.BinaryOperation{
		LeftNode:  multiplicationNode,
		RightNode: &ast.IntLiteral{IntValue: 4},
		Operation: ast.Add,
	}
	modNode := &ast.BinaryOperation{
		LeftNode:  &ast.IntLiteral{IntValue: 2},
		RightNode: &ast.IntLiteral{IntValue: 5},
		Operation: ast.Modulus,
	}
	greaterThanNode := &ast.BinaryOperation{
		LeftNode:  addNode,
		RightNode: modNode,
		Operation: ast.GreaterThan,
	}
	notNode := &ast.UnaryOperation{
		LeftOperation: ast.Not,
		RightNode:     &ast.BooleanLiteral{BooleanValue: true},
	}
	andNode := &ast.BinaryOperation{
		LeftNode:  &ast.Identifier{IdentifierName: "$.test"},
		RightNode: notNode,
		Operation: ast.And,
	}
	root := &ast.BinaryOperation{
		LeftNode:  greaterThanNode,
		RightNode: andNode,
		Operation: ast.Or,
	}

	// Create parser
	p, err := ast.InitParser(expression, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.InstanceOf[*ast.BinaryOperation](t, parsedResult)
	// For some reason, comparing the raw results was failing falsely.
	assert.Equals(t, parsedResult.String(), root.String())
}

// Test unexpected tokens
// This is specifically targeted for places where a specific token is always expected,
// which is where .eat is called.
func TestExpression_MismatchedPair(t *testing.T) {
	bracketAccessExpr := "$.test[5)"
	// Create parser
	p, err := ast.InitParser(bracketAccessExpr, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	_, err = p.ParseExpression()

	assert.Error(t, err)
	var grammarErr *ast.InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []ast.TokenID{ast.BracketAccessDelimiterEndToken})

	funcExpr := "5 * (5 * 5]"
	// Create parser
	p, err = ast.InitParser(funcExpr, t.Name())

	assert.NoError(t, err)

	// Parse and validate
	_, err = p.ParseExpression()

	assert.Error(t, err)
	ok = errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []ast.TokenID{ast.ParenthesesEndToken})
}

func TestExpressionErrorChainLiteral(t *testing.T) {
	expression := `"a".a`
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dot notation cannot follow a literal")
}

func TestParseArgs_badStart(t *testing.T) {
	expression := `))`
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.AdvanceTokenForTest()
	assert.NoError(t, err)
	_, err = p.ParseArgsForTest()
	assert.Error(t, err)
	var grammarErr *ast.InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []ast.TokenID{ast.ParenthesesStartToken})
}

func TestParseArgs_badEnd2(t *testing.T) {
	// An incomplete argument list with a missing close parentheses.
	expression := `(""`
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.AdvanceTokenForTest()
	assert.NoError(t, err)
	_, err = p.ParseArgsForTest()
	assert.Error(t, err)
	var grammarErr *ast.InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []ast.TokenID{ast.ParenthesesEndToken, ast.ListSeparatorToken})
}

func TestParseArgs_badSeparator(t *testing.T) {
	// Testing a bad argument list
	expression := `(""1`
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.AdvanceTokenForTest()
	assert.NoError(t, err)
	_, err = p.ParseArgsForTest()
	assert.Error(t, err)
	var grammarErr *ast.InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []ast.TokenID{ast.ListSeparatorToken, ast.ParenthesesEndToken})
}

func TestParseArgs_endedAfterSeparator(t *testing.T) {
	// Test separator in place of close parentheses
	expression := `("",`
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.AdvanceTokenForTest()
	assert.NoError(t, err)
	_, err = p.ParseArgsForTest()
	assert.Error(t, err)
	var grammarErr *ast.InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []ast.TokenID{ast.ParenthesesEndToken})
}

func TestParseArgs_badFirstToken(t *testing.T) {
	// Test a missing open parentheses.
	// This is testing an edge-case that should not be hit if designed correctly.
	expression := `1`
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.AdvanceTokenForTest()
	assert.NoError(t, err)
	_, err = p.ParseArgsForTest()
	assert.Error(t, err)
	var grammarErr *ast.InvalidGrammarError
	ok := errors.As(err, &grammarErr)
	if !ok {
		t.Fatalf("Returned error is not InvalidGrammarError")
	}
	assert.Equals(t, grammarErr.ExpectedTokens, []ast.TokenID{ast.ParenthesesStartToken})
}

func TestParseString_EscapedStrings(t *testing.T) {
	expression := `"a\"b" "a\tb" "a\\b" "a\bb" "a\nb" "a\\nb" '\''`
	p, err := ast.InitParser(expression, t.Name())
	assert.NoError(t, err)
	err = p.AdvanceTokenForTest()
	assert.NoError(t, err)
	result, err := p.ParseStringLiteralForTest()
	assert.NoError(t, err)
	assert.Equals(t, result.StrValue, `a"b`)
	result, err = p.ParseStringLiteralForTest()
	assert.NoError(t, err)
	assert.Equals(t, result.StrValue, "a\tb")
	result, err = p.ParseStringLiteralForTest()
	assert.NoError(t, err)
	assert.Equals(t, result.StrValue, `a\b`)
	result, err = p.ParseStringLiteralForTest()
	assert.NoError(t, err)
	assert.Equals(t, result.StrValue, "a\bb")
	result, err = p.ParseStringLiteralForTest()
	assert.NoError(t, err)
	assert.Equals(t, result.StrValue, "a\nb")
	result, err = p.ParseStringLiteralForTest()
	assert.NoError(t, err)
	assert.Equals(t, result.StrValue, "a\\nb")
	result, err = p.ParseStringLiteralForTest()
	assert.NoError(t, err)
	assert.Equals(t, result.StrValue, "'")
}
