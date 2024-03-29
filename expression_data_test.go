package expressions_test

import (
	"go.flow.arcalot.io/pluginsdk/schema"
	"regexp"
)

var testScope = schema.NewScopeSchema(
	schema.NewObjectSchema(
		"root",
		map[string]*schema.PropertySchema{
			"foo": schema.NewPropertySchema(
				schema.NewObjectSchema(
					"foo",
					map[string]*schema.PropertySchema{
						"bar": schema.NewPropertySchema(
							schema.NewStringSchema(nil, nil, nil),
							nil,
							true,
							nil,
							nil,
							nil,
							nil,
							nil,
						),
						"int_list": schema.NewPropertySchema(
							schema.NewListSchema(
								schema.NewIntSchema(nil, nil, nil),
								nil,
								nil,
							),
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
				true,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
			"faz": schema.NewPropertySchema(
				schema.NewMapSchema(
					schema.NewStringSchema(nil, nil, nil),
					schema.NewObjectSchema(
						"foo",
						map[string]*schema.PropertySchema{},
					),
					nil, nil,
				),
				nil,
				true,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
			"simple_str": schema.NewPropertySchema(
				schema.NewStringSchema(nil, nil, nil),
				nil,
				true,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
			"restrictive_str": schema.NewPropertySchema(
				schema.NewStringSchema(nil, nil, regexp.MustCompile(`^a$`)),
				nil,
				true,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
			"simple_int": schema.NewPropertySchema(
				schema.NewIntSchema(nil, nil, nil),
				nil,
				true,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
			"simple_int_2": schema.NewPropertySchema(
				schema.NewIntSchema(nil, nil, nil),
				nil,
				true,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
			"simple_any": schema.NewPropertySchema(
				schema.NewAnySchema(),
				nil,
				true,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
			"simple_bool": schema.NewPropertySchema(
				schema.NewBoolSchema(),
				nil,
				true,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
			"int_list": schema.NewPropertySchema(
				schema.NewListSchema(
					schema.NewIntSchema(nil, nil, nil),
					nil,
					nil,
				),
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
