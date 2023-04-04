package expressions

import (
	"fmt"
	"strconv"

	"go.flow.arcalot.io/expressions/internal/ast"
	"go.flow.arcalot.io/pluginsdk/schema"
)

// dependencyContext holds the root data for a dependency evaluation in an expression. This is useful so that we
// don't need to pass the root type, path, and workflow context along with each function call.
type dependencyContext struct {
	rootType        schema.Scope
	rootPath        *PathTree
	workflowContext map[string][]byte
}

// dependencies evaluates an AST node for possible dependencies. It adds items to the specified path tree and returns
// it. You can use this to build a list of value paths that make up the dependencies of this expression. Furthermore,
// you can also use this function to evaluate the type the resolved expression's value will have.
func (d *dependencyContext) dependencies(
	node ast.Node,
	currentType schema.Type,
	path *PathTree,
) (schema.Type, *PathTree, error) {
	switch n := node.(type) {
	case *ast.DotNotation:
		// The dot notation is when item.item is encountered. We simply traverse the AST in order, left to right,
		// nothing specific to do.
		leftType, leftPath, err := d.dependencies(n.LeftAccessibleNode, currentType, path)
		if err != nil {
			return nil, nil, err
		}
		return d.dependencies(n.RightAccessIdentifier, leftType, leftPath)
	case *ast.MapAccessor:
		// A map accessor is when an item[item] is encountered. Here we need to evaluate the left tree as usual, then
		// use the right tree according to its type. This is either a literal (e.g. a string), or it is a subexpression.
		// Literals will call dependenciesMapKey, while subexpressions need to be evaluated on their own on the root
		// type.
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
				// For maps, we try to compare the type of the map key with the resulting type of the subexpression to
				// make sure that there are no runtime type failures. The user may need to add type conversion functions
				// to their expressions to convert an integer to a string, for example.
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
				// Lists have integer indexes, so we try to make sure that the subexpression is yielding an int or
				// int-like type. This will have the best chance of not resulting in a runtime error.

				list := leftType.(schema.UntypedList)
				switch keyType.TypeID() {
				case schema.TypeIDInt:
				default:
					return nil, nil, fmt.Errorf("subexpressions resulted in a %s type for a list key, integer expected", keyType.TypeID())
				}
				pathItem := &PathTree{
					PathItem: list,
					Subtrees: nil,
				}
				path.Subtrees = append(path.Subtrees, pathItem)
				return list.Items(), pathItem, nil
			default:
				// We don't support subexpressions to pick a property on an object type since that would result in
				// unpredictable behavior and runtime errors. Furthermore, we would not be able to perform type
				// evaluation.
				return nil, nil, fmt.Errorf("subexpressions are only supported on map and list types, %s given", currentType.TypeID())
			}

		default:
			return nil, nil, fmt.Errorf("bug: neither literal, nor subexpression are set on key")
		}
	case *ast.Key:
		// Keys should only be found in map accessors, which is already handled above, so this should never happen.
		return nil, nil, fmt.Errorf("bug: reached key outside a map accessor")
	case *ast.Identifier:
		switch n.IdentifierName {
		case "$":
			// This identifier means the root of the expression.
			return d.rootType, path, nil
		default:
			// This case is the item.item type expression, where the right item is the "identifier" in question.
			return d.dependenciesMapKey(currentType, n.IdentifierName, path)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported AST node type: %T", n)
	}
}

// dependenciesMapKey is a helper function that extracts an item in a list, map, or object. This is used when an
// identifier or a map accessor are encountered.
func (d *dependencyContext) dependenciesMapKey(currentType schema.Type, key any, path *PathTree) (schema.Type, *PathTree, error) {
	switch currentType.TypeID() {
	case schema.TypeIDList:
		// Lists can only have numeric indexes, therefore we need to convert the types to integers. Since internally
		// the SDK doesn't use anything but ints, that's what we are converting to.
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
		// Maps can have various key types, so we need to unserialize the passed key according to its schema and use
		// it to find the correct key.
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
		// Object-likes always have field names (strings) as keys, so we need to convert the passed value to a string.
		// 99% of the time these are going to be strings anyway.
		var objectItem string
		switch k := key.(type) {
		case string:
			objectItem = k
		case int:
			objectItem = fmt.Sprintf("%d", k)
		default:
			return nil, nil, fmt.Errorf("bug: invalid key type encountered for object key: %T", key)
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