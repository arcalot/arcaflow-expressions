// Package ast is designed to tokenizer and parse jsonpath expressions. The  module contains representations of the
// components of the abstract tree representation of use of the grammar.
package ast

import "strconv"

const (
	invalid = "INVALID/MISSING"
)

// Node represents any node in the abstract syntax tree.
// Left() and Right() can return nil for any node types that do not
// have left and right sides.
type Node interface {
	Left() Node
	Right() Node
	String() string
}

// ValueLiteral represents any kind of literals that can be represented
// by the abstract syntax tree. Examples: ints, strings.
type ValueLiteral interface {
	Value() interface{}
	String() string
}

// StringLiteral represents a string literal value in the abstract syntax
// tree.
type StringLiteral struct {
	StrValue string
}

// String returns the string surrounded by double quotes.
func (l *StringLiteral) String() string {
	return `"` + l.StrValue + `"`
}

// Value returns the string contained. It can be cast to a string.
func (l *StringLiteral) Value() interface{} {
	return l.StrValue
}

// IntLiteral represents an integer literal value in the abstract syntax
// tree.
type IntLiteral struct {
	IntValue int
}

// String returns a string representation of the integer contained.
func (l *IntLiteral) String() string {
	return strconv.Itoa(l.IntValue)
}

// Value returns the integer contained.
func (l *IntLiteral) Value() interface{} {
	return l.IntValue
}

// Key represents any of the valid values that can be used in map/object
// bracket access. It can be either a sub-expression, represented as a
// Node, or as any supported literal, represented as a ValueLiteral.
// The one that is not being represented will be nil.
type Key struct {
	// A key can be either a literal or
	// a sub-expression that can be evaluated
	SubExpression Node
	Literal       ValueLiteral
}

// Right returns nil, because a key does not branch left and right.
func (k *Key) Right() Node {
	return nil
}

// Left returns nil, because a key does not branch left and right.
func (k *Key) Left() Node {
	return nil
}

// String returns the string from either the literal, or its sub-expression,
// surrounded by '(' and ')'.
func (k *Key) String() string {
	switch {
	case k.Literal != nil:
		return k.Literal.String()
	case k.SubExpression != nil:
		return "(" + k.SubExpression.String() + ")"
	default:
		return invalid
	}
}

// MapAccessor represents a part of the abstract syntax tree that is accessing
// the value at a key in an object.
type MapAccessor struct {
	LeftNode Node
	RightKey Key
}

// Right returns the key.
func (m *MapAccessor) Right() Node {
	return &m.RightKey
}

// Left returns the node being accessed.
func (m *MapAccessor) Left() Node {
	return m.LeftNode
}

// String returns the string from the accessed node, followed by '[', followed
// by the string from the key, followed by ']'.
func (m *MapAccessor) String() string {
	return m.LeftNode.String() + "[" + m.RightKey.String() + "]"
}

// Identifier represents a valid identifier in the abstract syntax tree.
type Identifier struct {
	IdentifierName string
}

// Right returns nil, because an identifier does not branch left and right.
func (i *Identifier) Right() Node {
	return nil
}

// Left returns nil, because an identifier does not branch left and right.
func (i *Identifier) Left() Node {
	return nil
}

// String returns the identifier name.
func (i *Identifier) String() string {
	return i.IdentifierName
}

// DotNotation represents the access of an identifier in a node.
type DotNotation struct {
	// The identifier on the right of the dot
	RightAccessIdentifier Node
	// The expression on the left could be one of several nodes.
	// I.e. An Identifier, a MapAccessor, or another DotNotation
	LeftAccessibleNode Node
}

// Right returns the identifier being accessed in the left node.
func (d *DotNotation) Right() Node {
	return d.RightAccessIdentifier
}

// Left returns the left node being accessed.
func (d *DotNotation) Left() Node {
	return d.LeftAccessibleNode
}

// String returns the string representing the left node, followed by '.',
// followed by the string representing the right identifier.
func (d *DotNotation) String() string {
	if d == nil {
		return invalid
	}
	var left, right string
	if d.LeftAccessibleNode != nil {
		left = d.LeftAccessibleNode.String()
	} else {
		left = invalid
	}
	if d.RightAccessIdentifier != nil {
		right = d.RightAccessIdentifier.String()
	} else {
		right = invalid
	}
	return left + "." + right
}
