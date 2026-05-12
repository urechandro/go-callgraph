// Package callgraph builds SSA-based call graphs from Go source code and
// provides query methods for caller/callee lookups, test discovery, and edge
// iteration.
//
// # Algorithms
//
// Two algorithms are supported:
//
//   - [CHA] (Class Hierarchy Analysis) — conservative. Includes all possible
//     dispatch targets for interface calls. Fast, may over-approximate.
//   - [RTA] (Rapid Type Analysis) — precise. Tracks which concrete types are
//     actually instantiated and reachable. Slower, fewer false edges.
//
// RTA is the default for most use cases. Use CHA when you need faster builds
// and can tolerate some extra edges.
//
// # Quick start
//
//	g, err := callgraph.Build([]string{"/path/to/module"}, callgraph.RTA)
//	if err != nil { ... }
//
//	refs := g.FindFunctions("MyFunc")
//	callers := g.DirectCallers(refs)
//	for _, c := range callers {
//	    fmt.Println(c.Name, c.File, c.Line)
//	}
//
// # Integration with persistence layers
//
// If you already load packages yourself (e.g. with custom config or exclusion
// filters), use [BuildFromPackages] to skip the load step. Combine with
// [Graph.ForEachEdgeInPackages] and [QualifiedID] to persist edges to a store:
//
//	g, _ := callgraph.BuildFromPackages(pkgs, callgraph.RTA)
//	g.ForEachEdgeInPackages(myPkgSet, func(e callgraph.EdgeInfo) bool {
//	    store.Insert(callgraph.QualifiedID(e.Caller), callgraph.QualifiedID(e.Callee))
//	    return true
//	})
package callgraph

import (
	"fmt"

	gocg "golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Method selects the call graph algorithm.
type Method string

const (
	// CHA selects Class Hierarchy Analysis: conservative, fast, may over-approximate interface dispatch.
	CHA Method = "cha"
	// RTA selects Rapid Type Analysis: precise, tracks concrete types, recommended for most use cases.
	RTA Method = "rta"
)

// Graph wraps one or more call graphs (one per module) and provides
// query methods for test discovery and caller/callee lookups.
type Graph struct {
	method  Method
	modules []*ModuleGraph
}

// ModuleGraph holds the call graph and SSA program for a single module.
// Fields are exported so consumers can access the underlying data for
// custom traversal beyond what the query methods provide.
type ModuleGraph struct {
	CG       *gocg.Graph
	Prog     *ssa.Program
	ByFile   map[string][]*ssa.Function // source file → SSA functions defined there
	UserPkgs map[string]bool            // package paths explicitly loaded (excludes stdlib and indirect deps)
}

// FuncInfo describes a function found in the call graph.
type FuncInfo struct {
	Name string
	File string
	Line int
}

// FuncRef is a reference to a function in the graph. Pass to DirectCallers,
// DirectCallees, CallersToTests, etc.
type FuncRef struct {
	Fn   *ssa.Function
	Node *gocg.Node
	MG   *ModuleGraph
}

// Build loads packages from each module root, builds SSA, and runs the
// selected call graph algorithm. Each root is analysed independently.
func Build(modRoots []string, method Method) (*Graph, error) {
	g := &Graph{method: method}
	for _, root := range modRoots {
		pkgs, err := LoadPackages(root)
		if err != nil {
			fmt.Printf("warning: load packages for %s: %v\n", root, err)
			continue
		}

		// Log package-level errors but don't fail — partial programs are OK.
		for _, pkg := range pkgs {
			for _, e := range pkg.Errors {
				fmt.Printf("warning: %s: %v\n", pkg.PkgPath, e)
			}
		}

		mg, err := buildModuleGraph(pkgs, method)
		if err != nil {
			fmt.Printf("warning: call graph failed for %s: %v\n", root, err)
			continue
		}
		g.modules = append(g.modules, mg)
	}
	return g, nil
}

// BuildFromPackages builds SSA and a call graph from pre-loaded packages.
// Use this when you have already called packages.Load yourself (e.g. with a
// custom Config). Returns a single-module Graph.
func BuildFromPackages(pkgs []*packages.Package, method Method) (*Graph, error) {
	mg, err := buildModuleGraph(pkgs, method)
	if err != nil {
		return nil, err
	}
	return &Graph{
		method:  method,
		modules: []*ModuleGraph{mg},
	}, nil
}

// buildModuleGraph is the shared core: SSA build → algorithm selection → byFile index.
func buildModuleGraph(pkgs []*packages.Package, method Method) (*ModuleGraph, error) {
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()

	var cg *gocg.Graph
	switch method {
	case RTA:
		cg = buildRTA(prog)
	default:
		cg = cha.CallGraph(prog)
	}

	// Collect the package paths that were explicitly loaded (not indirect deps).
	userPkgs := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		if p.PkgPath != "" {
			userPkgs[p.PkgPath] = true
		}
	}

	// Build file → functions index for seed lookup.
	byFile := make(map[string][]*ssa.Function)
	for fn := range cg.Nodes {
		if fn == nil {
			continue
		}
		pos := prog.Fset.Position(fn.Pos())
		if !pos.IsValid() {
			continue
		}
		byFile[pos.Filename] = append(byFile[pos.Filename], fn)
	}

	return &ModuleGraph{
		CG:       cg,
		Prog:     prog,
		ByFile:   byFile,
		UserPkgs: userPkgs,
	}, nil
}

// Method returns the algorithm used to build this graph.
func (g *Graph) Method() Method { return g.method }

// Modules returns the underlying module graphs for direct access.
func (g *Graph) Modules() []*ModuleGraph { return g.modules }

// SymbolCount returns the total number of user-defined (non-synthetic)
// functions with valid source positions in the graph.
func (g *Graph) SymbolCount() int {
	n := 0
	for _, mg := range g.modules {
		for fn := range mg.CG.Nodes {
			if fn != nil && fn.Synthetic == "" {
				pos := mg.Prog.Fset.Position(fn.Pos())
				if pos.IsValid() {
					n++
				}
			}
		}
	}
	return n
}
