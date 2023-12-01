package expressions_test

import (
	"fmt"

	"go.flow.arcalot.io/expressions"
	"go.flow.arcalot.io/pluginsdk/schema"
)

var scopeForType = schema.NewScopeSchema(
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

func ExampleExpression_Type() {
	expr, err := expressions.New("$.foo")
	if err != nil {
		panic(err)
	}

	t, err := expr.Type(
		scopeForType,
		nil,
		nil,
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v", t.TypeID())
	// Output: string
}
