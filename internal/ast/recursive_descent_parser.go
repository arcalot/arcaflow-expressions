package ast

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

/*
Current grammar in Backus–Naur form:
<root_expression> ::= <or_expression>
<or_expression> ::= <and_expression> [ "|" "|" <and_expression> ]
<and_expression> ::= <not_expression> [ "&" "&" <not_expression> ]
<not_expression> ::= [ "!" ] <comparison_expression>
<comparison_expression> ::= <addition_subtraction_expression> [ <comparison_operator> <add_sub_expression> ]
<comparison_operator> ::= ">" | "<" | ">" "=" | "<" "=" | "=" "=" | "!" "="
<add_sub_expression> ::= <multiply_divide_expression> [ <add_sub_operator> <multiply_divide_expression>]
<add_sub_operator> ::=  "+" | "-"
<multiply_divide_expression> ::= <exponents_expression> [ <multiply_divide_operator> <exponents_expression> ]
<multiply_divide_operator> ::=  "*" | "/" | "%"
<exponents_expression> ::= <parentheses_expression> [ "^" <parentheses_expression> ]
<parentheses_expression> ::= <negation_expression> | "(" <root_expression> ")"
<negation_expression> ::= ["-"] <value_or_access_expression>
<value_or_access_expression> ::= <literal> | <identifier_or_function> [ <chained_access> ]
<identifier_or_function> := IdentifierToken | <function_call>
<function_call> := IdentifierToken "(" [ <argument_list> ] ")"
<chained_access> := <chainable_access> [ <chained_access> ]
<chainable_access_token> := <dot_notation> | <bracket_access>
<dot_notation> := "." IdentifierToken
<bracket_access> := "[" <root_expression> "]"
<literal> := IntLiteralToken | StringLiteralToken | FloatLiteralToken | BooleanLiteralToken
<argument_list> := <root_expression> | <argument_list> "," <root_expression>

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

var escapeReplacer = strings.NewReplacer(`\\`, `\`, `\t`, "\t", `\n`, "\n", `\r`, "\r", `\b`, "\b", `\"`, `"`)

func (p *Parser) parseStringLiteral() (*StringLiteral, error) {
	// The literal token includes the "", so trim the ends off.
	parsedString := p.currentToken.Value[1 : len(p.currentToken.Value)-1]
	// Replace escaped characters
	parsedString = escapeReplacer.Replace(parsedString)
	// Now create the literal itself and advance the token.
	literal := &StringLiteral{StrValue: parsedString}
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
		// Check for incomplete scenario.
		if p.currentToken == nil && i != 0 { // Reached end too early.
			return nil, &InvalidGrammarError{
				FoundToken:     p.currentToken,
				ExpectedTokens: []TokenID{ParenthesesEndToken},
			}
		}
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
			expectedTokens := []TokenID{expectedToken}
			if i != 0 {
				// Example `func(0` expect either func(0) func(0,
				expectedTokens = append(expectedTokens, ParenthesesEndToken)
			}
			return nil, &InvalidGrammarError{
				FoundToken:     p.currentToken,
				ExpectedTokens: expectedTokens,
			}
		}

		// Advances past the ( and the commas.
		err := p.advanceToken()
		if err != nil {
			return nil, err
		}
		// Check for incomplete scenario.
		if p.currentToken == nil { // Reached end too early.
			return nil, &InvalidGrammarError{
				FoundToken:     p.currentToken,
				ExpectedTokens: []TokenID{ParenthesesEndToken},
			}
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
	// "".a
	// Found . expected end of expression
	// OR dot notation cannot follow a literal // current
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
	case AsteriskToken:
		return Multiply, nil
	case DivideToken:
		return Divide, nil
	case PowerToken:
		return Power, nil
	case ModulusToken:
		return Modulus, nil
	case NotToken, GreaterThanToken, LessThanToken, EqualsToken:
		// Need to validate and advance past the following =
		if p.currentToken != nil && p.currentToken.TokenID == EqualsToken {
			// Equals is next, so return based on the token preceding the = token.
			err := p.advanceToken()
			if err != nil {
				return Invalid, err
			}
			switch firstToken {
			case NotToken:
				return NotEqualTo, nil
			case GreaterThanToken:
				return GreaterThanEqualTo, nil
			case LessThanToken:
				return LessThanEqualTo, nil
			case EqualsToken:
				return EqualTo, nil
			default:
				// If you get here, there is a case missing here that is in the outer switch
				panic(fmt.Errorf("illegal code state hit after token %s", firstToken))
			}
		} else {
			// No token, or non-equals token next, so validate as a single token.
			switch firstToken {
			case GreaterThanToken:
				return GreaterThan, nil
			case LessThanToken:
				return LessThan, nil
			case NotToken:
				return Not, nil
			case EqualsToken:
				// Expected double equals, but got single equals
				return Invalid, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{EqualsToken}}
			default:
				// If you get here, there is a case missing here that is in the outer switch
				panic(fmt.Errorf("illegal code state hit after token %s", firstToken))
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
			AsteriskToken,
			DivideToken,
			PowerToken,
			NotToken,
			GreaterThanToken,
			LessThanToken,
			EqualsToken,
			AndToken,
			OrToken,
			ModulusToken,
		}}
	}
}

// parseBinaryExpression parses a binary expression that has one of the supported operators,
// and uses childNodeParser for the left and right of the node.
// If, after parsing the first operand, the operator is not present, then the function returns
// successfully.
func (p *Parser) parseBinaryExpression(supportedOperators []TokenID, childNodeParser func() (Node, error)) (Node, error) {
	root, err := childNodeParser()
	if err != nil {
		return nil, err
	}
	// Loop to allow non-recursively evaluated repeating compatible operations.
	// Necessary for proper order of operations as currently designed.
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
		subNode, err := p.parseRootExpression()
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
// negation Parentheses Exponent Multiplication&Division Addition&Subtraction Comparisons not and or
// The higher-precedence ones should be deepest in the call tree. So logical or should be called first.
// For more details, see the grammar at the top of this file.

func (p *Parser) parseRootExpression() (Node, error) {
	// Currently `or` is the first one to call based on the order of operations specified above,
	// and based on the grammar specified at the top of the file.
	return p.parseConditionalOr()
}

func (p *Parser) parseConditionalOr() (Node, error) {
	return p.parseBinaryExpression([]TokenID{OrToken}, p.parseConditionalAnd)
}

func (p *Parser) parseConditionalAnd() (Node, error) {
	return p.parseBinaryExpression([]TokenID{AndToken}, p.parseConditionalNot)
}

func (p *Parser) parseConditionalNot() (Node, error) {
	return p.parseLeftUnaryExpression([]TokenID{NotToken}, p.parseComparisonExpression)
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
	return p.parseBinaryExpression([]TokenID{AsteriskToken, DivideToken, ModulusToken}, p.parseExponents)
}

func (p *Parser) parseExponents() (Node, error) {
	return p.parseBinaryExpression([]TokenID{PowerToken}, p.parseParentheses)
}

func (p *Parser) parseParentheses() (Node, error) {
	// If parentheses, continue recursing back from the root.
	// If not parentheses, recurse down into negation.
	if p.currentToken.TokenID != ParenthesesStartToken {
		return p.parseNegationOperation()
	}
	err := p.advanceToken() // Go past the parentheses
	if err != nil {
		return nil, err
	}
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

func (p *Parser) parseNegationOperation() (Node, error) {
	return p.parseLeftUnaryExpression([]TokenID{NegationToken}, p.parseValueOrAccessExpression)
}

var expStartIdentifierTokens = []TokenID{RootAccessToken, CurrentObjectAccessToken, IdentifierToken}
var literalTokens = []TokenID{StringLiteralToken, IntLiteralToken, BooleanLiteralToken, FloatLiteralToken}
var validStartTokens = append(expStartIdentifierTokens, literalTokens...)

// parseValueOrAccessExpression parses a root expression
func (p *Parser) parseValueOrAccessExpression() (Node, error) {
	if p.currentToken == nil || !sliceContains(validStartTokens, p.currentToken.TokenID) {
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: validStartTokens}
	} else if p.atRoot && p.currentToken.TokenID == CurrentObjectAccessToken {
		// Can't support @ at root
		return nil, &InvalidGrammarError{FoundToken: p.currentToken, ExpectedTokens: []TokenID{RootAccessToken, IdentifierToken}}
	}
	p.atRoot = false // Know when you can reference the current object.

	var literalNode Node
	var err error
	// A value or access expression can start with a literal, or an identifier.
	// If an identifier, it can lead to a chain or a function.
	switch p.currentToken.TokenID {
	case StringLiteralToken:
		literalNode, err = p.parseStringLiteral()
	case IntLiteralToken:
		literalNode, err = p.parseIntLiteral()
	case FloatLiteralToken:
		literalNode, err = p.parseFloatLiteral()
	case BooleanLiteralToken:
		literalNode, err = p.parseBooleanLiteral()
	default:
		// Not a literal, or a literal case is missing in the switch
		// So if it gets here it's an identifier. That can lead to a chain or function call.
		return p.parseIdentifierOrFunction()
	}
	// Literal case
	if err != nil {
		return nil, err
	}
	// Lookahead validation for nothing incorrect following the literal for better error messages.
	if p.currentToken == nil { // Nothing after, so likely valid.
		return literalNode, nil
	}
	switch p.currentToken.TokenID {
	// These are all access start tokens, which cannot follow a literal.
	case ParenthesesStartToken:
		return nil, fmt.Errorf("function call must start with an identifier; got %q after %q", p.currentToken.Value, literalNode.String())
	case DotObjectAccessToken:
		return nil, fmt.Errorf("dot notation cannot follow a literal; got %q after %q", p.currentToken.Value, literalNode.String())
	case BracketAccessDelimiterStartToken:
		return nil, fmt.Errorf("bracket access cannot follow a literal; got %q after %q", p.currentToken.Value, literalNode.String())
	}
	return literalNode, nil
}

// Parses the current identifier, parses the arg list if available, then checks for chainable accesses.
// Expects to be called when the current node is an identifier.
func (p *Parser) parseIdentifierOrFunction() (Node, error) {
	firstNode := &Identifier{IdentifierName: p.currentToken.Value}
	err := p.advanceToken()
	if err != nil {
		return nil, err
	}
	chainableNode, err := p.parseFunctionArgs(firstNode)
	if err != nil {
		return nil, err
	}
	if p.currentToken == nil {
		// Nothing follows, so stop here
		return chainableNode, nil
	}
	return p.parseChainedAccess(chainableNode)
}

// parseFunctionArgs parses all parts of a function call that follow the identifier, including the parentheses.
// If a parameter list is not found, it returns the identifier.
func (p *Parser) parseFunctionArgs(precedingNode *Identifier) (Node, error) {
	if p.currentToken == nil || p.currentToken.TokenID != ParenthesesStartToken {
		// No function call. Return the original input for chaining.
		return precedingNode, nil
	}
	argList, err := p.parseArgs()
	if err != nil {
		return nil, err
	}
	return &FunctionCall{
		FuncIdentifier: precedingNode,
		ArgumentInputs: argList,
	}, nil
}

// parseChainedAccess parses all the dot notations, map accesses, binary operations, and function calls.
// Must be called after Identifier, FunctionCall, or another chained access node.
func (p *Parser) parseChainedAccess(rootNode Node) (Node, error) {
	var currentNode = rootNode
	for p.currentToken != nil {
		switch p.currentToken.TokenID {
		case DotObjectAccessToken:
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
	return currentNode, nil
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
