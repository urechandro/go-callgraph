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

// Build a call graph for a module.
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

Use **RTA** (default) when accuracy matters. Use **CHA** for quick scans or
when the program has few interface-heavy dispatch sites.

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

### Symbol IDs

| Function | Description |
|----------|-------------|
| `QualifiedID(fn)` | `"pkg/path.Type.Method"` for an `*ssa.Function` |
| `FuncPkgPath(fn)` | Package import path of an `*ssa.Function` |
| `QualifiedObjID(pkgPath, obj)` | Stable ID from a `types.Object` (AST-level indexing) |

## License

MIT — see [LICENSE](LICENSE).
