# go-callgraph

A Go library for building and querying SSA-based call graphs. Wraps
`golang.org/x/tools/go/callgraph` with a clean API for common use cases:
caller/callee lookups, test discovery, and edge iteration for persistence
layers.

## Installation

```sh
go get github.com/urechandro/go-callgraph
```

Requires Go 1.26 and a working Go toolchain in `PATH` (the SSA builder
shells out to `go list`).

## Quick start

```go
import callgraph "github.com/urechandro/go-callgraph"

// Build a call graph for a module (defaults to CHA when method is "").
g, err := callgraph.Build([]string{"/path/to/module"}, callgraph.RTA)
if err != nil {
    log.Fatal(err)
}

// Find all functions named "ProcessOrder".
refs := g.FindFunctions("ProcessOrder")

// Who calls ProcessOrder?
for _, c := range g.DirectCallers(refs) {
    fmt.Printf("%s  %s:%d\n", c.Name, c.File, c.Line)
}

// Which tests exercise ProcessOrder transitively?
for _, t := range g.CallersToTests(refs) {
    fmt.Println(t.Name)
}
```

## Algorithms

| Algorithm | Constant | Precision | Speed |
|-----------|----------|-----------|-------|
| Rapid Type Analysis | `callgraph.RTA` | High — tracks concrete types | Slower |
| Class Hierarchy Analysis | `callgraph.CHA` | Conservative — may over-approximate | Faster |

**CHA** is the default (`callgraph.DefaultMethod`). Pass `""` or omit an explicit
choice to get it. Use **RTA** when accuracy matters and you have a binary or
test suite as an entry point.

## Pre-loaded packages

If you already call `packages.Load` with your own config (custom exclusions,
`token.FileSet`, etc.), skip the load step:

```go
pkgs, err := packages.Load(myCfg, "./...")
g, err := callgraph.BuildFromPackages(pkgs, callgraph.RTA)
```

The `callgraph.LoadMode` constant exports the required `packages.Config.Mode`
flags so you can compose them with your own.

## Persisting edges

For indexers that store edges in a database:

```go
g.ForEachEdgeInPackages(myPkgs, func(e callgraph.EdgeInfo) bool {
    fromID := callgraph.QualifiedID(e.Caller) // "pkg/path.Type.Method"
    toID   := callgraph.QualifiedID(e.Callee)
    if fromID != "" && toID != "" {
        db.Insert(fromID, toID)
    }
    return true // return false to stop
})
```

`QualifiedObjID(pkgPath, obj)` is also exported for consumers that build IDs
from `types.Object` (e.g. during AST-level symbol indexing).

## Multi-module support

Pass multiple roots to `Build`; each is analysed independently and the results
are unified under a single `*Graph`:

```go
g, err := callgraph.Build([]string{
    "/repo/service-a",
    "/repo/service-b",
}, callgraph.RTA)
```

## Visualization

### DOT / Graphviz

Export the full graph to a `.dot` file and render it with Graphviz:

```go
g, _ := callgraph.Build([]string{"."}, callgraph.DefaultMethod)

f, _ := os.Create("callgraph.dot")
callgraph.WriteDOT(f, g)
f.Close()
```

```sh
dot -Tsvg callgraph.dot > callgraph.svg   # brew install graphviz
```

Output is deterministic — nodes are sorted alphabetically, duplicate edges
are suppressed.

### ASCII tree

Print direct callers or callees as a tree in the terminal:

```go
refs := g.FindFunctions("ProcessOrder")

callgraph.WriteTree(os.Stdout, "ProcessOrder", g.DirectCallers(refs))
// ProcessOrder
// ├── HandleCheckout  checkout.go:42
// ├── RetryJob        worker.go:18
// └── TestProcessOrder  order_test.go:7

callgraph.WriteTree(os.Stdout, "ProcessOrder", g.DirectCallees(refs))
```

Both functions accept any `io.Writer`, so you can write to a file or
`bytes.Buffer` the same way.

## API reference

See [pkg.go.dev/github.com/urechandro/go-callgraph](https://pkg.go.dev/github.com/urechandro/go-callgraph) for the full godoc.

### Core

| Function | Description |
|----------|-------------|
| `Build(roots, method)` | Load + build call graph for one or more module roots |
| `BuildFromPackages(pkgs, method)` | Build from pre-loaded packages |
| `LoadPackages(dir, patterns...)` | Load packages with the required mode flags |

### Query

| Method | Description |
|--------|-------------|
| `FindFunctions(name)` | Find all functions with an exact name match |
| `DirectCallers(refs)` | Functions that directly call the given functions |
| `DirectCallees(refs)` | Functions directly called by the given functions |
| `CallersToTests(refs)` | Transitive walk upward until Test/Benchmark functions are reached |
| `DirectCallerTests(refs)` | Tests that directly call the given functions (one level) |
| `FindTestsByName(substrings)` | Find tests whose name contains any of the given substrings |
| `TestsCovering(dirs, files, symbols)` | Tests in affected dirs reachable from changed symbols |
| `FuncInfos(refs)` | Resolve refs to name/file/line info |

### Edges

| Function | Description |
|----------|-------------|
| `ForEachEdge(fn)` | Iterate all call edges |
| `ForEachEdgeInPackages(pkgs, fn)` | Iterate edges where caller is in the given package set |

### Visualization

| Function | Description |
|----------|-------------|
| `WriteDOT(w, g)` | Write the full graph in Graphviz DOT format |
| `WriteTree(w, root, nodes)` | Print a root and its neighbors as an ASCII tree |

### Symbol IDs

| Function | Description |
|----------|-------------|
| `QualifiedID(fn)` | `"pkg/path.Type.Method"` for an `*ssa.Function` |
| `FuncPkgPath(fn)` | Package import path of an `*ssa.Function` |
| `QualifiedObjID(pkgPath, obj)` | Stable ID from a `types.Object` (AST-level indexing) |

## License

MIT — see [LICENSE](LICENSE).
