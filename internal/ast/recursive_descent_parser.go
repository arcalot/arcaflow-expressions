package ast

import (
	"errors"
	"fmt"
	"strconv"
)

/*
Current grammar:
root_expression ::= root_identifier [expression_access] | literal | function_call
chained_expression := identifier [expression_access]
expression_access ::= map_access | dot_notation
map_access ::= "[" key "]" [chained_expression]
dot_notation ::= "." identifier [chained_expression]
root_identifier ::= identifier | "$"
literal := IntLiteralToken | StringLiteralToken
function_call := identifier "(" [argument_list] ")"
argument_list := argument_list "," root_expression | root_expression

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
func (p *Parser) parseBracketAccess(expressionToAccess Node) (*BracketAccessor, error) {
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

	subExpr, err := p.parseRootExpression()
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

	return &BracketAccessor{LeftNode: expressionToAccess, RightExpression: subExpr}, nil
}

func (p *Parser) parseIntLiteral() (*IntLiteral, error) {
	if p.currentToken.TokenID != IntLiteralToken {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{IntLiteralToken}}
	}
	parsedInt, err := strconv.Atoi(p.currentToken.Value)
	if err != nil {
		return nil, err // Should not fail if the parser is set up correctly
	}
	literal := &IntLiteral{IntValue: parsedInt}
	err = p.advanceToken()
	if err != nil {
		return nil, err
	}
	return literal, nil
}

func (p *Parser) parseStringLiteral() (*StringLiteral, error) {
	// The literal token includes the "", so trim the ends off.
	literal := &StringLiteral{StrValue: p.currentToken.Value[1 : len(p.currentToken.Value)-1]}
	err := p.advanceToken()
	if err != nil {
		return nil, err
	}
	return literal, nil
}

func (p *Parser) parseArgs() (*ArgumentList, error) {
	// Keep parsing expressions until you hit a comma.
	argNodes := make([]Node, 0)
	expectedToken := ArgListStartToken
	for i := 0; ; i++ {
		// Validate and go past the first ( on the first iteration, and commas on later iterations.
		if i != 0 && p.currentToken.TokenID == ArgListEndToken {
			// Advances past the )
			err := p.advanceToken()
			if err != nil {
				return nil, err
			}
			return &ArgumentList{Arguments: argNodes}, nil
		} else if p.currentToken.TokenID != expectedToken {
			// The first is preceded by a (, the others are preceded by ,
			return nil, &InvalidGrammarError{
				FoundToken:     p.currentToken,
				ExpectedTokens: []TokenID{expectedToken},
			}
		}

		// Advances past the ( and the commas.
		err := p.advanceToken()
		if err != nil {
			return nil, err
		}
		// Check end condition
		if i == 0 && p.currentToken.TokenID == ArgListEndToken {
			// Advances past the )
			err := p.advanceToken()
			if err != nil {
				return nil, err
			}
			return &ArgumentList{Arguments: argNodes}, nil
		}

		// It should be able to process a whole expression within the arg
		arg, err := p.parseRootExpression()
		if err != nil {
			return nil, err
		}
		argNodes = append(argNodes, arg)
		// From this point forward, commas will precede all the args.
		if i == 0 {
			expectedToken = ListSeparatorToken
		}
	}
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

	node, err := p.parseRootExpression()
	if err != nil {
		return nil, err
	} else if p.currentToken != nil {
		// Reached wrong token. It should be at the end here.
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: nil}
	}
	return node, err
}

var expStartIdentifierTokens = []TokenID{RootAccessToken, CurrentObjectAccessToken, IdentifierToken}
var literalTokens = []TokenID{StringLiteralToken, IntLiteralToken}
var validStartTokens = append(expStartIdentifierTokens, literalTokens...)

// parseSubExpression parses all the dot notations, map accesses, and function calls.
func (p *Parser) parseAfterIdentifier(identifier *Identifier) (Node, error) {
	var currentNode Node = identifier
	// Handle types that cannot be chained first.
	if p.currentToken.TokenID == ArgListStartToken {
		// Function call
		argList, err := p.parseArgs()
		if err != nil {
			return nil, err
		}
		currentNode = &FunctionCall{
			FuncIdentifier:  identifier,
			ParameterInputs: argList,
		}
	}
	for {
		switch {
		case p.currentToken == nil:
			// Reached end
			return currentNode, nil
		case p.currentToken.TokenID == DotObjectAccessToken:
			// Dot notation
			err := p.advanceToken() // Move past the .
			if err != nil {
				return nil, err
			}
			accessingIdentifier, err := p.parseIdentifier()
			if err != nil {
				return nil, err
			}
			currentNode = &DotNotation{LeftAccessibleNode: currentNode, RightAccessIdentifier: accessingIdentifier}
		case p.currentToken.TokenID == BracketAccessDelimiterStartToken:
			// Bracket notation
			parsedMapAccess, err := p.parseBracketAccess(currentNode)
			if err != nil {
				return nil, err
			}
			currentNode = parsedMapAccess
		default:
			// Reached a token this function is not responsible for
			return currentNode, nil
		}
	}
}

// parseRootExpression parses a root expression
func (p *Parser) parseRootExpression() (Node, error) {
	if p.currentToken == nil || !sliceContains(validStartTokens, p.currentToken.TokenID) {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: validStartTokens}
	} else if p.atRoot && p.currentToken.TokenID == CurrentObjectAccessToken {
		// Can't support @ at root
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{RootAccessToken, IdentifierToken}}
	}
	if p.atRoot {
		p.atRoot = false // Know when you can reference the current object.
	}

	// An expression can start with a literal, or an identifier. If an identifier, it can lead to a chain or a function.
	if sliceContains(literalTokens, p.currentToken.TokenID) {
		switch literalToken := p.currentToken.TokenID; literalToken {
		case StringLiteralToken:
			return p.parseStringLiteral()
		case IntLiteralToken:
			return p.parseIntLiteral()
		default:
			return nil, fmt.Errorf(
				"bug: Literal token type %s is missing from switch in parseUnchainedRootExpression",
				literalTokens)
		}
	}
	// The literal case is accounted for, so if it gets here it's an identifier. That can lead to a chain or function call.
	var firstIdentifier = &Identifier{IdentifierName: p.currentToken.Value}
	err := p.advanceToken()
	if err != nil {
		return nil, err
	}
	if p.currentToken != nil {
		return p.parseAfterIdentifier(firstIdentifier)
	} else {
		return firstIdentifier, nil
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
