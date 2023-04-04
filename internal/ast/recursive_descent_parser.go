package ast

import (
	"errors"
	"strconv"
)

/*
Current grammar:
root_expression ::= root_identifier [expression_access]
expression ::= identifier [expression_access]
expression_access ::= map_access | dot_notation
map_access ::= "[" key "]" [expression]
dot_notation ::= "." identifier [expression]
root_identifier ::= identifier | "$"
key ::= IntLiteralToken | StringLiteralToken | "(" expression ")"

filtering/querying will be added later if needed.
*/

// Parser represents the object that handles parsing the grammar for the
// expression.
// Create this with the function expressions.InitParser
// This struct and its functions are used to parse the
// expression it was initialized with.
type Parser struct {
	t            *tokenizer
	currentToken *TokenValue
	atRoot       bool
}

// InitParser initializes the parser with the given raw expression.
func InitParser(expression string, fileName string) (*Parser, error) {
	t := initTokenizer(expression, fileName)
	p := &Parser{t: t}
	p.atRoot = true

	return p, nil
}

// advanceToken advances to the next token by updating the current token var.
// Also needed before parsing.
func (p *Parser) advanceToken() error {
	if p.t.hasNextToken() {
		newToken, err := p.t.getNext()
		p.currentToken = newToken
		return err
	}
	p.currentToken = nil
	return nil
}

// parseBracketAccess parses a bracket access in the form of a
// bracket, followed by the key, followed by a closing bracket.
//
//nolint:funlen
func (p *Parser) parseBracketAccess(expressionToAccess Node) (*MapAccessor, error) {
	if expressionToAccess == nil {
		return nil, errors.New("parameter expressionToAccess is nil")
	}
	// Verify and read in the [
	if p.currentToken == nil ||
		p.currentToken.TokenID != BracketAccessDelimiterStartToken {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{IdentifierToken}}
	}
	err := p.advanceToken()
	if err != nil {
		return nil, err
	}

	validTokens := []TokenID{StringLiteralToken, IntLiteralToken, ExpressionStartToken}

	// Read in the key
	if p.currentToken == nil || !sliceContains(validTokens, p.currentToken.TokenID) {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: validTokens}
	}
	var key *Key
	// Bracket access notation allows string literals, int literals, and sub-expressions
	switch {
	case p.currentToken.TokenID == StringLiteralToken:
		// The literal token includes the "", so trim the ends off.
		key = &Key{Literal: &StringLiteral{StrValue: p.currentToken.Value[1 : len(p.currentToken.Value)-1]}}
	case p.currentToken.TokenID == IntLiteralToken:
		parsedInt, err := strconv.Atoi(p.currentToken.Value)
		if err != nil {
			return nil, err // Should not fail if the parser is set up correctly
		}
		key = &Key{Literal: &IntLiteral{IntValue: parsedInt}}
	case p.currentToken.TokenID == ExpressionStartToken:
		err = p.advanceToken() // Read past (
		if err != nil {
			return nil, err
		}
		node, err := p.parseSubExpression()
		if err != nil {
			return nil, err
		}
		key = &Key{SubExpression: node}

		// Verify that next token is end of expression )
		if p.currentToken == nil || p.currentToken.TokenID != ExpressionEndToken {
			return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{ExpressionEndToken}}
		}
	}
	err = p.advanceToken()
	if err != nil {
		return nil, err
	}

	// Verify and read in the ]
	if p.currentToken == nil ||
		p.currentToken.TokenID != BracketAccessDelimiterEndToken {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{IdentifierToken}}
	}
	err = p.advanceToken()
	if err != nil {
		return nil, err
	}

	return &MapAccessor{LeftNode: expressionToAccess, RightKey: *key}, nil

}

// parseIdentifier parses a valid identifier.
func (p *Parser) parseIdentifier() (*Identifier, error) {
	// Only accessing one token, the identifier
	if p.currentToken == nil ||
		p.currentToken.TokenID != IdentifierToken {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{IdentifierToken}}
	}

	parsedIdentifier := &Identifier{IdentifierName: p.currentToken.Value}
	err := p.advanceToken()
	if err != nil {
		return nil, err
	}
	return parsedIdentifier, nil
}

// ParseExpression is the correct entrypoint for parsing an expression.
// It advances to the first token, and parses the expression.
func (p *Parser) ParseExpression() (Node, error) {
	err := p.advanceToken()
	if err != nil {
		return nil, err
	}

	node, err := p.parseSubExpression()
	if p.currentToken != nil {
		// Reached wrong token. It should be at the end here.
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: nil}
	}
	return node, err
}

// parseSubExpression parses all the dot notations and map accesses.
func (p *Parser) parseSubExpression() (Node, error) {
	supportedTokens := []TokenID{RootAccessToken, CurrentObjectAccessToken, IdentifierToken}
	// The first identifier should always be the root identifier, $
	if p.currentToken == nil || !sliceContains(supportedTokens, p.currentToken.TokenID) {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: supportedTokens}
	} else if p.atRoot && p.currentToken.TokenID == CurrentObjectAccessToken {
		// Can't support @ at root
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{RootAccessToken, IdentifierToken}}
	}
	if p.atRoot {
		p.atRoot = false // No longer allow $
	}

	var parsed Node = &Identifier{IdentifierName: p.currentToken.Value}
	err := p.advanceToken()
	if err != nil {
		return nil, err
	}

	for {

		switch {
		case p.currentToken == nil:
			// Reached end
			return parsed, nil
		case p.currentToken.TokenID == DotObjectAccessToken:
			// Dot notation
			err = p.advanceToken() // Move past the .
			if err != nil {
				return nil, err
			}
			accessingIdentifier, err := p.parseIdentifier()
			if err != nil {
				return nil, err
			}
			parsed = &DotNotation{LeftAccessibleNode: parsed, RightAccessIdentifier: accessingIdentifier}
		case p.currentToken.TokenID == BracketAccessDelimiterStartToken:
			// Bracket notation
			parsedMapAccess, err := p.parseBracketAccess(parsed)
			if err != nil {
				return nil, err
			}
			parsed = parsedMapAccess
		default:
			// Reached a token this function is not responsible for
			return parsed, nil
		}
	}
}

func sliceContains(slice []TokenID, value TokenID) bool {
	for _, val := range slice {
		if val == value {
			return true
		}
	}
	return false
}
