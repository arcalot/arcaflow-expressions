package ast_test

import (
	"errors"
	"strings"
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions/internal/ast"
)

var filename = "example.go"

func TestTokenizer(t *testing.T) {
	// Include trailing whitespace to ensure that it has no ill effects.
	input := `$.steps.read_kubeconfig.output["success"].credentials[f(1,2)]   `
	tokenizer := ast.InitTokenizerForTest(input, filename)
	expectedValue := []ast.TokenValue{
		{"$", ast.RootAccessToken, filename, 1, 1},
		{".", ast.DotObjectAccessToken, filename, 1, 2},
		{"steps", ast.IdentifierToken, filename, 1, 3},
		{".", ast.DotObjectAccessToken, filename, 1, 8},
		{"read_kubeconfig", ast.IdentifierToken, filename, 1, 9},
		{".", ast.DotObjectAccessToken, filename, 1, 24},
		{"output", ast.IdentifierToken, filename, 1, 25},
		{"[", ast.BracketAccessDelimiterStartToken, filename, 1, 31},
		{"\"success\"", ast.StringLiteralToken, filename, 1, 32},
		{"]", ast.BracketAccessDelimiterEndToken, filename, 1, 41},
		{".", ast.DotObjectAccessToken, filename, 1, 42},
		{"credentials", ast.IdentifierToken, filename, 1, 43},
		{"[", ast.BracketAccessDelimiterStartToken, filename, 1, 54},
		{"f", ast.IdentifierToken, filename, 1, 55},
		{"(", ast.ParenthesesStartToken, filename, 1, 56},
		{"1", ast.IntLiteralToken, filename, 1, 57},
		{",", ast.ListSeparatorToken, filename, 1, 58},
		{"2", ast.IntLiteralToken, filename, 1, 59},
		{")", ast.ParenthesesEndToken, filename, 1, 60},
		{"]", ast.BracketAccessDelimiterEndToken, filename, 1, 61},
	}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
		nextToken, err := tokenizer.GetNextForTest()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected.Value)
		assert.Equals(t, nextToken.TokenID, expected.TokenID)
		assert.Equals(t, nextToken.Filename, expected.Filename)
		assert.Equals(t, nextToken.Line, expected.Line)
		assert.Equals(t, nextToken.Column, expected.Column)
	}
}

func TestTokenizer_TokenizerWithEscapedStr(t *testing.T) {
	input := `$.output["ab\"|cd"]`
	tokenizer := ast.InitTokenizerForTest(input, filename)
	expectedValue := []string{"$", ".", "output", "[", `"ab\"|cd"`, "]"}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
		nextToken, err := tokenizer.GetNextForTest()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected)
	}
}

func TestTokenizer_BinaryOperations(t *testing.T) {
	input := `5 + 5 / 1 >= 5^5`
	tokenizer := ast.InitTokenizerForTest(input, filename)
	expectedValue := []ast.TokenValue{
		{"5", ast.IntLiteralToken, filename, 1, 1},
		{"+", ast.PlusToken, filename, 1, 3},
		{"5", ast.IntLiteralToken, filename, 1, 5},
		{"/", ast.DivideToken, filename, 1, 7},
		{"1", ast.IntLiteralToken, filename, 1, 9},
		{">", ast.GreaterThanToken, filename, 1, 11},
		{"=", ast.EqualsToken, filename, 1, 12},
		{"5", ast.IntLiteralToken, filename, 1, 14},
		{"^", ast.PowerToken, filename, 1, 15},
		{"5", ast.IntLiteralToken, filename, 1, 16},
	}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
		nextToken, err := tokenizer.GetNextForTest()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected.Value)
		assert.Equals(t, nextToken.TokenID, expected.TokenID)
		assert.Equals(t, nextToken.Filename, expected.Filename)
		assert.Equals(t, nextToken.Line, expected.Line)
		assert.Equals(t, nextToken.Column, expected.Column)
	}
	assert.Equals(t, tokenizer.HasNextTokenForTest(), false)
}

func TestTokenizer_WithFilterType(t *testing.T) {
	input := "$.steps.foo.outputs[\"bar\"][?(@._type=='x')].a"
	tokenizer := ast.InitTokenizerForTest(input, filename)
	expectedValue := []string{"$", ".", "steps", ".", "foo", ".", "outputs",
		"[", "\"bar\"", "]", "[", "?", "(", "@", ".", "_type", "=", "=", "'x'", ")", "]", ".", "a"}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
		nextToken, err := tokenizer.GetNextForTest()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected)
	}
}

