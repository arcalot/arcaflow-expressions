package ast

import (
	"strings"
	"testing"

	"go.arcalot.io/assert"
)

func TestIdentifierParser(t *testing.T) {
	identifierName := "abc"

	// Create parser
	p, err := InitParser(identifierName, "test.go")

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	identifierResult, err := p.parseIdentifier()

	assert.NoError(t, err)
	assert.Equals(t, identifierName, identifierResult.IdentifierName)

	// No tokens left, so should error out

	_, err = p.parseIdentifier()
	assert.NotNil(t, err)
}

func TestIdentifierParserInvalidToken(t *testing.T) {
	identifierName := "["

	// Create parser
	p, err := InitParser(identifierName, "test.go")

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	_, err = p.parseIdentifier()

	assert.NotNil(t, err)
}

// Test proper map access.
func TestMapAccessParser(t *testing.T) {
	expression := "[0]['a']"

	// Create parser
	p, err := InitParser(expression, "test.go")

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	mapResult, err := p.parseBracketAccess(&Identifier{IdentifierName: "a"})

	assert.NoError(t, err)
	assert.Equals(t, mapResult.RightKey, Key{Literal: &ASTIntLiteral{IntValue: 0}})
	assert.Equals(t, mapResult.RightKey.Literal.Value(), 0)
	assert.Equals(t, mapResult.RightKey.Left(), nil)
	assert.Equals(t, mapResult.RightKey.Right(), nil)

	mapResult, err = p.parseBracketAccess(&Identifier{IdentifierName: "a"})

	assert.NoError(t, err)
	assert.Equals(t, mapResult.RightKey, Key{Literal: &ASTStringLiteral{StrValue: "a"}})
	assert.Equals(t, mapResult.RightKey.Literal.Value(), "a")

	// Test left and right functions
	assert.Equals(t, mapResult.Right().(*Key), &mapResult.RightKey)
	assert.Equals(t, mapResult.Left(), mapResult.LeftNode)

	// No tokens left, so should error out

	_, err = p.parseIdentifier()
	assert.NotNil(t, err)
}

// Test invalid key.
func TestInvalidKey(t *testing.T) {
	blankKey := Key{}
	assert.Equals(t, blankKey.String(), "INVALID/MISSING")
}

// Test invalid param.
func TestParseBracketAccessInvalidParam(t *testing.T) {
	identifierName := "[0]"

	// Create parser
	p, err := InitParser(identifierName, "test.go")

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	_, err = p.parseBracketAccess(nil)

	assert.NotNil(t, err)
}

// Test invalid input.
func TestParseBracketAccessInvalidPrefixToken(t *testing.T) {
	identifierName := "]0]"

	// Create parser
	p, err := InitParser(identifierName, "test.go")

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	_, err = p.parseBracketAccess(&Identifier{IdentifierName: "a"})

	assert.NotNil(t, err)
}

func TestParseBracketAccessInvalidPostfixToken(t *testing.T) {
	identifierName := "[$]"

	// Create parser
	p, err := InitParser(identifierName, "test.go")

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	_, err = p.parseBracketAccess(&Identifier{IdentifierName: "a"})

	assert.NotNil(t, err)
}

func TestParseBracketAccessInvalidKey(t *testing.T) {
	identifierName := "[0["

	// Create parser
	p, err := InitParser(identifierName, "test.go")

	assert.NoError(t, err)
	assert.NoError(t, p.advanceToken())

	_, err = p.parseBracketAccess(&Identifier{IdentifierName: "a"})

	assert.NotNil(t, err)
}

