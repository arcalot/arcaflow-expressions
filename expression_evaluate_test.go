package expressions_test

import (
	"fmt"
	"go.flow.arcalot.io/pluginsdk/schema"
	"reflect"
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
)

var voidFunc, voidFuncErr = schema.NewCallableFunction(
	"test",
	make([]schema.Type, 0),
	nil,
	false,
	nil,
	func() {
	},
)
var strFunc, strFuncErr = schema.NewCallableFunction(
	"test",
	make([]schema.Type, 0),
	schema.NewStringSchema(nil, nil, nil),
	true,
	nil,
	func() (string, error) {
		return "test", nil
	},
)
var strToStrFunc, strToStrFuncErr = schema.NewCallableFunction(
	"test",
	[]schema.Type{schema.NewStringSchema(nil, nil, nil)},
	schema.NewStringSchema(nil, nil, nil),
	true,
	nil,
	func(a string) (string, error) {
		return a, nil
	},
)

var twoIntToIntFunc, twoIntToIntFuncErr = schema.NewCallableFunction(
	"multiply",
	[]schema.Type{
		schema.NewIntSchema(nil, nil, nil),
		schema.NewIntSchema(nil, nil, nil),
	},
	schema.NewIntSchema(nil, nil, nil),
	true,
	nil,
	func(a int64, b int64) (int64, error) {
		return a * b, nil
	},
)

var dynamicToListFunc, dynamicToListFuncErr = schema.NewDynamicCallableFunction(
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
		if len(inputType) == 1 {
			return schema.NewListSchema(inputType[0], nil, nil), nil
		} else {
			return nil, fmt.Errorf("incorrect param count")
		}
	},
)

