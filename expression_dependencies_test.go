package expressions_test

import (
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
	"go.flow.arcalot.io/pluginsdk/schema"
)

func pathStrExtractor(value expressions.Path) string {
	return value.String()
}

var noKeyOrPastTerminalRequirements = expressions.UnpackRequirements{
	ExcludeDataRootPaths:     false,
	ExcludeFunctionRootPaths: true,
	StopAtTerminals:          true,
	IncludeKeys:              false,
}
var fullDataRequirements = expressions.UnpackRequirements{
	ExcludeDataRootPaths:     false,
	ExcludeFunctionRootPaths: true,
	StopAtTerminals:          false,
	IncludeKeys:              true,
}
var withFunctionsRequirements = expressions.UnpackRequirements{
	ExcludeDataRootPaths:     false,
	ExcludeFunctionRootPaths: false,
	StopAtTerminals:          false,
	IncludeKeys:              true,
}

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
				path, err := expr.Dependencies(schemaType, nil, nil, fullDataRequirements)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.foo.bar")
				pathWithoutExtra, err := expr.Dependencies(schemaType, nil, nil, noKeyOrPastTerminalRequirements)
				assert.NoError(t, err)
				if name == "any" {
					assert.Equals(t, pathWithoutExtra[0].String(), "$")
				} else {
					assert.Equals(t, pathWithoutExtra[0].String(), "$.foo.bar")
				}
			})

			t.Run("map-accessor", func(t *testing.T) {
				expr, err := expressions.New("$[\"foo\"].bar")
				assert.NoError(t, err)
				paths, err := expr.Dependencies(schemaType, nil, nil, fullDataRequirements)
				if name == "any" {
					// There isn't enough info to say this for an any type, but we set in the requirements
					// to include past-terminal (any) data types.
					assert.NoError(t, err)
					assert.Equals(t, len(paths), 1)
					assert.Equals(t, paths[0].String(), "$.foo.bar")
				} else {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "not supported for object/scope/ref")
				}
			})

			t.Run("map", func(t *testing.T) {
				expr, err := expressions.New("$.faz")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil, fullDataRequirements)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.faz")
			})

			t.Run("map-subkey", func(t *testing.T) {
				expr, err := expressions.New(`$.faz["foo"]`)
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil, fullDataRequirements)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.faz.foo")
			})
			t.Run("subexpression-invalid", func(t *testing.T) {
				expr, err := expressions.New("$.foo[$.faz.foo]")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil, fullDataRequirements)
				if name == "scope" {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					// Order isn't deterministic, so use contains and length validations.
					assert.Equals(t, len(path), 2)
					assert.SliceContainsExtractor(t, pathStrExtractor, "$.foo", path)
					assert.SliceContainsExtractor(t, pathStrExtractor, "$.faz.foo", path)
				}
			})

			t.Run("subexpression", func(t *testing.T) {
				expr, err := expressions.New("$.faz[$.foo.bar]")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil, nil, fullDataRequirements)
				assert.NoError(t, err)
				// Order isn't deterministic, so use contains and length validations.
				assert.Equals(t, len(path), 2)
				assert.SliceContainsExtractor(t, pathStrExtractor, "$.faz", path)
				assert.SliceContainsExtractor(t, pathStrExtractor, "$.foo.bar", path)
			})

			t.Run("list-literal-key", func(t *testing.T) {
				expr, err := expressions.New("$.int_list[0]")
				assert.NoError(t, err)
				pathWithExtra, err := expr.Dependencies(schemaType, nil, nil, fullDataRequirements)
				assert.NoError(t, err)
				assert.Equals(t, len(pathWithExtra), 1)
				assert.Equals(t, pathWithExtra[0].String(), "$.int_list.0")
				pathWithoutExtra, err := expr.Dependencies(schemaType, nil, nil, noKeyOrPastTerminalRequirements)
				assert.NoError(t, err)
				assert.Equals(t, len(pathWithoutExtra), 1)
				if name == "any" {
					assert.Equals(t, pathWithoutExtra[0].String(), "$")
				} else {
					assert.Equals(t, pathWithoutExtra[0].String(), "$.int_list")
				}

			})

			t.Run("list-subexpr-key", func(t *testing.T) {
				expr, err := expressions.New("$.int_list[$.simple_int]")
				assert.NoError(t, err)
				pathWithExtra, err := expr.Dependencies(schemaType, nil, nil, fullDataRequirements)
				assert.NoError(t, err)
				assert.Equals(t, len(pathWithExtra), 2)
				assert.SliceContainsExtractor(t, pathStrExtractor, "$.int_list", pathWithExtra)
				assert.SliceContainsExtractor(t, pathStrExtractor, "$.simple_int", pathWithExtra)

			})
		})
	}
}

func TestLiteralDependencyResolution(t *testing.T) {
	expr, err := expressions.New(`"test"`)
	assert.NoError(t, err)
	path, err := expr.Dependencies(testScope, nil, nil, noKeyOrPastTerminalRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(path), 0) // Does not depend on anything.
}

func TestImplicitRootDependencyResolution(t *testing.T) {
	// Tests the root being implicit. $ should be first, even though it's not explicitly specified in the input string.
	expr, err := expressions.New("foo.bar")
	assert.NoError(t, err)
	path, err := expr.Dependencies(testScope, nil, nil, noKeyOrPastTerminalRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(path), 1)
	assert.Equals(t, path[0].String(), "$.foo.bar")
}

