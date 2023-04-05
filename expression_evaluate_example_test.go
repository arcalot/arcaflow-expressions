package expressions_test

import (
	"fmt"

	"go.flow.arcalot.io/expressions"
)

func ExampleExpression_Evaluate() {
	expr, err := expressions.New("$.foo.bar")
	if err != nil {
		panic(err)
	}

	data, err := expr.Evaluate(
		map[string]any{
			"foo": map[string]any{
				"bar": 42,
			},
		},
		nil,
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v", data)
	// Output: 42
}
