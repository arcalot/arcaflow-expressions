package ast

import (
	"regexp"
	"strings"
	"text/scanner"
)

// TokenID Represents the name of a type of token that has a pattern.
type TokenID string

const (
	// IdentifierToken represents a token with any valid object name.
	IdentifierToken TokenID = "identifier"
	// StringLiteralToken represents a token that has a sequence of characters.
	// Supports the string format used in golang, and will include
	// the " before and after the contents of the string.
	// Characters can be escaped the common way with a backslash.
	StringLiteralToken    TokenID = "string"
	RawStringLiteralToken TokenID = "raw-string"
	// IntLiteralToken represents an integer token. Must not start with 0.
	IntLiteralToken TokenID = "int"
	// FloatLiteralToken represents a float token.
	FloatLiteralToken TokenID = "float"
	// BooleanLiteralToken represents true or false.
	BooleanLiteralToken TokenID = "boolean"
	// BracketAccessDelimiterStartToken represents the token before an object
	//  access. The '[' in 'obj["key"]'.
	//nolint:gosec
	BracketAccessDelimiterStartToken TokenID = "map-delimiter-start"
	// BracketAccessDelimiterEndToken represents the token before an object
	// access. The '[' in 'obj["key"]'.
	//nolint:gosec
	BracketAccessDelimiterEndToken TokenID = "map-delimiter-end"
	// ParenthesesStartToken represents the start token of an argument list or a parenthesized expression. '('
	ParenthesesStartToken TokenID = "parentheses-start"
	// ParenthesesEndToken represents the closing of the argument list. ')'
	ParenthesesEndToken TokenID = "parentheses-end"
	// DotObjectAccessToken represents the '.' token in 'a.b' (dot notation).
	DotObjectAccessToken TokenID = "object-access"
	// RootAccessToken represents the token that identifies accessing the
	// root object.
	RootAccessToken TokenID = "root-access"
	// CurrentObjectAccessToken represents the token, @, that identifies the current
	// object in a filter.
	CurrentObjectAccessToken TokenID = "current-object-access"
	// EqualsToken represents the token that represents a single equals sign.
	EqualsToken TokenID = "equals-sign"
	// SelectorToken Represents the ':' character used in selector expressions in bracket
	// object access.
	SelectorToken TokenID = "selector"
	// FilterToken represents the '?' used in filter expressions in bracket object access.
	FilterToken TokenID = "filter"
	// NegationToken represents a negation sign '-'.
	//nolint:gosec
	NegationToken TokenID = "negation-sign"
	// AsteriskToken represents a wildcard/multiplication token '*'.
	AsteriskToken TokenID = "asterisk"
	// ListSeparatorToken represents a comma in a parameter list
	ListSeparatorToken TokenID = "list-separator" //nolint:gosec // not a security credential
	// DivideToken represents the forward slash used to specify division.
	DivideToken TokenID = "divide"
	// GreaterThanToken represents a > symbol.
	GreaterThanToken TokenID = "greater-than"
	// LessThanToken represents a < symbol.
	LessThanToken TokenID = "less-than"
	// PlusToken represents a + symbol.
	PlusToken TokenID = "plus"
	// NotToken represents an ! symbol.
	NotToken TokenID = "not"
	// PowerToken represents a caret symbol for exponentiation.
	PowerToken TokenID = "power"
	// ModulusToken represents a percent symbol for remainder.
	ModulusToken TokenID = "mod"
	// AndToken represents logical-and &&
	AndToken TokenID = "and"
	// OrToken represents logical-or ||
	OrToken TokenID = "or"
	// UnknownToken is a placeholder for when there was an error in the token.
	UnknownToken TokenID = "error"
)

// TokenValue represents the token parsed from the expression the tokenizer
// was initialized with.
// The line number and column is relative to the beginning of the expression.
// If part of a greater file, it's recommended that you offset those values to
// get the line and column within the file to prevent confusion.
type TokenValue struct {
	Value    string
	TokenID  TokenID
	Filename string
	Line     int
	Column   int
}

// tokenizer is used for reading tokens of an expression.
type tokenizer struct {
	s      scanner.Scanner
	reader *strings.Reader
}

