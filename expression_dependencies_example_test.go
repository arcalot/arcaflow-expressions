package expressions_test

import (
	"fmt"

	"go.flow.arcalot.io/expressions"
	"go.flow.arcalot.io/pluginsdk/schema"
)

var myScope = schema.NewScopeSchema(
	schema.NewObjectSchema(
		"root",
		map[string]*schema.PropertySchema{
			"foo": schema.NewPropertySchema(
				schema.NewStringSchema(nil, nil, nil),
				nil,
				true,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
		},
	),
)

func ExampleExpression_Dependencies() {
	expr, err := expressions.New("$.foo")
	if err != nil {
		panic(err)
	}

	dependencyList, err := expr.Dependencies(
		myScope,
		nil,
		nil,
		false,
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v", dependencyList)
	// Output: [$.foo]
}
