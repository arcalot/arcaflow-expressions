package ast

import (
	"errors"
	"testing"

	"go.arcalot.io/assert"
)

var filename = "example.go"

func TestTokenizer(t *testing.T) {
	input := `$.steps.read_kubeconfig.output["success"].credentials`
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
	input := "[&"
	tokenizer := initTokenizer(input, filename)
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err := tokenizer.getNext()
	assert.Nil(t, err)
	assert.Equals(t, tokenVal.TokenID, BracketAccessDelimiterStartToken)
	assert.Equals(t, tokenVal.Value, "[")
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err = tokenizer.getNext()
	assert.NotNil(t, err)
	assert.Equals(t, tokenVal.TokenID, UnknownToken)
	assert.Equals(t, tokenVal.Value, "&")
	expectedError := &InvalidTokenError{}
	isCorrectErrType := errors.As(err, &expectedError)
	if !isCorrectErrType {
		t.Fatalf("Error is of incorrect type")
	}
	assert.Equals(t, expectedError.InvalidToken.Column, 2)
	assert.Equals(t, expectedError.InvalidToken.Line, 1)
	assert.Equals(t, expectedError.InvalidToken.Filename, filename)
	assert.Equals(t, expectedError.InvalidToken.Value, "&")
}

func TestIntLiteral(t *testing.T) {
	input := "90 09"
	tokenizer := initTokenizer(input, filename)
	assert.Equals(t, tokenizer.hasNextToken(), true)
	tokenVal, err := tokenizer.getNext()
	assert.Nil(t, err)
	assert.Equals(t, tokenVal.TokenID, IntLiteralToken)
	assert.Equals(t, tokenVal.Value, "90")
	assert.Equals(t, tokenizer.hasNextToken(), true)
	// Numbers that start with 0 appear to cause error in scanner
	tokenVal, err = tokenizer.getNext()
	assert.Nil(t, err)
	assert.Equals(t, tokenVal.TokenID, IdentifierToken)
	assert.Equals(t, tokenVal.Value, "09")
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
