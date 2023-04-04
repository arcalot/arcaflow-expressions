package expressions_test

import (
	"testing"

	"go.arcalot.io/assert"
	"go.flow.arcalot.io/expressions"
)

func TestDependencyResolution(t *testing.T) {
	t.Run("object", func(t *testing.T) {
		expr, err := expressions.New("$.foo.bar")
		assert.NoError(t, err)
		path, err := expr.Dependencies(testScope, nil)
		assert.NoError(t, err)
		assert.Equals(t, len(path), 1)
		assert.Equals(t, path[0].String(), "$.foo.bar")
	})

	t.Run("map-accessor", func(t *testing.T) {
		expr, err := expressions.New("$[\"foo\"].bar")
		assert.NoError(t, err)
		path, err := expr.Dependencies(testScope, nil)
		assert.NoError(t, err)
		assert.Equals(t, len(path), 1)
		assert.Equals(t, path[0].String(), "$.foo.bar")
	})

	t.Run("map", func(t *testing.T) {
		expr, err := expressions.New("$.faz")
		assert.NoError(t, err)
		path, err := expr.Dependencies(testScope, nil)
		assert.NoError(t, err)
		assert.Equals(t, len(path), 1)
		assert.Equals(t, path[0].String(), "$.faz")
	})

	t.Run("map-subkey", func(t *testing.T) {
		expr, err := expressions.New("$.faz.foo")
		assert.NoError(t, err)
		path, err := expr.Dependencies(testScope, nil)
		assert.NoError(t, err)
		assert.Equals(t, len(path), 1)
		assert.Equals(t, path[0].String(), "$.faz.foo")
	})

	t.Run("subexpression-invalid", func(t *testing.T) {
		expr, err := expressions.New("$.foo[($.faz.foo)]")
		assert.NoError(t, err)
		_, err = expr.Dependencies(testScope, nil)
		assert.Error(t, err)
	})

	t.Run("subexpression", func(t *testing.T) {
		expr, err := expressions.New("$.faz[($.foo.bar)]")
		assert.NoError(t, err)
		path, err := expr.Dependencies(testScope, nil)
		assert.NoError(t, err)
		assert.Equals(t, len(path), 2)
		assert.Equals(t, path[0].String(), "$.faz.*")
		assert.Equals(t, path[1].String(), "$.foo.bar")
	})
}