func TestFunctionDependencyResolution_void(t *testing.T) {
	voidFunc, err := schema.NewCallableFunction("voidFunc", make([]schema.Type, 0), nil, false, nil, func() {})
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"voidFunc": voidFunc}
	expr, err := expressions.New(`voidFunc()`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 0)
	dependencyTree, err = expr.Dependencies(testScope, funcMap, nil, withFunctionsRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 1)
	assert.Equals(t, dependencyTree[0].String(), "voidFunc")
}

func TestFunctionDependencyResolution_error_unknown_func(t *testing.T) {
	expr, err := expressions.New(`missing()`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, map[string]schema.Function{}, nil, noKeyOrPastTerminalRequirements)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find function")
}

func TestFunctionDependencyResolution_singleParam(t *testing.T) {
	intInFunc, err := schema.NewCallableFunction(
		"intIn",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		nil,
		false,
		nil,
		func(a int64) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"intIn": intInFunc}

	expr, err := expressions.New(`intIn(5)`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 0)
	dependencyTree, err = expr.Dependencies(testScope, funcMap, nil, withFunctionsRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 1)
	assert.Equals(t, dependencyTree[0].String(), "intIn")

	expr, err = expressions.New(`intIn($.simple_int)`)
	assert.NoError(t, err)
	dependencyTree, err = expr.Dependencies(testScope, funcMap, nil, withFunctionsRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 2)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.simple_int", dependencyTree)
	assert.SliceContainsExtractor(t, pathStrExtractor, "intIn", dependencyTree)
}

func TestFunctionDependencyResolution_duplicateDependency(t *testing.T) {
	// Tests that it doesn't include the same dependency twice.
	intInFunc, err := schema.NewCallableFunction(
		"twoIntIn",
		[]schema.Type{
			schema.NewIntSchema(nil, nil, nil),
			schema.NewIntSchema(nil, nil, nil),
		},
		nil,
		false,
		nil,
		func(a int64, b int64) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"twoIntIn": intInFunc}

	expr, err := expressions.New(`twoIntIn($.simple_int, $.simple_int)`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 1)
	assert.Equals(t, dependencyTree[0].String(), "$.simple_int")
}

func TestFunctionDependencyResolution_manyDependencies(t *testing.T) {
	// This tests two dependencies from the function, and one from the map access.
	// This is necessary because we need to validate that the dependencies are merged correctly.
	intInFunc, err := schema.NewCallableFunction(
		"test",
		[]schema.Type{
			schema.NewIntSchema(nil, nil, nil),
			schema.NewStringSchema(nil, nil, nil),
		},
		schema.NewStringSchema(nil, nil, nil),
		false,
		nil,
		func(a int64, b string) string { return b },
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"test": intInFunc}

	expr, err := expressions.New(`$.faz[test($.simple_int, $.simple_str)]`)
	assert.NoError(t, err)
	dependencyPaths, err := expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyPaths), 3)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.faz", dependencyPaths)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.simple_int", dependencyPaths)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.simple_str", dependencyPaths)
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
		false,
		nil,
		func(a int64, b int64, c string) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"test": testFunc}

	expr, err := expressions.New(`test(5, $.simple_int, $.simple_str)`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 2)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.simple_int", dependencyTree)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.simple_str", dependencyTree)
}

func TestFunctionDependencyResolution_compoundFunctions(t *testing.T) {
	intInOutFunc, err := schema.NewCallableFunction(
		"intInOut",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		schema.NewIntSchema(nil, nil, nil),
		false,
		nil,
		func(a int64) int64 { return a },
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"intInOut": intInOutFunc}

	expr, err := expressions.New(`intInOut(intInOut($.simple_int))`)
	assert.NoError(t, err)
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil, withFunctionsRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 2)
	assert.SliceContainsExtractor(t, pathStrExtractor, "$.simple_int", dependencyTree)
	assert.SliceContainsExtractor(t, pathStrExtractor, "intInOut", dependencyTree)
}

func TestFunctionDependencyResolution_error_wrongType(t *testing.T) {
	intInFunc, err := schema.NewCallableFunction(
		"intIn",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		nil,
		false,
		nil,
		func(a int64) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"intIn": intInFunc}

	expr, err := expressions.New(`intIn("wrongType")`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
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
		false,
		nil,
		func(a int64) {},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"intIn": intInFunc}

	expr, err := expressions.New(`intIn(5, 5)`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
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
			assert.Equals(t, len(inputType), 1)
			return inputType[0], nil
		},
	)
	assert.NoError(t, err)
	intInFunc, err := schema.NewCallableFunction(
		"intIn",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		nil,
		false,
		nil,
		func(a int64) {},
	)
	assert.NoError(t, err)
	strInFunc, err := schema.NewCallableFunction(
		"strIn",
		[]schema.Type{schema.NewStringSchema(nil, nil, nil)},
		nil,
		false,
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
	dependencyTree, err := expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 0)
	// Test identity returning str when given str
	expr, err = expressions.New(`strIn(identity("test"))`)
	assert.NoError(t, err)
	dependencyTree, err = expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
	assert.NoError(t, err)
	assert.Equals(t, len(dependencyTree), 0)
	// Test type mismatch
	expr, err = expressions.New(`strIn(identity(1))`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported data type")
	// Second test type mismatch
	expr, err = expressions.New(`intIn(identity("test"))`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, funcMap, nil, noKeyOrPastTerminalRequirements)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported data type")
}
