package expressions

import (
	"fmt"

	"go.flow.arcalot.io/expressions/internal/ast"
	"go.flow.arcalot.io/pluginsdk/schema"
)

// New parses the specified expression and returns the expression structure.
func New(expressionString string) (Expression, error) {
	parser, err := ast.InitParser(expressionString, "workflow.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression: %s (%w)", expressionString, err)
	}
	exprAst, err := parser.ParseExpression()
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression: %s (%v)", expressionString, err)
	}

	return &expression{
		ast:        exprAst,
		expression: expressionString,
	}, nil
}

// Expression is an interface describing how expressions should behave.
type Expression interface {
	// Type evaluates the expression and evaluates the type on the specified schema.
	Type(schema schema.Scope, workflowContext map[string][]byte) (schema.Type, error)
	// Dependencies traverses the passed scope and evaluates the items this expression depends on. This is useful to
	// construct a dependency tree based on expressions.
	Dependencies(schema schema.Scope, workflowContext map[string][]byte) ([]Path, error)
	// Evaluate evaluates the expression on the given data set regardless of any
	// schema. The caller is responsible for validating the expected schema.
	Evaluate(data any, workflowContext map[string][]byte) (any, error)
	// String returns the string representation of the expression.
	String() string
}

// expression is the implementation of Expression. It holds the original expression, as well as the parsed AST.
type expression struct {
	expression string
	ast        ast.Node
}

func (e expression) String() string {
	return e.expression
}

func (e expression) Type(scope schema.Scope, workflowContext map[string][]byte) (schema.Type, error) {
	tree := &PathTree{
		PathItem: "$",
		Subtrees: nil,
	}
	d := &dependencyContext{
		rootType:        scope,
		rootPath:        tree,
		workflowContext: workflowContext,
	}
	result, _, err := d.dependencies(e.ast, scope, tree)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e expression) Dependencies(scope schema.Scope, workflowContext map[string][]byte) ([]Path, error) {
	tree := &PathTree{
		PathItem: "$",
		Subtrees: nil,
	}
	d := &dependencyContext{
		rootType:        scope,
		rootPath:        tree,
		workflowContext: workflowContext,
	}
	_, _, err := d.dependencies(e.ast, scope, tree)
	if err != nil {
		return nil, err
	}
	return tree.Unpack(), nil
}

func (e expression) Evaluate(data any, workflowContext map[string][]byte) (any, error) {
	return evaluate(e.ast, data, data, workflowContext)
}
