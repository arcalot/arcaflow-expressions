package ast

import (
	"errors"
	"fmt"
	"strconv"
)

/*
Current grammar:
root_expression ::= root_identifier [expression_access] | literal | function_call | binary_operation
chained_expression := identifier [expression_access]
expression_access ::= map_access | dot_notation
map_access ::= "[" key "]" [chained_expression]
dot_notation ::= "." identifier [chained_expression]
root_identifier ::= identifier | "$"
literal := IntLiteralToken | StringLiteralToken
function_call := identifier "(" [argument_list] ")"
argument_list := argument_list "," root_expression | root_expression
binary_operator := ">" | "<" | ">" "=" | "<" "=" | "=" "=" | "!" "=" | "+" | "-" | "*" | "/"
binary_operation := root_expression binary_operator root_expression

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
	err = p.eat([]TokenID{BracketAccessDelimiterEndToken})
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
	literal := &IntLiteral{IntValue: int64(parsedInt)}
	err = p.advanceToken()
	if err != nil {
		return nil, err
	}
	return literal, nil
}

func (p *Parser) parseFloatLiteral() (*FloatLiteral, error) {
	if p.currentToken.TokenID != FloatLiteralToken {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{FloatLiteralToken}}
	}
	parsedFloat, err := strconv.ParseFloat(p.currentToken.Value, 64)
	if err != nil {
		return nil, err // Should not fail if the parser is set up correctly
	}
	literal := &FloatLiteral{FloatValue: parsedFloat}
	err = p.advanceToken()
	if err != nil {
		return nil, err
	}
	return literal, nil
}

func (p *Parser) parseBooleanLiteral() (*BooleanLiteral, error) {
	if p.currentToken.TokenID != BooleanLiteralToken {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{BooleanLiteralToken}}
	}
	parsedBoolean, err := strconv.ParseBool(p.currentToken.Value)
	if err != nil {
		return nil, err // Should not fail if the parser is set up correctly
	}
	literal := &BooleanLiteral{BooleanValue: parsedBoolean}
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
	expectedToken := ParenthesesStartToken
	for i := 0; ; i++ {
		// Validate and go past the first ( on the first iteration, and commas on later iterations.
		if i != 0 && p.currentToken.TokenID == ParenthesesEndToken {
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
				ExpectedTokens: []TokenID{expectedToken, ParenthesesEndToken},
			}
		}

		// Advances past the ( and the commas.
		err := p.advanceToken()
		if err != nil {
			return nil, err
		}
		// Check end condition
		if i == 0 && p.currentToken.TokenID == ParenthesesEndToken {
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

func (p *Parser) parseMathOperator() (MathOperationType, error) {
	firstToken := p.currentToken.TokenID
	err := p.advanceToken()
	if err != nil {
		return Invalid, err
	}
	switch firstToken {
	case PlusToken:
		return Add, nil
	case NegationToken:
		return Subtract, nil
	case WildcardMultiplyToken:
		return Multiply, nil
	case DivideToken:
		return Divide, nil
	case PowerToken:
		return Power, nil
	case ModulusToken:
		return Modulus, nil
	case NotToken, GreaterThanToken, LessThanToken, EqualsToken:
		// Need to validate and advance past the following =
		if p.currentToken.TokenID == EqualsToken {
			err := p.advanceToken()
			if err != nil {
				return Invalid, err
			}
			switch firstToken {
			case NotToken:
				return NotEquals, nil
			case GreaterThanToken:
				return GreaterThanEquals, nil
			case LessThanToken:
				return LessThanEquals, nil
			case EqualsToken:
				return Equals, nil
			default:
				// If you get here, there is a case missing here that is in the outer switch
				return Invalid, fmt.Errorf("illegal code state hit after token %s", firstToken)
			}
		} else {
			switch firstToken {
			case GreaterThanToken:
				return GreaterThan, nil
			case LessThanToken:
				return LessThan, nil
			default:
				// Not equal or double equals
				return Invalid, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{EqualsToken}}
			}
		}
	case AndToken:
		if p.currentToken == nil || p.currentToken.TokenID != AndToken {
			return Invalid, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{AndToken}}
		}
		err := p.advanceToken()
		if err != nil {
			return Invalid, err
		}
		return And, nil
	case OrToken:
		if p.currentToken == nil || p.currentToken.TokenID != OrToken {
			return Invalid, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{OrToken}}
		}
		err := p.advanceToken()
		if err != nil {
			return Invalid, err
		}
		return Or, nil
	default:
		return Invalid, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{
			PlusToken,
			NegationToken,
			WildcardMultiplyToken,
			DivideToken,
			PowerToken,
			NotToken,
			GreaterThanToken,
			LessThanToken,
			EqualsToken,
		}}
	}
}

// parseBinaryExpression parses a binary expression that has one of the supported operators,
// and uses childNodeParser for the left and right of the node.
// If the given operator isn't found, it still continues down recursively with the childNodeParser.
func (p *Parser) parseBinaryExpression(supportedOperators []TokenID, childNodeParser func() (Node, error)) (Node, error) {
	root, err := childNodeParser()
	if err != nil {
		return nil, err
	}
	// Loop while there is addition or subtraction ahead.
	for p.currentToken != nil && sliceContains(supportedOperators, p.currentToken.TokenID) {
		operatorToken, err := p.parseMathOperator()
		if err != nil {
			return nil, err
		}
		right, err := childNodeParser()
		if err != nil {
			return nil, err
		}
		root = &BinaryOperation{
			LeftNode:  root,
			RightNode: right,
			Operation: operatorToken,
		}
	}
	return root, nil
}

// parseLeftUnaryExpression parses an expression with the operator on the left, and the rest of the expression
// on the right. If the expected token is not there, it continues recursively with childNodeParser.
func (p *Parser) parseLeftUnaryExpression(supportedOperators []TokenID, childNodeParser func() (Node, error)) (Node, error) {
	if p.currentToken == nil {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{}}
	}
	if sliceContains(supportedOperators, p.currentToken.TokenID) {
		operation, err := p.parseMathOperator()
		if err != nil {
			return nil, err
		}
		subNode, err := childNodeParser()
		if err != nil {
			return nil, err
		}
		return &UnaryOperation{
			LeftOperation: operation,
			RightNode:     subNode,
		}, nil
	}
	return childNodeParser()
}

// ORDER OF OPERATIONS
// negation P E MD AS Comparisons not and or
// The higher-precedence ones should be deepest in the call tree. So logical or should be called first.

func (p *Parser) parseRootExpression() (Node, error) {
	// Currently or is the first one to call based on the order of operations specified above.
	return p.parseConditionalOr()
}

func (p *Parser) parseConditionalOr() (Node, error) {
	return p.parseBinaryExpression([]TokenID{OrToken}, p.parseConditionalAnd)
}

func (p *Parser) parseConditionalAnd() (Node, error) {
	return p.parseBinaryExpression([]TokenID{AndToken}, p.parseConditionalNot)
}

func (p *Parser) parseConditionalNot() (Node, error) {
	return p.parseLeftUnaryExpression([]TokenID{NegationToken}, p.parseComparisonExpression)
}

func (p *Parser) parseComparisonExpression() (Node, error) {
	// The allowed tokens are the FIRST ones associated with a binary comparison. The parseMathOperator func called by
	// parseBinaryExpression will handle the second token, if present.
	return p.parseBinaryExpression([]TokenID{GreaterThanToken, LessThanToken, NotToken, EqualsToken}, p.parseAdditionSubtraction)
}

func (p *Parser) parseAdditionSubtraction() (Node, error) {
	return p.parseBinaryExpression([]TokenID{PlusToken, NegationToken}, p.parseMultiplicationDivision)
}

func (p *Parser) parseMultiplicationDivision() (Node, error) {
	return p.parseBinaryExpression([]TokenID{WildcardMultiplyToken, DivideToken, ModulusToken}, p.parseExponents)
}

func (p *Parser) parseExponents() (Node, error) {
	return p.parseBinaryExpression([]TokenID{PowerToken}, p.parseParentheses)
}

func (p *Parser) parseParentheses() (Node, error) {
	// If parentheses are hit, start back at addition/subtraction.
	if p.currentToken.TokenID == ParenthesesStartToken {
		err := p.advanceToken() // Go past the parentheses
		if err != nil {
			return nil, err
		}
		// Back to the root
		node, err := p.parseRootExpression()
		if err != nil {
			return nil, err
		}
		err = p.eat([]TokenID{ParenthesesEndToken})
		if err != nil {
			return nil, err
		}
		return node, nil
	}
	return p.parseNegationOperation()
}

func (p *Parser) parseNegationOperation() (Node, error) {
	return p.parseLeftUnaryExpression([]TokenID{NegationToken}, p.parseRootValueOrAccess)
}

// parseRootExpression parses a root expression
func (p *Parser) parseRootValueOrAccess() (Node, error) {
	if p.currentToken == nil || !sliceContains(validStartTokens, p.currentToken.TokenID) {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: validStartTokens}
	} else if p.atRoot && p.currentToken.TokenID == CurrentObjectAccessToken {
		// Can't support @ at root
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{RootAccessToken, IdentifierToken}}
	}
	if p.atRoot {
		p.atRoot = false // Know when you can reference the current object.
	}

	var firstNode Node
	var err error
	// An expression can start with a literal, or an identifier. If an identifier, it can lead to a chain or a function.
	if sliceContains(literalTokens, p.currentToken.TokenID) {
		switch literalToken := p.currentToken.TokenID; literalToken {
		case StringLiteralToken:
			firstNode, err = p.parseStringLiteral()
		case IntLiteralToken:
			firstNode, err = p.parseIntLiteral()
		case FloatLiteralToken:
			firstNode, err = p.parseFloatLiteral()
		case BooleanLiteralToken:
			firstNode, err = p.parseBooleanLiteral()
		default:
			return nil, fmt.Errorf(
				"bug: Literal token type %s is missing from switch in parseUnchainedRootExpression",
				literalTokens)
		}
		if err != nil {
			return nil, err
		}
	} else {
		firstNode = &Identifier{IdentifierName: p.currentToken.Value}
		// The literal case is accounted for, so if it gets here it's an identifier. That can lead to a chain or function call.
		err = p.advanceToken()
		if err != nil {
			return nil, err
		}
	}
	if p.currentToken != nil {
		return p.parseChainedValueOrAccess(firstNode)
	} else {
		return firstNode, nil
	}
}

var expStartIdentifierTokens = []TokenID{RootAccessToken, CurrentObjectAccessToken, IdentifierToken}
var literalTokens = []TokenID{StringLiteralToken, IntLiteralToken, BooleanLiteralToken, FloatLiteralToken}
var validStartTokens = append(expStartIdentifierTokens, literalTokens...)

// parseSubExpression parses all the dot notations, map accesses, binary operations, and function calls.
func (p *Parser) parseChainedValueOrAccess(rootNode Node) (Node, error) {
	var rootNodeAny any = rootNode
	identifier, rootIsIdentier := rootNodeAny.(*Identifier)
	var currentNode = rootNode
	// Handle types that cannot be chained first.
	if p.currentToken.TokenID == ParenthesesStartToken {
		// Function call
		argList, err := p.parseArgs()
		if err != nil {
			return nil, err
		}
		currentNode = &FunctionCall{
			FuncIdentifier: identifier,
			ArgumentInputs: argList,
		}
	}
	for {
		if p.currentToken == nil {
			// Reached end
			return currentNode, nil
		}
		switch p.currentToken.TokenID {
		case DotObjectAccessToken:
			if !rootIsIdentier {
				return nil, fmt.Errorf("dot notation cannot follow a literal")
			}
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
		case BracketAccessDelimiterStartToken:
			if !rootIsIdentier {
				return nil, fmt.Errorf("bracket access cannot follow a literal")
			}
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

// eat validates then goes past the given token.
// For use when you know which tokens are required.
func (p *Parser) eat(validTokens []TokenID) error {
	if p.currentToken == nil || !sliceContains(validTokens, p.currentToken.TokenID) {
		return &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: validTokens}
	}
	return p.advanceToken()
}

// sliceContains is here to support versions of go before slices.Contains was added in Go 1.21
func sliceContains(slice []TokenID, value TokenID) bool {
	for _, val := range slice {
		if val == value {
			return true
		}
	}
	return false
}
