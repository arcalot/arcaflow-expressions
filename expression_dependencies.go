package expressions

import (
    "fmt"
    "strconv"

    "go.flow.arcalot.io/expressions/internal/ast"
    "go.flow.arcalot.io/pluginsdk/schema"
)

type dependencyContext struct {
    rootType        schema.Scope
    rootPath        *PathTree
    workflowContext map[string][]byte
}

func (d *dependencyContext) dependencies(
    node ast.ASTNode,
    currentType schema.Type,
    path *PathTree,
) (schema.Type, *PathTree, error) {
    switch n := node.(type) {
    case *ast.DotNotation:
        leftType, leftPath, err := d.dependencies(n.LeftAccessableNode, currentType, path)
        if err != nil {
            return nil, nil, err
        }
        return d.dependencies(n.RightAccessIdentifier, leftType, leftPath)
    case *ast.MapAccessor:
        leftType, leftPath, err := d.dependencies(n.LeftNode, currentType, path)
        if err != nil {
            return nil, nil, err
        }

        switch {
        case n.RightKey.Literal != nil:
            return d.dependenciesMapKey(leftType, n.RightKey.Literal.Value(), leftPath)
        case n.RightKey.SubExpression != nil:
            // If we have a subexpression, we need to add all possible keys to the dependency map since we can't
            // determine the correct one to extract. This could be further refined by evaluating the type. If it is an
            // enum, we could potentially limit the number of dependencies.

            // Evaluate the subexpression
            keyType, _, err := d.dependencies(n.RightKey.SubExpression, d.rootType, d.rootPath)
            if err != nil {
                return nil, nil, err
            }
            switch leftType.TypeID() {
            case schema.TypeIDMap:
                mapType := leftType.(schema.UntypedMap)
                if keyType.TypeID() != mapType.Keys().TypeID() {
                    return nil, nil, fmt.Errorf("subexpressions resulted in a %s type for a map, %s expected", keyType.TypeID(), mapType.Keys().TypeID())
                }
                pathItem := &PathTree{
                    PathItem: "*",
                    Subtrees: nil,
                }
                leftPath.Subtrees = append(leftPath.Subtrees, pathItem)
                return mapType.Values(), pathItem, nil
            case schema.TypeIDList:
                list := leftType.(*schema.ListSchema)
                if keyType.TypeID() != schema.TypeIDInt && keyType.TypeID() != schema.TypeIDIntEnum {
                    return nil, nil, fmt.Errorf("subexpressions resulted in a %s type for a list key, integer expected", keyType.TypeID())
                }
                pathItem := &PathTree{
                    PathItem: list,
                    Subtrees: nil,
                }
                path.Subtrees = append(path.Subtrees, pathItem)
                return list.Items(), pathItem, nil
            default:
                return nil, nil, fmt.Errorf("subexpressions are only supported on map and list types, %s given", currentType.TypeID())
            }

        default:
            return nil, nil, fmt.Errorf("bug: neither literal, nor subexpression are set on key")
        }
    case *ast.Key:
        return nil, nil, fmt.Errorf("bug: reached key outside a map accessor")
    case *ast.Identifier:
        switch n.IdentifierName {
        case "$":
            return d.rootType, path, nil
        default:
            return d.dependenciesMapKey(currentType, n.IdentifierName, path)
        }
    default:
        return nil, nil, fmt.Errorf("unsupported AST node type: %T", n)
    }
}

func (d *dependencyContext) dependenciesMapKey(currentType schema.Type, key any, path *PathTree) (schema.Type, *PathTree, error) {
    switch currentType.TypeID() {
    case schema.TypeIDList:
        var listItem any
        var err error
        switch k := key.(type) {
        case string:
            listItem, err = strconv.ParseInt(k, 10, 64)
            if err != nil {
                return nil, nil, fmt.Errorf("cannot use non-integer expression identifier %s on list", key)
            }
        case int:
            listItem = k
        default:
            return nil, nil, fmt.Errorf("bug: invalid key type encountered for map key: %T", key)
        }
        pathItem := &PathTree{
            PathItem: listItem,
            Subtrees: nil,
        }
        path.Subtrees = append(path.Subtrees, pathItem)
        return currentType.(*schema.ListSchema).ItemsValue, pathItem, nil
    case schema.TypeIDMap:
        pathItem := &PathTree{
            PathItem: key,
            Subtrees: nil,
        }
        mapType := currentType.(schema.UntypedMap)
        if _, err := mapType.Keys().Unserialize(key); err != nil {
            return nil, nil, fmt.Errorf("cannot unserialize map key type %v (%w)", key, err)
        }
        path.Subtrees = append(path.Subtrees, pathItem)
        return mapType.Values(), pathItem, nil
    case schema.TypeIDObject:
        fallthrough
    case schema.TypeIDRef:
        fallthrough
    case schema.TypeIDScope:
        var objectItem string
        switch k := key.(type) {
        case string:
            objectItem = k
        case int:
            objectItem = fmt.Sprintf("%d", k)
        default:
            return nil, nil, fmt.Errorf("bug: invalid key type encountered for map key: %T", key)
        }

        currentObject := currentType.(schema.Object)
        properties := currentObject.Properties()
        property, ok := properties[objectItem]
        if !ok {
            return nil, nil, fmt.Errorf("object %s does not have a property named %s", currentObject.ID(), objectItem)
        }
        pathItem := &PathTree{
            PathItem: key,
            Subtrees: nil,
        }
        path.Subtrees = append(path.Subtrees, pathItem)
        return property.Type(), pathItem, nil
    default:
        return nil, nil, fmt.Errorf("cannot evaluate expression identifier %s on data type %s", key, currentType.TypeID())
    }
}
