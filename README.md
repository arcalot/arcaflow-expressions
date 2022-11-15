# Arcaflow Expressions Library

This library holds a library for parsing expressions. The expression language is based on jsonpath, but may change to support more than jsonpath supports. It is not turing complete, since it is intended for accessing values in a workflow.

The core components are the:
- tokenizer: Splits the input into tokens
- recursive_descent_parser: Interprets the tokens to build an abstract syntax tree
- ast: Holds the components that can be a part of the abstract syntax tree.

To use, call `InitParser` in `recursive_descent_parser.go` with the expression and filename, then call `ParseExpression()`
If it's a valid expression, you will get an abstract syntax tree, and you will be able to traverse that like a normal tree. To make per-node-type decisions, do a type check on the node.

