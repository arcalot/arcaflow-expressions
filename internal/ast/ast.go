// Package ast is designed to tokenizer and parse jsonpath expressions. The  module contains representations of the
// components of the abstract tree representation of use of the grammar.
package ast

import (
	"fmt"
	"strconv"
)

const (
	invalid = "INVALID/MISSING"
)

// Node represents any node in the abstract syntax tree.
// Left() and Right() can return nil for any node types that do not
// have left and right sides.
type Node interface {
	String() string
}

type BinaryNode interface {
	Left() Node
	Right() Node
}

type NNode interface {
	NumChildren() int
	GetChild(int) (Node, error)
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
	IntValue int64
}

// String returns a string representation of the integer contained.
func (l *IntLiteral) String() string {
	return strconv.FormatInt(l.IntValue, 10) // Format in base 10
}

// Value returns the integer contained.
func (l *IntLiteral) Value() interface{} {
	return l.IntValue
}

// FloatLiteral represents a floating point literal value in the abstract syntax
// tree.
type FloatLiteral struct {
	FloatValue float64
}

// String returns a string representation of the float contained.
func (l *FloatLiteral) String() string {
	// 'f' for full float, instead of exponential format.
	// The third arg, prec, is -1 to give an exact output.
	// The fourth arg specifies that we're using 64-bit floats.
	return strconv.FormatFloat(l.FloatValue, 'f', -1, 64)
}

// Value returns the float contained.
func (l *FloatLiteral) Value() interface{} {
	return l.FloatValue
}

// BooleanLiteral represents a boolean literal value in the abstract syntax
// tree. true or false
type BooleanLiteral struct {
	BooleanValue bool
}

// String returns a string representation of the boolean contained.
func (l *BooleanLiteral) String() string {
	return strconv.FormatBool(l.BooleanValue)
}

// Value returns the float contained.
func (l *BooleanLiteral) Value() interface{} {
	return l.BooleanValue
}

// BracketAccessor represents a part of the abstract syntax tree that is accessing
// the value at a key in a map/object, or index of a list.
// The format is the value to the left, followed by an open/right square bracket, followed
// by the key, followed by a close/left square bracket.
type BracketAccessor struct {
	LeftNode        Node
	RightExpression Node
}

// Right returns the key.
func (m *BracketAccessor) Right() Node {
	return m.RightExpression
}

// Left returns the node being accessed.
func (m *BracketAccessor) Left() Node {
	return m.LeftNode
}

// String returns the string from the accessed node, followed by '[', followed
// by the string from the key, followed by ']'.
func (m *BracketAccessor) String() string {
	return m.LeftNode.String() + "[" + m.RightExpression.String() + "]"
}

// Identifier represents a valid identifier in the abstract syntax tree.
type Identifier struct {
	IdentifierName string
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

// FunctionCall represents a call to a function with 0 or more parameters.
type FunctionCall struct {
	FuncIdentifier *Identifier
	ArgumentInputs *ArgumentList
}

// Right returns nil, because an identifier does not branch left and right.
func (f *FunctionCall) Right() Node {
	return f.ArgumentInputs
}

// Left returns nil, because an identifier does not branch left and right.
func (f *FunctionCall) Left() Node {
	return f.FuncIdentifier
}

// String returns the identifier name.
func (f *FunctionCall) String() string {
	return f.FuncIdentifier.String() + "(" + f.ArgumentInputs.String() + ")"
}

// ArgumentList is a list of expressions being used to specify values to input into function parameters.
type ArgumentList struct {
	Arguments []Node
}

func (l *ArgumentList) NumChildren() int {
	return len(l.Arguments)
}
func (l *ArgumentList) GetChild(index int) (Node, error) {
	if index >= len(l.Arguments) {
		return nil, fmt.Errorf("index requested is out of bounds. Got %d, expected less than %d",
			index, len(l.Arguments))
	}
	return l.Arguments[index], nil
}

// String gives a comma-separated list of the arguments
func (l *ArgumentList) String() string {
	if len(l.Arguments) == 0 {
		return ""
	}
	result := l.Arguments[0].String()
	for i := 1; i < len(l.Arguments); i++ {
		result += ", " + l.Arguments[i].String()
	}
	return result
}

type MathOperationType int

const (
	Invalid MathOperationType = iota
	Add
	Subtract
	Multiply
	Divide
	Modulus
	Power
	EqualTo
	NotEqualTo
	GreaterThan
	LessThan
	GreaterThanEqualTo
	LessThanEqualTo
	And
	Or
	Not
)

func (e MathOperationType) String() string {
	switch e {
	case Invalid:
		return "INVALID"
	case Add:
		return "+"
	case Subtract:
		return "-"
	case Multiply:
		return "*"
	case Divide:
		return "÷"
	case Modulus:
		return "%"
	case Power:
		return "^"
	case EqualTo:
		return "=="
	case NotEqualTo:
		return "!="
	case GreaterThan:
		return ">"
	case LessThan:
		return "<"
	case GreaterThanEqualTo:
		return ">="
	case LessThanEqualTo:
		return "<="
	case And:
		return "&&"
	case Or:
		return "||"
	case Not:
		return "!"
	default:
		return "ENTRY MISSING"
	}
}

type BinaryOperation struct {
	LeftNode  Node
	RightNode Node
	Operation MathOperationType
}

func (b *BinaryOperation) Right() Node {
	return b.RightNode
}

func (b *BinaryOperation) Left() Node {
	return b.LeftNode
}

// String returns the left node, followed by the operator, followed by the right node.
// The left and right nodes are clarified with (), because context that determined order of
// operations, like parentheses, are not explicitly retained in the tree. But the structure
// of the tree represents the evaluation order present in the original expression.
func (b *BinaryOperation) String() string {
	return "(" + b.LeftNode.String() + ") " + b.Operation.String() + " (" + b.RightNode.String() + ")"
}

type UnaryOperation struct {
	LeftOperation MathOperationType
	RightNode     Node
}

// String returns the operation, followed by the string representation of the right node.
// The wrapped node is surrounded by parentheses to remove ambiguity.
func (b *UnaryOperation) String() string {
	return b.LeftOperation.String() + "(" + b.RightNode.String() + ") "
}
