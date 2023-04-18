package expressions_test

import (
	"testing"

	"go.flow.arcalot.io/expressions"
	"go.flow.arcalot.io/pluginsdk/schema"
)

func TestExpression_InferSourceType_root(t *testing.T) {
	expr, err := expressions.New("$")
	if err != nil {
		t.Fatalf("%v", err)
	}

	resultType, err := expr.InferSourceType(
		schema.NewStringSchema(nil, nil, nil),
		schema.NewObjectSchema(
			"root",
			map[string]*schema.PropertySchema{
				"input": schema.NewPropertySchema(
					schema.NewAnySchema(),
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
		nil,
	)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if typeID := resultType.TypeID(); typeID != schema.TypeIDObject {
		t.Fatalf("incorrect type found: %s", typeID)
	}
}

func TestExpression_InferSourceType_identifier(t *testing.T) {
	expr, err := expressions.New("$.input")
	if err != nil {
		t.Fatalf("%v", err)
	}

	resultType, err := expr.InferSourceType(
		schema.NewStringSchema(nil, nil, nil),
		schema.NewObjectSchema(
			"root",
			map[string]*schema.PropertySchema{
				"input": schema.NewPropertySchema(
					schema.NewAnySchema(),
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
		nil,
	)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if typeID := resultType.(schema.Object).Properties()["input"].TypeID(); typeID != schema.TypeIDString {
		t.Fatalf("incorrect type found: %s", typeID)
	}
}