var testData = map[string]struct {
	data           any
	functions      map[string]schema.CallableFunction
	expr           string
	parseError     bool
	evalError      bool
	expectedResult any
}{
	"root": {
		"Hello world!",
		nil,
		"$",
		false,
		false,
		"Hello world!",
	},
	"sub1": {
		map[string]any{
			"message": "Hello world!",
		},
		nil,
		"$.message",
		false,
		false,
		"Hello world!",
	},
	"sub1map": {
		map[string]any{
			"message": "Hello world!",
		},
		nil,
		`$["message"]`,
		false,
		false,
		"Hello world!",
	},
	"sub2": {
		map[string]any{
			"container": map[string]any{
				"message": "Hello world!",
			},
		},
		nil,
		"$.container.message",
		false,
		false,
		"Hello world!",
	},
	"list": {
		[]string{
			"Hello world!",
		},
		nil,
		"$[0]",
		false,
		false,
		"Hello world!",
	},
	"parameterless-void-func": {
		[]any{},
		map[string]schema.CallableFunction{
			"test": voidFunc,
		},
		"test()",
		false,
		false,
		nil,
	},
	"parameterless-str-func": {
		[]any{},
		map[string]schema.CallableFunction{
			"test": strFunc,
		},
		"test()",
		false,
		false,
		"test",
	},
	"single-literal-param-func": {
		[]any{},
		map[string]schema.CallableFunction{
			"test": strToStrFunc,
		},
		`test("a")`,
		false,
		false,
		"a",
	},
	"single-reference-param-func": {
		map[string]any{
			"message": "Hello world!",
		},
		map[string]schema.CallableFunction{
			"test": strToStrFunc,
		},
		`test($.message)`,
		false,
		false,
		"Hello world!",
	},
	"multi-param-func": {
		map[string]any{
			"val": int64(5),
		},
		map[string]schema.CallableFunction{
			"multiply": twoIntToIntFunc,
		},
		`multiply($.val, 5)`,
		false,
		false,
		int64(25),
	},
	"chained-functions": {
		map[string]any{
			"val": int64(5),
		},
		map[string]schema.CallableFunction{
			"multiply": twoIntToIntFunc,
		},
		`multiply($.val, multiply($.val, 2))`,
		false,
		false,
		int64(50),
	},
	"to-list-function-int": {
		map[string]any{
			"val": int64(5),
		},
		map[string]schema.CallableFunction{
			"toList": dynamicToListFunc,
		},
		`toList($.val)`,
		false,
		false,
		[]int64{5, 5},
	},
	"to-list-function-str": {
		map[string]any{
			"val": "test",
		},
		map[string]schema.CallableFunction{
			"toList": dynamicToListFunc,
		},
		`toList($.val)`,
		false,
		false,
		[]string{"test", "test"},
	},
	"simple-int-addition": {
		nil,
		nil,
		`5 + 5`,
		false,
		false,
		int64(10),
	},
	"simple-int-subtraction": {
		nil,
		nil,
		`5 - 1`,
		false,
		false,
		int64(4),
	},
	"simple-int-multiplication": {
		nil,
		nil,
		`2 * 2`,
		false,
		false,
		int64(4),
	},
	"simple-int-division": {
		nil,
		nil,
		`2 / 2`,
		false,
		false,
		int64(1),
	},
	"simple-int-mod": {
		nil,
		nil,
		`3 % 2`,
		false,
		false,
		int64(1),
	},
	"simple-int-power": {
		nil,
		nil,
		`2 ^ 3`,
		false,
		false,
		int64(8),
	},
	"simple-int-equals-same": {
		nil,
		nil,
		`1 == 1`,
		false,
		false,
		true,
	},
	"simple-int-equals-different": {
		nil,
		nil,
		`1 == 2`,
		false,
		false,
		false,
	},
	"simple-int-not-equals-same": {
		nil,
		nil,
		`1 != 1`,
		false,
		false,
		false,
	},
	"simple-int-not-equals-different": {
		nil,
		nil,
		`1 != 2`,
		false,
		false,
		true,
	},
	"simple-int-greater-than": {
		nil,
		nil,
		`1 > 1`,
		false,
		false,
		false,
	},
	"simple-int-less-than": {
		nil,
		nil,
		`1 < 1`,
		false,
		false,
		false,
	},
	"simple-int-greater-than-equals": {
		nil,
		nil,
		`1 >= 1`,
		false,
		false,
		true,
	},
	"simple-int-less-than-equals": {
		nil,
		nil,
		`1 <= 1`,
		false,
		false,
		true,
	},
	// Careful. Floating point numbers aren't exact.
	"simple-float-addition": {
		nil,
		nil,
		`5.0 + 5.0`,
		false,
		false,
		10.0,
	},
	"simple-float-subtraction": {
		nil,
		nil,
		`5.0 - 1.0`,
		false,
		false,
		4.0,
	},
	"simple-float-multiplication": {
		nil,
		nil,
		`2.0 * 2.0`,
		false,
		false,
		4.0,
	},
	"simple-float-division": {
		nil,
		nil,
		`2.0 / 2.0`,
		false,
		false,
		1.0,
	},
	// This case has a special case in the code.
	"simple-float-mod": {
		nil,
		nil,
		`3.0 % 2.0`,
		false,
		false,
		1.0,
	},
	"simple-float-power": {
		nil,
		nil,
		`2.0 ^ 3.0`,
		false,
		false,
		8.0,
	},
	"simple-float-equals-same": {
		nil,
		nil,
		`1.0 == 1.0`,
		false,
		false,
		true,
	},
	"simple-float-equals-different": {
		nil,
		nil,
		`1.0 == 2.0`,
		false,
		false,
		false,
	},
	"simple-float-not-equals-same": {
		nil,
		nil,
		`1.0 != 1.0`,
		false,
		false,
		false,
	},
	"simple-float-not-equals-different": {
		nil,
		nil,
		`1.0 != 2.0`,
		false,
		false,
		true,
	},
	"simple-float-greater-than": {
		nil,
		nil,
		`1.0 > 1.0`,
		false,
		false,
		false,
	},
	"simple-float-less-than": {
		nil,
		nil,
		`1.0 < 1.0`,
		false,
		false,
		false,
	},
	"simple-float-greater-than-equals": {
		nil,
		nil,
		`1.0 >= 1.0`,
		false,
		false,
		true,
	},
	"simple-float-less-than-equals": {
		nil,
		nil,
		`1.0 <= 1.0`,
		false,
		false,
		true,
	},
}

func TestEvaluate(t *testing.T) {
	assert.NoError(t, voidFuncErr)
	assert.NoError(t, strFuncErr)
	assert.NoError(t, strToStrFuncErr)
	assert.NoError(t, twoIntToIntFuncErr)
	assert.NoError(t, dynamicToListFuncErr)

	for name, tc := range testData {
		testCase := tc
		t.Run(name, func(t *testing.T) {
			expr, err := expressions.New(testCase.expr)
			if testCase.parseError && err == nil {
				t.Fatalf("No parse error returned for test %s", name)
			}
			if !testCase.parseError && err != nil {
				t.Fatalf("Unexpected parse error returned for test %s (%v)", name, err)
			}
			result, err := expr.Evaluate(testCase.data, tc.functions, nil)
			if testCase.evalError && err == nil {
				t.Fatalf("No eval error returned for test %s", name)
			}
			if !testCase.evalError {
				if err != nil {
					t.Fatalf("Unexpected eval error returned for test '%s' (%v)", name, err)
				}
				assert.Equals(t, result, testCase.expectedResult)
			}
		})
	}
}
