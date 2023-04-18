package expressions_test

import (
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
	"go.flow.arcalot.io/pluginsdk/schema"
)

func TestDependencyResolution(t *testing.T) {
	scopes := map[string]schema.Type{
		"scope": testScope,
		"any":   schema.NewAnySchema(),
	}
	for name, schemaType := range scopes {
		name := name
		schemaType := schemaType
		t.Run(name, func(t *testing.T) {
			t.Run("object", func(t *testing.T) {
				expr, err := expressions.New("$.foo.bar")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.foo.bar")
			})

			t.Run("map-accessor", func(t *testing.T) {
				expr, err := expressions.New("$[\"foo\"].bar")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.foo.bar")
			})

			t.Run("map", func(t *testing.T) {
				expr, err := expressions.New("$.faz")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.faz")
			})

			t.Run("map-subkey", func(t *testing.T) {
				expr, err := expressions.New("$.faz.foo")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil)
				assert.NoError(t, err)
				assert.Equals(t, len(path), 1)
				assert.Equals(t, path[0].String(), "$.faz.foo")
			})
			t.Run("subexpression-invalid", func(t *testing.T) {
				expr, err := expressions.New("$.foo[($.faz.foo)]")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil)
				if name == "scope" {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equals(t, path[0].String(), "$.foo")
					assert.Equals(t, path[1].String(), "$.faz.foo")
				}
			})

			t.Run("subexpression", func(t *testing.T) {
				expr, err := expressions.New("$.faz[($.foo.bar)]")
				assert.NoError(t, err)
				path, err := expr.Dependencies(schemaType, nil)
				if name == "scope" {
					assert.NoError(t, err)
					assert.Equals(t, len(path), 2)
					assert.Equals(t, path[0].String(), "$.faz.*")
					assert.Equals(t, path[1].String(), "$.foo.bar")
				} else {
					assert.NoError(t, err)
					assert.Equals(t, len(path), 2)
					assert.Equals(t, path[0].String(), "$.faz")
					assert.Equals(t, path[1].String(), "$.foo.bar")
				}
			})
		})
	}

}
