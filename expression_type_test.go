package expressions_test

import (
	"reflect"
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
	"go.flow.arcalot.io/pluginsdk/schema"
)

func TestTypeEvaluation(t *testing.T) {
	t.Run("object", func(t *testing.T) {
		expr, err := expressions.New("$.foo.bar")
		assert.NoError(t, err)
		resultType, err := expr.Type(testScope, nil, nil)
		assert.NoError(t, err)
		assert.Equals(t, resultType.TypeID(), schema.TypeIDString)
	})

	t.Run("map-accessor", func(t *testing.T) {
		expr, err := expressions.New(`$.faz["abc"]`)
		assert.NoError(t, err)
		resultType, err := expr.Type(testScope, nil, nil)
		assert.NoError(t, err)
		// The value type of the map is object.
		assert.Equals(t, resultType.TypeID(), schema.TypeIDObject)
	})

	t.Run("map", func(t *testing.T) {
		expr, err := expressions.New("$.faz")
		assert.NoError(t, err)
		resultType, err := expr.Type(testScope, nil, nil)
		assert.NoError(t, err)
		assert.Equals(t, resultType.TypeID(), schema.TypeIDMap)
	})

	t.Run("subexpression-invalid", func(t *testing.T) {
		expr, err := expressions.New("$.foo[$.faz.foo]")
		assert.NoError(t, err)
		_, err = expr.Type(testScope, nil, nil)
		assert.Error(t, err)
	})

	t.Run("subexpression", func(t *testing.T) {
		expr, err := expressions.New("$.faz[$.foo.bar]")
		assert.NoError(t, err)
		resultType, err := expr.Type(testScope, nil, nil)
		assert.NoError(t, err)
		assert.Equals(t, resultType.TypeID(), schema.TypeIDObject)

	})

	t.Run("list-value", func(t *testing.T) {
		expr, err := expressions.New("$.int_list")
		assert.NoError(t, err)
		resultType, err := expr.Type(testScope, nil, nil)
		assert.NoError(t, err)
		assert.Equals(t, resultType.TypeID(), schema.TypeIDList)
	})

	t.Run("list-item", func(t *testing.T) {
		expr, err := expressions.New("$.int_list[0]")
		assert.NoError(t, err)
		resultType, err := expr.Type(testScope, nil, nil)
		assert.NoError(t, err)
		assert.Equals(t, resultType.TypeID(), schema.TypeIDInt)
	})

	t.Run("any-schema", func(t *testing.T) {
		expr, err := expressions.New("$.simple_any.a.b")
		assert.NoError(t, err)
		resultType, err := expr.Type(testScope, nil, nil)
		assert.NoError(t, err)
		assert.Equals(t, resultType.TypeID(), schema.TypeIDAny)
	})
}

func TestLiteralTypeResolution(t *testing.T) {
	expr, err := expressions.New(`"test"`)
	assert.NoError(t, err)
	typeResult, err := expr.Type(testScope, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewStringSchema(nil, nil, nil))
}

func TestFunctionTypeResolution_void(t *testing.T) {
	voidFunc, err := schema.NewCallableFunction("voidFunc", make([]schema.Type, 0), nil, false, nil, func() {})
	assert.NoError(t, err)

	expr, err := expressions.New(`voidFunc()`)
	assert.NoError(t, err)
	typeResult, err := expr.Type(testScope, map[string]schema.Function{"voidFunc": voidFunc}, nil)
	assert.NoError(t, err)
	assert.Nil(t, typeResult)
}

func TestFunctionTypeResolution_compoundFunctions(t *testing.T) {
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
	typeResult, err := expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewIntSchema(nil, nil, nil))
}

func TestFunctionTypeResolution_dynamicTyping(t *testing.T) {
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
	funcMap := map[string]schema.Function{
		"identity": identityFunc,
	}
	// Test identity returning int when given int
	expr, err := expressions.New(`identity(1)`)
	assert.NoError(t, err)
	typeResult, err := expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewIntSchema(nil, nil, nil))
	// Test identity returning str when given str
	expr, err = expressions.New(`identity("test")`)
	assert.NoError(t, err)
	typeResult, err = expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewStringSchema(nil, nil, nil))
	// Same but with a reference instead of a literal
	// Test identity returning int when given int
	expr, err = expressions.New(`identity($.simple_int)`)
	assert.NoError(t, err)
	typeResult, err = expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewIntSchema(nil, nil, nil))
	// Test identity returning str when given str
	expr, err = expressions.New(`identity($.simple_str)`)
	assert.NoError(t, err)
	typeResult, err = expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewStringSchema(nil, nil, nil))
}

func TestFunctionTypeResolution_advancedDynamicTyping(t *testing.T) {
	// It's an identity function. It returns what it's given.
	toListFunc, err := schema.NewDynamicCallableFunction(
		"toList",
		[]schema.Type{schema.NewAnySchema()},
		nil,
		func(a any) (any, error) {
			aVal := reflect.ValueOf(a)
			result := reflect.MakeSlice(reflect.SliceOf(aVal.Type()), 2, 2)
			result.Index(0).Set(aVal)
			result.Index(1).Set(aVal)
			return result.Interface(), nil
		},
		func(inputType []schema.Type) (schema.Type, error) {
			assert.Equals(t, len(inputType), 1)
			return schema.NewListSchema(inputType[0], nil, nil), nil
		},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{
		"toList": toListFunc,
	}
	// Test returning []int when given int
	expr, err := expressions.New(`toList(1)`)
	assert.NoError(t, err)
	typeResult, err := expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult,
		schema.NewListSchema(
			schema.NewIntSchema(nil, nil, nil),
			nil,
			nil,
		),
	)
	// Test returning []str when given str
	expr, err = expressions.New(`toList("test")`)
	assert.NoError(t, err)
	typeResult, err = expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult,
		schema.NewListSchema(
			schema.NewStringSchema(nil, nil, nil),
			nil,
			nil,
		),
	)
	// Test toList followed by indexing
	expr, err = expressions.New(`toList("test")[0]`)
	assert.NoError(t, err)
	typeResult, err = expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult,
		schema.NewStringSchema(nil, nil, nil),
	)
}