func TestTokenizer_InvalidToken(t *testing.T) {
	input := "[€"
	tokenizer := ast.InitTokenizerForTest(input, filename)
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err := tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.BracketAccessDelimiterStartToken)
	assert.Equals(t, tokenVal.Value, "[")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.Error(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.UnknownToken)
	assert.Equals(t, tokenVal.Value, "€")
	expectedError := &ast.InvalidTokenError{}
	isCorrectErrType := errors.As(err, &expectedError)
	if !isCorrectErrType {
		t.Fatalf("Error is of incorrect type")
	}
	assert.Equals(t, expectedError.InvalidToken.Column, 2)
	assert.Equals(t, expectedError.InvalidToken.Line, 1)
	assert.Equals(t, expectedError.InvalidToken.Filename, filename)
	assert.Equals(t, expectedError.InvalidToken.Value, "€")
}

func TestTokenizer_IntLiteral(t *testing.T) {
	input := "70 07"
	tokenizer := ast.InitTokenizerForTest(input, filename)
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err := tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.IntLiteralToken)
	assert.Equals(t, tokenVal.Value, "70")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	// Numbers that start with 0 are interpreted as octal by the string tokenizer,
	// resulting in an error printed to stderr. It doesn't change the behavior.
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.IdentifierToken)
	assert.Equals(t, tokenVal.Value, "07")
}

func TestTokenizer_FloatLiteral(t *testing.T) {
	input := "0.0 40.099 5.0e5 5.0E-5 05.00 5."
	tokenizer := ast.InitTokenizerForTest(input, filename)
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err := tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.FloatLiteralToken)
	assert.Equals(t, tokenVal.Value, "0.0")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.FloatLiteralToken)
	assert.Equals(t, tokenVal.Value, "40.099")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.FloatLiteralToken)
	assert.Equals(t, tokenVal.Value, "5.0e5")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.FloatLiteralToken)
	assert.Equals(t, tokenVal.Value, "5.0E-5")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.FloatLiteralToken)
	assert.Equals(t, tokenVal.Value, "05.00")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.FloatLiteralToken)
	assert.Equals(t, tokenVal.Value, "5.")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), false)

}

func TestTokenizer_BooleanLiterals(t *testing.T) {
	input := "true && false || false"
	tokenizer := ast.InitTokenizerForTest(input, filename)
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err := tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.BooleanLiteralToken)
	assert.Equals(t, tokenVal.Value, "true")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.AndToken)
	assert.Equals(t, tokenVal.Value, "&")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.AndToken)
	assert.Equals(t, tokenVal.Value, "&")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.BooleanLiteralToken)
	assert.Equals(t, tokenVal.Value, "false")
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.OrToken)
	assert.Equals(t, tokenVal.Value, "|")
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.OrToken)
	assert.Equals(t, tokenVal.Value, "|")
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.BooleanLiteralToken)
	assert.Equals(t, tokenVal.Value, "false")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), false)
}

func TestTokenizer_TokenPrefixSuffixInvalid(t *testing.T) {
	// This test checks that each of the input tokens are not recognized as boolean
	// literals (due to their substrings) by checking that they are recognized,
	// instead, as identifiers.
	input := "atrue truea atruea afalse falsea afalsea"
	expectedNumTokens := strings.Count(input, " ") + 1
	tokenizer := ast.InitTokenizerForTest(input, filename)
	actualTokens := 0
	for tokenizer.HasNextTokenForTest() {
		actualTokens++
		tokenVal, err := tokenizer.GetNextForTest()
		assert.NoError(t, err)
		assert.Equals(t, tokenVal.TokenID, ast.IdentifierToken)
	}
	assert.Equals(t, actualTokens, expectedNumTokens)
}

func TestTokenizer_StringLiteral(t *testing.T) {
	input := `"" "a" "a\"b"` + " `raw_str/\\`"
	tokenizer := ast.InitTokenizerForTest(input, filename)
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err := tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.StringLiteralToken)
	assert.Equals(t, tokenVal.Value, `""`)
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.StringLiteralToken)
	assert.Equals(t, tokenVal.Value, `"a"`)
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.StringLiteralToken)
	assert.Equals(t, tokenVal.Value, `"a\"b"`)
	assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
	tokenVal, err = tokenizer.GetNextForTest()
	assert.NoError(t, err)
	assert.Equals(t, tokenVal.TokenID, ast.RawStringLiteralToken)
	assert.Equals(t, tokenVal.Value, "`raw_str/\\`")
	assert.Equals(t, tokenizer.HasNextTokenForTest(), false)
}

func TestTokenizer_Wildcard(t *testing.T) {
	input := `$.*`
	tokenizer := ast.InitTokenizerForTest(input, filename)
	expectedValue := []string{"$", ".", "*"}
	for _, expected := range expectedValue {
		assert.Equals(t, tokenizer.HasNextTokenForTest(), true)
		nextToken, err := tokenizer.GetNextForTest()
		assert.NoError(t, err)
		assert.Equals(t, nextToken.Value, expected)
	}
}
