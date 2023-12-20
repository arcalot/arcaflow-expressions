package ast

import (
	"errors"
	"testing"

	"go.arcalot.io/assert"
)

var filename = "example.go"

func TestTokenizer(t *testing.T) {
	input := `$.steps.read_kubeconfig.output["success"].credentials[f(1,2)]`
	tokenizer := initTokenizer(input, filename)
	expectedValue := []TokenValue{
		{"$", RootAccessToken, filename, 1, 1},
		{".", DotObjectAccessToken, filename, 1, 2},
		{"steps", IdentifierToken, filename, 1, 3},
		{".", DotObjectAccessToken, filename, 1, 8},
		{"read_kubeconfig", IdentifierToken, filename, 1, 9},
		{".", DotObjectAccessToken, filename, 1, 24},
		{"output", IdentifierToken, filename, 1, 25},
		{"[", BracketAccessDelimiterStartToken, filename, 1, 31},
		{"\"success\"", StringLiteralToken, filename, 1, 32},
		{"]", BracketAccessDelimiterEndToken, filename, 1, 41},
		{".", DotObjectAccessToken, filename, 1, 42},
		{"credentials", IdentifierToken, filename, 1, 43},
		{"[", BracketAccessDelimiterStartToken, filename, 1, 54},
		{"f", IdentifierToken, filename, 1, 55},
		{"(", ParenthesesStartToken, filename, 1, 56},
		{"1", IntLiteralToken, filename, 1, 57},
		{",", ListSeparatorToken, filename, 1, 58},
		{"2", IntLiteralToken, filename, 1, 59},
		{")", ParenthesesEndToken, filename, 1, 60},
		{"]", BracketAccessDelimiterEndToken, filename, 1, 61},
	}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.hasNextToken(), true)
		nextToken, err := tokenizer.getNext()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected.Value)
		assert.Equals(t, nextToken.TokenID, expected.TokenID)
		assert.Equals(t, nextToken.Filename, expected.Filename)
		assert.Equals(t, nextToken.Line, expected.Line)
		assert.Equals(t, nextToken.Column, expected.Column)
	}
}

func TestTokenizerWithEscapedStr(t *testing.T) {
	input := `$.output["ab\"|cd"]`
	tokenizer := initTokenizer(input, filename)
	expectedValue := []string{"$", ".", "output", "[", `"ab\"|cd"`, "]"}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.hasNextToken(), true)
		nextToken, err := tokenizer.getNext()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected)
	}
}

func TestBinaryOperations(t *testing.T) {
	input := `5 + 5 / 1 >= 5^5`
	tokenizer := initTokenizer(input, filename)
	expectedValue := []TokenValue{
		{"5", IntLiteralToken, filename, 1, 1},
		{"+", PlusToken, filename, 1, 3},
		{"5", IntLiteralToken, filename, 1, 5},
		{"/", DivideToken, filename, 1, 7},
		{"1", IntLiteralToken, filename, 1, 9},
		{">", GreaterThanToken, filename, 1, 11},
		{"=", EqualsToken, filename, 1, 12},
		{"5", IntLiteralToken, filename, 1, 14},
		{"^", PowerToken, filename, 1, 15},
		{"5", IntLiteralToken, filename, 1, 16},
	}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.hasNextToken(), true)
		nextToken, err := tokenizer.getNext()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected.Value)
		assert.Equals(t, nextToken.TokenID, expected.TokenID)
		assert.Equals(t, nextToken.Filename, expected.Filename)
		assert.Equals(t, nextToken.Line, expected.Line)
		assert.Equals(t, nextToken.Column, expected.Column)
	}
}

func TestWithFilterType(t *testing.T) {
	input := "$.steps.foo.outputs[\"bar\"][?(@._type=='x')].a"
	tokenizer := initTokenizer(input, filename)
	expectedValue := []string{"$", ".", "steps", ".", "foo", ".", "outputs",
		"[", "\"bar\"", "]", "[", "?", "(", "@", ".", "_type", "=", "=", "'x'", ")", "]", ".", "a"}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.hasNextToken(), true)
		nextToken, err := tokenizer.getNext()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected)
	}
}

func TestInvalidToken(t *testing.T) {
	input := "[€"
	tokenizer := initTokenizer(input, filename)
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err := tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, BracketAccessDelimiterStartToken)
	assert.Equals(t, tokenVal.Value, "[")
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err = tokenizer.getNext()
	assert.Error(t, err)
	assert.Equals(t, tokenVal.TokenID, UnknownToken)
	assert.Equals(t, tokenVal.Value, "€")
	expectedError := &InvalidTokenError{}
	isCorrectErrType := errors.As(err, &expectedError)
	if !isCorrectErrType {
		t.Fatalf("Error is of incorrect type")
	}
	assert.Equals(t, expectedError.InvalidToken.Column, 2)
	assert.Equals(t, expectedError.InvalidToken.Line, 1)
	assert.Equals(t, expectedError.InvalidToken.Filename, filename)
	assert.Equals(t, expectedError.InvalidToken.Value, "€")
}

func TestIntLiteral(t *testing.T) {
	input := "70 07"
	tokenizer := initTokenizer(input, filename)
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err := tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, IntLiteralToken)
	assert.Equals(t, tokenVal.Value, "70")
	assert.Equals(t, tokenizer.hasNextToken(), true)
	// Numbers that start with 0 are interpreted as octal by the string tokenizer,
	// resulting in an error printed to stderr. It doesn't change the behavior.
	tokenVal, err = tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, IdentifierToken)
	assert.Equals(t, tokenVal.Value, "07")
}

func TestFloatLiteral(t *testing.T) {
	input := "0.0 40.099 05.00"
	tokenizer := initTokenizer(input, filename)
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err := tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, FloatLiteralToken)
	assert.Equals(t, tokenVal.Value, "0.0")
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err = tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, FloatLiteralToken)
	assert.Equals(t, tokenVal.Value, "40.099")
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err = tokenizer.getNext()
	assert.Error(t, err)
}
func TestBooleanLiterals(t *testing.T) {
	input := "true && false || false"
	tokenizer := initTokenizer(input, filename)
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err := tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, BooleanLiteralToken)
	assert.Equals(t, tokenVal.Value, "true")
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err = tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, AndToken)
	assert.Equals(t, tokenVal.Value, "&")
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err = tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, AndToken)
	assert.Equals(t, tokenVal.Value, "&")
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err = tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, BooleanLiteralToken)
	assert.Equals(t, tokenVal.Value, "false")
	tokenVal, err = tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, OrToken)
	assert.Equals(t, tokenVal.Value, "|")
	tokenVal, err = tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, OrToken)
	assert.Equals(t, tokenVal.Value, "|")
	tokenVal, err = tokenizer.getNext()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, BooleanLiteralToken)
	assert.Equals(t, tokenVal.Value, "false")
}

func TestWildcard(t *testing.T) {
	input := `$.*`
	tokenizer := initTokenizer(input, filename)
	expectedValue := []string{"$", ".", "*"}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.hasNextToken(), true)
		nextToken, err := tokenizer.getNext()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected)
	}
}
