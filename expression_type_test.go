package expressions_test

import (
    "testing"

    "go.arcalot.io/assert"
    "go.flow.arcalot.io/expressions"
    "go.flow.arcalot.io/pluginsdk/schema"
)

func TestTypeEvaluation(t *testing.T) {
    t.Run("object", func(t *testing.T) {
        expr, err := expressions.New("$.foo.bar")
        assert.NoError(t, err)
        resultType, err := expr.Type(testScope, nil)
        assert.NoError(t, err)
        assert.Equals(t, resultType.TypeID(), schema.TypeIDString)
    })

    t.Run("map-accessor", func(t *testing.T) {
        expr, err := expressions.New("$[\"foo\"].bar")
        assert.NoError(t, err)
        resultType, err := expr.Type(testScope, nil)
        assert.NoError(t, err)
        assert.Equals(t, resultType.TypeID(), schema.TypeIDString)
    })

    t.Run("map", func(t *testing.T) {
        expr, err := expressions.New("$.faz")
        assert.NoError(t, err)
        resultType, err := expr.Type(testScope, nil)
        assert.NoError(t, err)
        assert.Equals(t, resultType.TypeID(), schema.TypeIDMap)
    })

    t.Run("map-subkey", func(t *testing.T) {
        expr, err := expressions.New("$.faz.foo")
        assert.NoError(t, err)
        resultType, err := expr.Type(testScope, nil)
        assert.NoError(t, err)
        assert.Equals(t, resultType.TypeID(), schema.TypeIDObject)
    })

    t.Run("subexpression-invalid", func(t *testing.T) {
        expr, err := expressions.New("$.foo[($.faz.foo)]")
        assert.NoError(t, err)
        _, err = expr.Type(testScope, nil)
        assert.Error(t, err)
    })

    t.Run("subexpression", func(t *testing.T) {
        expr, err := expressions.New("$.faz[($.foo.bar)]")
        assert.NoError(t, err)
        resultType, err := expr.Type(testScope, nil)
        assert.NoError(t, err)
        assert.Equals(t, resultType.TypeID(), schema.TypeIDObject)

    })
}
