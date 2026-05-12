# Contributing

Contributions are welcome. Please open an issue before sending a large PR so
we can agree on the direction.

## Prerequisites

- Go 1.23 or later
- A working Go toolchain in `PATH` (the SSA builder shells out to `go list`)

## Running tests

```sh
go test -race ./...
```

The test suite builds a real call graph from the `testdata/simple` fixture
module — no mocks. Tests take a few seconds due to SSA construction.

## Project structure

| File | Responsibility |
|------|---------------|
| `graph.go` | Core types (`Graph`, `ModuleGraph`, `FuncInfo`, `FuncRef`), `Build`, `BuildFromPackages` |
| `load.go` | `LoadPackages`, `LoadMode` constant |
| `rta.go` | RTA root selection (`init`, `main`, `Test*`, `Benchmark*`) |
| `query.go` | Graph query methods: callers, callees, test discovery |
| `edges.go` | Edge iteration for persistence consumers |
| `id.go` | Stable symbol ID helpers (`QualifiedID`, `QualifiedObjID`) |

## Algorithms background

This library uses two algorithms from `golang.org/x/tools/go/callgraph`:

**CHA (Class Hierarchy Analysis)** — for each interface call site, assumes
any type in the program that implements the interface could be the receiver.
Fast, but can generate spurious edges.

**RTA (Rapid Type Analysis)** — tracks which concrete types are actually
instantiated and reachable from the program's roots (`init`, `main`,
`Test*`). Fewer false edges than CHA, but requires explicit root selection
and a full program build.

RTA requires listing root functions explicitly. The `buildRTA` function in
`rta.go` selects roots by name pattern. If you add new root patterns (e.g.
`Fuzz*` functions), update that function.

## Sending a PR

1. Fork and create a feature branch.
2. Make your changes with tests.
3. Run `go vet ./...` and `go test -race ./...` — both must pass.
4. Open a PR against `main` with a clear description of what changed and why.
