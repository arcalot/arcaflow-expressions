package ast

import (
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
