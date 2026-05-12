package callgraph

import (
	"strings"

	gocg "golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// buildRTA runs Rapid Type Analysis starting from reachable entry points.
// Roots are: init functions, main, and all Test/Benchmark functions.
// Always returns a non-nil graph (empty if no roots are found).
func buildRTA(prog *ssa.Program) *gocg.Graph {
	var roots []*ssa.Function
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Synthetic != "" {
			continue
		}
		// Skip uninstantiated generic templates — their bodies contain
		// *types.TypeParam which causes RTA to panic. With
		// ssa.InstantiateGenerics the builder creates concrete copies
		// for every call site; only those should be roots.
		if fn.TypeParams().Len() > 0 {
			continue
		}
		name := fn.Name()
		switch {
		case name == "init":
			roots = append(roots, fn)
		case name == "main" && fn.Package() != nil && fn.Package().Pkg.Name() == "main":
			roots = append(roots, fn)
		case strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark"):
			roots = append(roots, fn)
		}
	}

	if len(roots) == 0 {
		return &gocg.Graph{Nodes: make(map[*ssa.Function]*gocg.Node)}
	}

	res := rta.Analyze(roots, true)
	return res.CallGraph
}
