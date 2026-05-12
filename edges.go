package callgraph

import "golang.org/x/tools/go/ssa"

// EdgeInfo holds a caller-callee pair from the call graph.
type EdgeInfo struct {
	Caller *ssa.Function
	Callee *ssa.Function
}

// ForEachEdge iterates all call edges in the graph. The callback receives
// each caller-callee pair. Return false to stop iteration.
func (g *Graph) ForEachEdge(fn func(EdgeInfo) bool) {
	for _, mg := range g.modules {
		for caller, node := range mg.CG.Nodes {
			if caller == nil || node == nil {
				continue
			}
			for _, edge := range node.Out {
				callee := edge.Callee.Func
				if callee == nil {
					continue
				}
				if !fn(EdgeInfo{Caller: caller, Callee: callee}) {
					return
				}
			}
		}
	}
}

// ForEachEdgeInPackages iterates call edges where the caller belongs to one
// of the given package paths. This is the filtered variant for consumers that
// only want edges originating from their own code.
func (g *Graph) ForEachEdgeInPackages(pkgPaths map[string]bool, fn func(EdgeInfo) bool) {
	for _, mg := range g.modules {
		for caller, node := range mg.CG.Nodes {
			if caller == nil || node == nil {
				continue
			}
			if !pkgPaths[FuncPkgPath(caller)] {
				continue
			}
			for _, edge := range node.Out {
				callee := edge.Callee.Func
				if callee == nil {
					continue
				}
				if !fn(EdgeInfo{Caller: caller, Callee: callee}) {
					return
				}
			}
		}
	}
}
