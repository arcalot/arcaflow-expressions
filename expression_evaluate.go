package expressions

import (
    "fmt"
    "reflect"

    "go.flow.arcalot.io/expressions/internal/ast"
)

// evaluate evaluates the passed AST node on a set of data consisting of primitive types. It must also have access
// to the root data to evaluate subexpressions, as well as the workflow context to pull in additional files. It will
// return the evaluated data.
func evaluate(node ast.ASTNode, data any, rootData any, workflowContext map[string][]byte) (any, error) {
    switch n := node.(type) {
    case *ast.DotNotation:
        // The dot notation is an item.item expression part, where we simply need to evaluate the left and right
        // subtrees in order.
        leftResult, err := evaluate(n.LeftAccessableNode, data, rootData, workflowContext)
        if err != nil {
            return nil, err
        }
        return evaluate(n.RightAccessIdentifier, leftResult, rootData, workflowContext)
    case *ast.MapAccessor:
        // The map accessor is an item[item] expression part, where we evaluate the left subtree first, then the right
        // subtree to obtain the map key. Finally, the map key is used to look up the resulting data.
        leftResult, err := evaluate(n.LeftNode, data, rootData, workflowContext)
        if err != nil {
            return nil, err
        }
        mapKey, err := evaluate(n.RightKey, leftResult, rootData, workflowContext)
        if err != nil {
            return nil, err
        }
        return evaluateMapKey(data, mapKey)
    case *ast.Key:
        // A key is an item in a map accessor (item[item]), which can either be a literal (e.g. string) or a
        // subexpression, needing to be evaluated in its own right.
        switch {
        case n.Literal != nil:
            return n.Literal.Value(), nil
        case n.SubExpression != nil:
            return evaluate(n.SubExpression, data, rootData, workflowContext)
        default:
            return nil, fmt.Errorf("bug: neither literal, nor subexpression are set on key")
        }
    case *ast.Identifier:
        // Identifiers are items in the dot notation.
        switch n.IdentifierName {
        case "$":
            // $ is the root node of the data structure.
            return rootData, nil
        default:
            // Otherwise, it's a normal accessor key, which we evaluate like a map key.
            return evaluateMapKey(data, n.IdentifierName)
        }
    default:
        return nil, fmt.Errorf("unsupported AST node type: %T", n)
    }
}

// evaluateMapKey is a helper function for evaluate that extracts an item in maps, lists, or object-likes when an
// identifier or map accessor is encountered.
func evaluateMapKey(data any, mapKey any) (any, error) {
    dataVal := reflect.ValueOf(data)
    switch dataVal.Kind() {
    case reflect.Map:
        // In case of a map, we simply look up the value passed.
        indexValue := dataVal.MapIndex(reflect.ValueOf(mapKey))
        if !indexValue.IsValid() {
            return nil, fmt.Errorf("map key %v not found", mapKey)
        }
        return indexValue.Interface(), nil
    case reflect.Slice:
        // In case of slices we want integers. The user is responsible for converting the type to an integer themselves.
        var sliceIndex int
        switch t := mapKey.(type) {
        case int:
            sliceIndex = t
        default:
            return nil, fmt.Errorf("unsupported map key type: %T", mapKey)
        }
        sliceLen := dataVal.Len()
        if sliceLen <= sliceIndex {
            return nil, fmt.Errorf("index %d is larger than the list items (%d)", sliceIndex, sliceLen)
        }
        indexValue := dataVal.Index(sliceIndex)
        return indexValue.Interface(), nil
    default:
        return nil, fmt.Errorf(
            "cannot evaluate identifier %v on a %s",
            mapKey,
            dataVal.Kind().String(),
        )
    }
}
