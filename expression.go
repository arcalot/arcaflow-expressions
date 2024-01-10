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
	Type(schema schema.Scope, functions map[string]schema.Function, workflowContext map[string][]byte) (schema.Type, error)
	// Dependencies traverses the passed scope and evaluates the items this expression depends on. This is useful to
	// construct a dependency tree based on expressions.
	// Returns the path to the object in the schema that it depends on, or nil if it's a literal that doesn't depend
	// on it.
	// includeExtraneous specifies whether to include things not explicitly present in the schema, like values within
	// any types, or indexes in lists.
	Dependencies(schema schema.Type, functions map[string]schema.Function, workflowContext map[string][]byte, includeExtraneous bool) ([]Path, error)
	// Evaluate evaluates the expression on the given data set regardless of any
	// schema. The caller is responsible for validating the expected schema.
	Evaluate(data any, functions map[string]schema.CallableFunction, workflowContext map[string][]byte) (any, error)
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

func (e expression) Type(scope schema.Scope, functions map[string]schema.Function, workflowContext map[string][]byte) (schema.Type, error) {
	tree := PathTree{
		PathItem: "$",
		Subtrees: nil,
	}
	d := &dependencyContext{
		rootType:        scope,
		rootPath:        tree,
		workflowContext: workflowContext,
		functions:       functions,
	}
	result, _, _, err := d.rootDependencies(e.ast)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e expression) Dependencies(
	scope schema.Type,
	functions map[string]schema.Function,
	workflowContext map[string][]byte,
	includeExtraneous bool,
) ([]Path, error) {
	root := PathTree{
		PathItem: "$",
		Subtrees: nil,
	}
	d := &dependencyContext{
		rootType:        scope,
		rootPath:        root,
		workflowContext: workflowContext,
		functions:       functions,
	}
	_, _, dependencies, err := d.rootDependencies(e.ast)
	if err != nil {
		return nil, err
	}
	// Now convert to paths, saving only unique values.
	finalDependencyMap := make(map[string]Path)
	for _, dependencyTree := range dependencies {
		unpackedDependencies := dependencyTree.Unpack(includeExtraneous)
		for _, dependency := range unpackedDependencies {
			asString := dependency.String()
			_, dependencyExists := finalDependencyMap[asString]
			if !dependencyExists {
				finalDependencyMap[asString] = dependency
			}
		}
	}
	finalDependencies := make([]Path, len(finalDependencyMap))
	i := 0
	for _, v := range finalDependencyMap {
		finalDependencies[i] = v
		i += 1
	}
	return finalDependencies, nil
}

func (e expression) Evaluate(data any, functions map[string]schema.CallableFunction, workflowContext map[string][]byte) (any, error) {
	context := &evaluateContext{
		functions:       functions,
		rootData:        data,
		workflowContext: workflowContext,
	}
	return context.evaluate(e.ast, data)
}
