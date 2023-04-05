package ast

// Walk walks the  in-order.
func Walk(
	ast Node,
	beforeNode func(node Node) error,
	onNode func(node Node) error,
	afterNode func(node Node) error,
) error {
	if err := beforeNode(ast); err != nil {
		return err
	}
	if left := ast.Left(); left != nil {
		if err := Walk(left, beforeNode, onNode, afterNode); err != nil {
			return err
		}
	}
	if err := onNode(ast); err != nil {
		return err
	}
	if right := ast.Right(); right != nil {
		if err := Walk(right, beforeNode, onNode, afterNode); err != nil {
			return err
		}
	}
	if err := afterNode(ast); err != nil {
		return err
	}
	return nil
}
