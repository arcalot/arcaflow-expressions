package expressions_test

import (
	"fmt"
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
	"go.flow.arcalot.io/pluginsdk/schema"
)

func TestDependencyResolution(t *testing.T) {
	scopes := map[string]schema.Type{
		"scope": testScope,
		"any":   schema.NewAnySchema(),
	}
	// All of these can apply to multiple types, so we'll iterate over the possibilities.
	for name, schemaType := range scopes {
		name := name
		schemaType := schemaType
		t.Run(name, func(t *testing.T) {
			t.Run("object", func(t *testing.T) {
				expr, err := expressions.New("$.foo.bar")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.foo.bar")
			})

			t.Run("map-accessor", func(t *testing.T) {
				expr, err := expressions.New("$[\"foo\"].bar")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.foo.bar")
			})

			t.Run("map", func(t *testing.T) {
				expr, err := expressions.New("$.faz")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.faz")
			})

			t.Run("map-subkey", func(t *testing.T) {
				expr, err := expressions.New("$.faz.foo")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.faz.foo")
			})
			t.Run("subexpression-invalid", func(t *testing.T) {
				expr, err := expressions.New("$.foo[$.faz.foo]")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil)
				if name == "scope" {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equals(t, path[0].String(), "$.foo")
					assert.Equals(t, path[1].String(), "$.faz.foo")
				}
			})

			t.Run("subexpression", func(t *testing.T) {
				expr, err := expressions.New("$.faz[$.foo.bar]")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil)
				if name == "scope" {
					assert.NoError(t, err)
					assert.Equals(t, len(path), 2)
					assert.Equals(t, path[0].String(), "$.faz.*")
					assert.Equals(t, path[1].String(), "$.foo.bar")
				} else {
					assert.NoError(t, err)
					assert.Equals(t, len(path), 2)
					assert.Equals(t, path[0].String(), "$.faz")
					assert.Equals(t, path[1].String(), "$.foo.bar")
				}
			})
		})
	}
}

func TestLiteralDependencyResolution(t *testing.T) {
	expr, err := expressions.New(`"test"`)
	assert.NoError(t, err)
	path, err := expr.Dependencies(testScope, nil, nil)
	assert.NoError(t, err)
	assert.Equals(t, len(path), 1) // Does not depend on anything.
}

func TestFunctionDependencyResolution_void(t *testing.T) {
	voidFunc, err := schema.NewCallableFunction("voidFunc", make([]schema.Type, 0), nil, nil, func() {})
	assert.NoError(t, err)

	expr, err := expressions.New(`voidFunc()`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, map[string]schema.Function{"voidFunc": voidFunc}, nil)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 1)
	assert.Equals(t, dependencyTree[0].String(), "$")
}

func TestFunctionDependencyResolution_error_unknown_func(t *testing.T) {
	expr, err := expressions.New(`missing()`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, map[string]schema.Function{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find function")
}

func TestFunctionDependencyResolution_singleParam(t *testing.T) {
	intInFunc, err := schema.NewCallableFunction(
		"intIn",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		nil,
		nil,
		func(a int64) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"intIn": intInFunc}

	expr, err := expressions.New(`intIn(5)`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 1)
	assert.Equals(t, dependencyTree[0].String(), "$")

	expr, err = expressions.New(`intIn($.simple_int)`)
	assert.NoError(t, err)
	dependencyTree, err = expr.Dependencies(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 1)
	assert.Equals(t, dependencyTree[0].String(), "$.simple_int")
}

func TestFunctionDependencyResolution_multiParam(t *testing.T) {
	testFunc, err := schema.NewCallableFunction(
		"test",
		[]schema.Type{
			schema.NewIntSchema(nil, nil, nil),
			schema.NewIntSchema(nil, nil, nil),
			schema.NewStringSchema(nil, nil, nil),
		},
		nil,
		nil,
		func(a int64, b int64, c string) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"test": testFunc}

	expr, err := expressions.New(`test(5, $.simple_int, $.simple_str)`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 2)
	assert.Equals(t, dependencyTree[0].String(), "$.simple_int")
	assert.Equals(t, dependencyTree[1].String(), "$.simple_str")
}

func TestFunctionDependencyResolution_compoundFunctions(t *testing.T) {
	intInOutFunc, err := schema.NewCallableFunction(
		"intInOut",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		schema.NewIntSchema(nil, nil, nil),
		nil,
		func(a int64) int64 { return a },
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"intInOut": intInOutFunc}

	expr, err := expressions.New(`intInOut(intInOut($.simple_int))`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 1)
	assert.Equals(t, dependencyTree[0].String(), "$.simple_int")
}

func TestFunctionDependencyResolution_error_wrongType(t *testing.T) {
	intInFunc, err := schema.NewCallableFunction(
		"intIn",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		nil,
		nil,
		func(a int64) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"intIn": intInFunc}

	expr, err := expressions.New(`intIn("wrongType")`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, funcMap, nil)
	assert.Error(t, err)
	// Validate that it detected a type problem
	assert.Contains(t, err.Error(), "error while validating arg/param type compatibility")
	// Validate that the function schema is in it
	assert.Contains(t, err.Error(), "intIn(integer) void")
}

func TestFunctionDependencyResolution_error_wrongArgCount(t *testing.T) {
	intInFunc, err := schema.NewCallableFunction(
		"intIn",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		nil,
		nil,
		func(a int64) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"intIn": intInFunc}

	expr, err := expressions.New(`intIn(5, 5)`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, funcMap, nil)
	assert.Error(t, err)
	// Validate that it detected a type problem
	assert.Contains(t, err.Error(), "Expected 1 args, got 2 args")
	// Validate that the function schema is in it
	assert.Contains(t, err.Error(), "intIn(integer) void")
}

func TestFunctionDependencyResolution_dynamicTyping(t *testing.T) {
	// It's an identity function. It returns what it's given.
	identityFunc, err := schema.NewDynamicCallableFunction(
		"identity",
		[]schema.Type{schema.NewAnySchema()},
		nil,
		func(a any) (any, error) { return a, nil },
		func(inputType []schema.Type) (schema.Type, error) {
			if len(inputType) == 1 {
				return inputType[0], nil
			} else {
				return nil, fmt.Errorf("incorrect param count")
			}
		},
	)
	assert.NoError(t, err)
	intInFunc, err := schema.NewCallableFunction(
		"intIn",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		nil,
		nil,
		func(a int64) {},
	)
	assert.NoError(t, err)
	strInFunc, err := schema.NewCallableFunction(
		"strIn",
		[]schema.Type{schema.NewStringSchema(nil, nil, nil)},
		nil,
		nil,
		func(a string) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{
		"identity": identityFunc,
		"intIn":    intInFunc,
		"strIn":    strInFunc,
	}
	// Test identity returning int when given int
	expr, err := expressions.New(`intIn(identity(1))`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 1)
	assert.Equals(t, dependencyTree[0].String(), "$")
	// Test identity returning str when given str
	expr, err = expressions.New(`strIn(identity("test"))`)
	assert.NoError(t, err)
	dependencyTree, err = expr.Dependencies(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 1)
	assert.Equals(t, dependencyTree[0].String(), "$")
	// Test type mismatch
	expr, err = expressions.New(`strIn(identity(1))`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, funcMap, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported data type")
	// Second test type mismatch
	expr, err = expressions.New(`intIn(identity("test"))`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, funcMap, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported data type")
}
