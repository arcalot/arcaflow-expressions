package expressions_test

import "go.flow.arcalot.io/pluginsdk/schema"

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
		},
	),
)
