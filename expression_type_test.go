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

func TestTypeResolution_BinaryMathHomogeneousIntLiterals(t *testing.T) {
	// Two ints added should give an int
	expr, err := expressions.New("5 + 5")
	assert.NoError(t, err)
	typeResult, err := expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewIntSchema(nil, nil, nil))
	expr, err = expressions.New("5 * 5")
	assert.NoError(t, err)
	typeResult, err = expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewIntSchema(nil, nil, nil))
}

func TestTypeResolution_BinaryConcatenateStrings(t *testing.T) {
	expr, err := expressions.New(`"5" + "5"`)
	assert.NoError(t, err)
	typeResult, err := expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewStringSchema(nil, nil, nil))
}

func TestTypeResolution_BinaryMathHomogeneousIntReference(t *testing.T) {
	// Two ints added should give an int. One int is a reference.
	expr, err := expressions.New("5 + $.simple_int")
	assert.NoError(t, err)
	typeResult, err := expr.Type(testScope, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewIntSchema(nil, nil, nil))
	expr, err = expressions.New("$.simple_int + 5")
	assert.NoError(t, err)
	typeResult, err = expr.Type(testScope, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewIntSchema(nil, nil, nil))
}

func TestTypeResolution_BinaryMathHomogeneousFloatLiterals(t *testing.T) {
	// Two floats added, subtracted, multiplied, and divided should give floats
	expr, err := expressions.New("5.0 / 5.0")
	assert.NoError(t, err)
	typeResult, err := expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewFloatSchema(nil, nil, nil))
	expr, err = expressions.New("5.0 + 5.0")
	assert.NoError(t, err)
	typeResult, err = expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewFloatSchema(nil, nil, nil))
	expr, err = expressions.New("5.0 - 5.0")
	assert.NoError(t, err)
	typeResult, err = expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewFloatSchema(nil, nil, nil))
	expr, err = expressions.New("5.0 * 5.0")
	assert.NoError(t, err)
	typeResult, err = expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewFloatSchema(nil, nil, nil))
	expr, err = expressions.New("5.0 % 5.0")
	assert.NoError(t, err)
	typeResult, err = expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewFloatSchema(nil, nil, nil))
	expr, err = expressions.New("5.0 ^ 5.0")
	assert.NoError(t, err)
	typeResult, err = expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewFloatSchema(nil, nil, nil))
}

func TestTypeResolution_Error_BinaryMathHeterogeneousLiterals(t *testing.T) {
	// Test literal int and float math, error mixed types
	expr, err := expressions.New("5 + 5.0")
	assert.NoError(t, err)
	_, err = expr.Type(nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "types do not match")
}

func TestTypeResolution_UnaryOperation(t *testing.T) {
	// Tests that the unary operator properly passes the type upwards.
	expr, err := expressions.New("-5")
	assert.NoError(t, err)
	typeResult, err := expr.Type(nil, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewIntSchema(nil, nil, nil))
}

func TestTypeResolution_TestMixedMathAndFunc(t *testing.T) {
	// Test int and float math, mixed with function.
	intInFunc, err := schema.NewCallableFunction(
		"intToFloat",
		[]schema.Type{schema.NewIntSchema(nil, nil, nil)},
		schema.NewFloatSchema(nil, nil, nil),
		false,
		nil,
		func(a int64) float64 {
			return float64(a)
		},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"intToFloat": intInFunc}

	expr, err := expressions.New("5.0 + intToFloat($.simple_int)")
	assert.NoError(t, err)
	typeResult, err := expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewFloatSchema(nil, nil, nil))
}

func TestTypeResolution_Error_NonBoolType(t *testing.T) {
	// Non-bool type for operation that requires boolean types
	expr, err := expressions.New(`0 && 1`)
	assert.NoError(t, err)
	_, err = expr.Dependencies(testScope, nil, nil, fullDataRequirements)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-boolean type")
}

func TestTypeResolution_TestMixedOperations(t *testing.T) {
	// Test int and float math, mixed with function.
	intInFunc, err := schema.NewCallableFunction(
		"giveFloat",
		[]schema.Type{},
		schema.NewFloatSchema(nil, nil, nil),
		false,
		nil,
		func() float64 {
			return 5.5
		},
	)
	assert.NoError(t, err)
	funcMap := map[string]schema.Function{"giveFloat": intInFunc}

	expr, err := expressions.New("1.0 == (5.0 / giveFloat()) && !true")
	assert.NoError(t, err)
	typeResult, err := expr.Type(testScope, funcMap, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewBoolSchema())
}

func TestDependencyResolution_Error_TestInvalidTypeOnBoolean(t *testing.T) {
	// Tests invalid type for relational operator
	expr, err := expressions.New("true > false")
	assert.NoError(t, err)
	_, err = expr.Type(testScope, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attempted quantity inequality comparison operation")
}

func TestDependencyResolution_TestSizeComparison(t *testing.T) {
	expr, err := expressions.New("5 > 6")
	assert.NoError(t, err)
	typeResult, err := expr.Type(testScope, nil, nil)
	assert.NoError(t, err)
	assert.Equals[schema.Type](t, typeResult, schema.NewBoolSchema())
}

func TestDependencyResolution_Error_TestInvalidNot(t *testing.T) {
	// 'not' expects boolean
	expr, err := expressions.New("!5")
	assert.NoError(t, err)
	_, err = expr.Type(testScope, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-boolean type")
}

func TestDependencyResolution_Error_TestInvalidNegation(t *testing.T) {
	// 'not' expects boolean
	expr, err := expressions.New("-true")
	assert.NoError(t, err)
	_, err = expr.Type(testScope, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-numeric type")
}

func TestDependencyResolution_Error_TestComparingScopes(t *testing.T) {
	// scopes cannot be compared
	expr, err := expressions.New("$ > $")
	assert.NoError(t, err)
	_, err = expr.Type(testScope, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible type")
}

func TestDependencyResolution_Error_TestAddingScopes(t *testing.T) {
	// scopes cannot be added
	expr, err := expressions.New("$ + $")
	assert.NoError(t, err)
	_, err = expr.Type(testScope, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible type")
}

func TestDependencyResolution_Error_TestAddingMaps(t *testing.T) {
	// maps cannot be added
	expr, err := expressions.New("$.faz + $.faz")
	assert.NoError(t, err)
	_, err = expr.Type(testScope, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible type")
}
func TestDependencyResolution_Error_TestAddingLists(t *testing.T) {
	// lists cannot be added
	expr, err := expressions.New("$.int_list + $.int_list")
	assert.NoError(t, err)
	_, err = expr.Type(testScope, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible type")
}
