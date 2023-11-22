package expressions_test

import (
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
)

var testData = map[string]struct {
	data           any
	expr           string
	parseError     bool
	evalError      bool
	expectedResult any
}{
	"root": {
		"Hello world!",
		"$",
		false,
		false,
		"Hello world!",
	},
	"sub1": {
		map[string]any{
			"message": "Hello world!",
		},
		"$.message",
		false,
		false,
		"Hello world!",
	},
	"sub1map": {
		map[string]any{
			"message": "Hello world!",
		},
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
		"$.container.message",
		false,
		false,
		"Hello world!",
	},
	"list": {
		[]string{
			"Hello world!",
		},
		"$[0]",
		false,
		false,
		"Hello world!",
	},
}

func TestEvaluate(t *testing.T) {
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
			result, err := expr.Evaluate(testCase.data, nil)
			if testCase.evalError && err == nil {
				t.Fatalf("No eval error returned for test %s", name)
			}
			if !testCase.evalError {
				if err != nil {
					t.Fatalf("Unexpected eval error returned for test %s (%v)", name, err)
				}
				assert.Equals(t, result, testCase.expectedResult)
			}
		})
	}
}
