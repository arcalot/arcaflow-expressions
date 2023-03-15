# Arcaflow Expressions Library

This library provides the ability to parse Arcaflow expressions.

## Installation

This library can be installed as a Go module dependency:

```
go get go.flow.arcalot.io/expressions
```

## Evaluating expressions

You can evaluate expressions against a data set consisting of primitive types (bool, int, float, string, map, slice) by
parsing the expression and then calling the `Evaluate()` function:

```go
package main

import (
    "fmt"
    "log"

    "go.flow.arcalot.io/expressions"
)

func main() {
    expr, err := expressions.New("$.foo.bar")
    if err != nil {
        log.Fatal(err)
    }
    result, err := expr.Evaluate(
        // Pass the data here:
        map[string]any{
            "foo": map[string]any{
                "bar": "hello world!",
            },
        },
        // Pass the workflow context (map of file names to content) here:
        nil,
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
}
```

## Building a dependency tree

Similarly, you can also evaluate an expression against a scope and get a list of dependencies an expression has:

```go
package main

import (
    "fmt"

    "go.flow.arcalot.io/expressions"
    "go.flow.arcalot.io/pluginsdk/schema"
)

var myScope = schema.NewScopeSchema(
    schema.NewObjectSchema(
        "root",
        map[string]*schema.PropertySchema{
            "foo": schema.NewPropertySchema(
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
)

func main() {
    expr, err := expressions.New("$.foo")
    if err != nil {
        panic(err)
    }

    dependencyList, err := expr.Dependencies(
        myScope,
        nil,
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("%v", dependencyList)
    // Output: [$.foo]
}
```

## Evaluating result types

You can also evaluate an expression and retrieve the result type. Note, that the result is not 100% guaranteed as the result may be optional and the value may not be available.

```go
package main

import (
    "fmt"

    "go.flow.arcalot.io/expressions"
    "go.flow.arcalot.io/pluginsdk/schema"
)

var scopeForType = schema.NewScopeSchema(
    schema.NewObjectSchema(
        "root",
        map[string]*schema.PropertySchema{
            "foo": schema.NewPropertySchema(
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
)

func main() {
    expr, err := expressions.New("$.foo")
    if err != nil {
        panic(err)
    }

    t, err := expr.Type(
        scopeForType,
        nil,
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("%v", t.TypeID())
    // Output: string
}
```