// Test cases that test the entire parser.
func TestRootVar(t *testing.T) {
	expression := "$.test"

	// $.test
	root := &DotNotation{}
	root.LeftAccessableNode = &Identifier{IdentifierName: "$"}
	root.RightAccessIdentifier = &Identifier{IdentifierName: "test"}

	// Create parser
	p, err := InitParser(expression, "test.go")

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

func TestDotNotation(t *testing.T) {
	expression := "$.parent.child"

	// level2: $.parent
	level2 := &DotNotation{}
	level2.LeftAccessableNode = &Identifier{IdentifierName: "$"}
	level2.RightAccessIdentifier = &Identifier{IdentifierName: "parent"}
	// root: <level2>.child
	root := &DotNotation{}
	root.LeftAccessableNode = level2
	root.RightAccessIdentifier = &Identifier{IdentifierName: "child"}

	// Create parser
	p, err := InitParser(expression, "test.go")

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
	level2.LeftAccessableNode = &Identifier{IdentifierName: "$"}
	level2.RightAccessIdentifier = &Identifier{IdentifierName: "map"}
	// root: <level2>.["key"]
	root := &MapAccessor{}
	root.LeftNode = level2
	root.RightKey = Key{Literal: &ASTStringLiteral{StrValue: "key"}}

	// Create parser
	p, err := InitParser(expression, "test.go")

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*MapAccessor)
	if !ok {
		t.Fatalf("Output is not of type *MapAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestDeepMapAccess(t *testing.T) {
	expression := `$.a.b[0].c["k"]`

	// level5: $.a
	level5 := &DotNotation{}
	level5.LeftAccessableNode = &Identifier{IdentifierName: "$"}
	level5.RightAccessIdentifier = &Identifier{IdentifierName: "a"}
	// level4: <level5>.b
	level4 := &DotNotation{}
	level4.LeftAccessableNode = level5
	level4.RightAccessIdentifier = &Identifier{IdentifierName: "b"}
	// level3: <level4>[0]
	level3 := &MapAccessor{}
	level3.LeftNode = level4
	level3.RightKey = Key{Literal: &ASTIntLiteral{IntValue: 0}}
	// level2: <level3>.c
	level2 := &DotNotation{}
	level2.LeftAccessableNode = level3
	level2.RightAccessIdentifier = &Identifier{IdentifierName: "c"}
	// root: <level2>["k"]
	root := &MapAccessor{}
	root.LeftNode = level2
	root.RightKey = Key{Literal: &ASTStringLiteral{StrValue: "k"}}

	// Create parser
	p, err := InitParser(expression, "test.go")

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*MapAccessor)
	if !ok {
		t.Fatalf("Output is not of type *MapAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestCompound(t *testing.T) {
	expression := `$.a.b.c["key"].d`

	// level5: $.a
	level5 := &DotNotation{}
	level5.LeftAccessableNode = &Identifier{IdentifierName: "$"}
	level5.RightAccessIdentifier = &Identifier{IdentifierName: "a"}
	// level4: <level5>.b
	level4 := &DotNotation{}
	level4.LeftAccessableNode = level5
	level4.RightAccessIdentifier = &Identifier{IdentifierName: "b"}
	// level3: <level4>.c
	level3 := &DotNotation{}
	level3.LeftAccessableNode = level4
	level3.RightAccessIdentifier = &Identifier{IdentifierName: "c"}
	// level2: <level3>["key"]
	level2 := &MapAccessor{}
	level2.LeftNode = level3
	level2.RightKey = Key{Literal: &ASTStringLiteral{StrValue: "key"}}
	// root: <level2>.d
	root := &DotNotation{}
	root.LeftAccessableNode = level2
	root.RightAccessIdentifier = &Identifier{IdentifierName: "d"}

	// Create parser
	p, err := InitParser(expression, "test.go")

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

	level4 := &MapAccessor{}
	level4.LeftNode = &Identifier{"$"}
	level4.RightKey = Key{Literal: &ASTStringLiteral{"a"}}
	level3 := &MapAccessor{}
	level3.LeftNode = level4
	level3.RightKey = Key{Literal: &ASTStringLiteral{"b"}}
	level2 := &MapAccessor{}
	level2.LeftNode = level3
	level2.RightKey = Key{Literal: &ASTIntLiteral{0}}
	root := &MapAccessor{}
	root.LeftNode = level2
	root.RightKey = Key{Literal: &ASTStringLiteral{"c"}}
	// Create parser
	p, err := InitParser(expression, "test.go")

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*MapAccessor)
	if !ok {
		t.Fatalf("Output is not of type *MapAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestEmptyExpression(t *testing.T) {
	expression := ""

	p, err := InitParser(expression, "test.go")
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.NotNil(t, err)
}

func TestMapWithSingleQuotes(t *testing.T) {
	expression := "$['a']"

	p, err := InitParser(expression, "test.go")
	assert.NoError(t, err)
	parsedResult, err := p.ParseExpression()
	assert.NoError(t, err)
	resultAsString := parsedResult.String()
	resultAsString = strings.ReplaceAll(resultAsString, "\"", "'")
	assert.Equals(t, expression, resultAsString)
}

func TestSubExpression(t *testing.T) {
	expression := "$[($.a)]"

	right := &DotNotation{}
	right.LeftAccessableNode = &Identifier{IdentifierName: "$"}
	right.RightAccessIdentifier = &Identifier{IdentifierName: "a"}
	root := &MapAccessor{}
	root.LeftNode = &Identifier{IdentifierName: "$"}
	root.RightKey = Key{SubExpression: right}

	// Create parser
	p, err := InitParser(expression, "test.go")

	assert.NoError(t, err)

	parsedResult, err := p.ParseExpression()

	assert.NoError(t, err)
	assert.NotNil(t, parsedResult)

	assert.Equals(t, expression, root.String())

	parsedRoot, ok := parsedResult.(*MapAccessor)
	if !ok {
		t.Fatalf("Output is not of type *MapAccessor")
	}
	assert.Equals(t, parsedRoot, root)
}

func TestExpressionInvalidStart(t *testing.T) {
	expression := "()"

	p, err := InitParser(expression, "test.go")
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.NotNil(t, err)
}

func TestExpressionInvalidNonRoot(t *testing.T) {
	expression := "$.$"

	p, err := InitParser(expression, "test.go")
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.NotNil(t, err)
}

func TestExpressionInvalidObjectAccess(t *testing.T) {
	expression := "@.a"

	p, err := InitParser(expression, "test.go")
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.NotNil(t, err)
}

func TestExpressionInvalidMapAccessGrammar(t *testing.T) {
	expression := "$[)]" // Invalid due to the )

	p, err := InitParser(expression, "test.go")
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.NotNil(t, err)
}

func TestExpressionInvalidDotNotationGrammar(t *testing.T) {
	expression := "$)a" // Invalid due to the )

	p, err := InitParser(expression, "test.go")
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.NotNil(t, err)
}

func TestExpressionInvalidIdentifier(t *testing.T) {
	expression := "$.(" // invalid due to the (

	p, err := InitParser(expression, "test.go")
	assert.NoError(t, err)
	_, err = p.ParseExpression()
	assert.NotNil(t, err)
}
