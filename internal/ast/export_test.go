package ast

// InitTokenizerForTest exports initTokenizer for use in tests.
func InitTokenizerForTest(expression string, sourceName string) *tokenizer {
	return initTokenizer(expression, sourceName)
}

// AdvanceTokenForTest exports advanceToken for use in tests.
func (p *Parser) AdvanceTokenForTest() error {
	return p.advanceToken()
}

// ParseIdentifierForTest exports parseIdentifier for use in tests.
func (p *Parser) ParseIdentifierForTest() (*Identifier, error) {
	return p.parseIdentifier()
}

// ParseBracketAccessForTest exports parseBracketAccess for use in tests.
func (p *Parser) ParseBracketAccessForTest(expressionToAccess Node) (*BracketAccessor, error) {
	return p.parseBracketAccess(expressionToAccess)
}

// ParseArgsForTest exports parseArgs for use in tests.
func (p *Parser) ParseArgsForTest() (*ArgumentList, error) {
	return p.parseArgs()
}

// ParseStringLiteralForTest exports parseStringLiteral for use in tests.
func (p *Parser) ParseStringLiteralForTest() (*StringLiteral, error) {
	return p.parseStringLiteral()
}

// Tokenizer exports the tokenizer type for tests.
type Tokenizer = tokenizer

// HasNextTokenForTest exports hasNextToken for use in tests.
func (t *tokenizer) HasNextTokenForTest() bool {
	return t.hasNextToken()
}

// GetNextForTest exports getNext for use in tests.
func (t *tokenizer) GetNextForTest() (*TokenValue, error) {
	return t.getNext()
}