type tokenPattern struct {
	TokenID TokenID
	*regexp.Regexp
}

var tokenPatterns = []tokenPattern{
	{BooleanLiteralToken, regexp.MustCompile(`^(?:true|false)$`)},          // true or false. Note: This needs to be above IdentifierToken
	{FloatLiteralToken, regexp.MustCompile(`^\d+\.\d*(?:[eE][+-]?\d+)?$`)}, // Like an integer, but with a period and digits after.
	{IntLiteralToken, regexp.MustCompile(`^(?:0|[1-9]\d*)$`)},              // Note: numbers that start with 0 are identifiers.
	{IdentifierToken, regexp.MustCompile(`^\w+$`)},                         // Any valid object name
	{StringLiteralToken, regexp.MustCompile(`^(?:".*"|'.*')$`)},            // "string example" 'alternative'
	{RawStringLiteralToken, regexp.MustCompile("^`.*`$")},                  // `raw string`
	{BracketAccessDelimiterStartToken, regexp.MustCompile(`^\[$`)},         // the [ in map["key"]
	{BracketAccessDelimiterEndToken, regexp.MustCompile(`^]$`)},            // the ] in map["key"]
	{ParenthesesStartToken, regexp.MustCompile(`^\($`)},                    // (
	{ParenthesesEndToken, regexp.MustCompile(`^\)$`)},                      // )
	{DotObjectAccessToken, regexp.MustCompile(`^\.$`)},                     // .
	{RootAccessToken, regexp.MustCompile(`^\$$`)},                          // $
	{CurrentObjectAccessToken, regexp.MustCompile(`^@$`)},                  // @
	{EqualsToken, regexp.MustCompile(`^=$`)},                               // =
	{SelectorToken, regexp.MustCompile(`^:$`)},                             // :
	{FilterToken, regexp.MustCompile(`^\?$`)},                              // ?
	{NegationToken, regexp.MustCompile(`^-$`)},                             // -
	{AsteriskToken, regexp.MustCompile(`^\*$`)},                            // *
	{ListSeparatorToken, regexp.MustCompile(`^,$`)},                        // ,
	{DivideToken, regexp.MustCompile(`^/$`)},                               // /
	{GreaterThanToken, regexp.MustCompile(`^>$`)},                          // >
	{LessThanToken, regexp.MustCompile(`^<$`)},                             // <
	{PlusToken, regexp.MustCompile(`^\+$`)},                                // +
	{NotToken, regexp.MustCompile(`^!$`)},                                  // !
	{PowerToken, regexp.MustCompile(`^\^$`)},                               // ^
	{ModulusToken, regexp.MustCompile(`^%$`)},                              // %
	{AndToken, regexp.MustCompile(`^&$`)},                                  // &&
	{OrToken, regexp.MustCompile(`^\|$`)},                                  // ||
}

// initTokenizer initializes the tokenizer struct with the given expression.
func initTokenizer(expression string, sourceName string) *tokenizer {
	var t tokenizer
	// Need to trim the whitespace first since that can cause unexpected blank tokens.
	t.reader = strings.NewReader(strings.TrimSpace(expression))
	t.s.Init(t.reader)
	t.s.Filename = sourceName
	return &t
}

// hasNextToken Checks to see if it has reached the end of the expression.
// If it has, it returns false. If there are tokens left, it returns true.
func (t *tokenizer) hasNextToken() bool {
	return t.s.Peek() != scanner.EOF
}

// getNext gets the next token type and value.
// If there is no token left, it returns an unknown token and an
// InvalidTokenError.
func (t *tokenizer) getNext() (*TokenValue, error) {
	t.s.Scan()
	tokenValue := t.s.TokenText()
	for _, tokenPattern := range tokenPatterns {
		if tokenPattern.Regexp.MatchString(tokenValue) {
			return &TokenValue{tokenValue, tokenPattern.TokenID, t.s.Filename, t.s.Line, t.s.Column}, nil
		}
	}
	result := TokenValue{tokenValue, UnknownToken, t.s.Filename, t.s.Line, t.s.Column}
	return &result, &InvalidTokenError{result}
}
