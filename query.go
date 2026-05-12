package callgraph

import (
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// FindFunctions returns references to SSA functions whose name matches symbol.
func (g *Graph) FindFunctions(symbol string) []FuncRef {
	var refs []FuncRef
	for _, mg := range g.modules {
		for fn, node := range mg.CG.Nodes {
			if fn == nil || fn.Name() != symbol {
				continue
			}
			refs = append(refs, FuncRef{Fn: fn, Node: node, MG: mg})
		}
	}
	return refs
}

// FuncInfos returns position info for the given function references.
func (g *Graph) FuncInfos(refs []FuncRef) []FuncInfo {
	var results []FuncInfo
	for _, ref := range refs {
		pos := ref.MG.Prog.Fset.Position(ref.Fn.Pos())
		if pos.IsValid() {
			results = append(results, FuncInfo{
				Name: ref.Fn.Name(),
				File: pos.Filename,
				Line: pos.Line,
			})
		}
	}
	return results
}

// DirectCallers returns functions that directly call any of the given functions.
func (g *Graph) DirectCallers(refs []FuncRef) []FuncInfo {
	var results []FuncInfo
	for _, ref := range refs {
		if ref.Node == nil {
			continue
		}
		for _, edge := range ref.Node.In {
			caller := edge.Caller.Func
			if caller == nil || caller.Synthetic != "" {
				continue
			}
			pos := ref.MG.Prog.Fset.Position(caller.Pos())
			if !pos.IsValid() {
				continue
			}
			results = append(results, FuncInfo{
				Name: caller.Name(),
				File: pos.Filename,
				Line: pos.Line,
			})
		}
	}
	return results
}

// DirectCallees returns functions that any of the given functions directly call.
func (g *Graph) DirectCallees(refs []FuncRef) []FuncInfo {
	var results []FuncInfo
	for _, ref := range refs {
		if ref.Node == nil {
			continue
		}
		for _, edge := range ref.Node.Out {
			callee := edge.Callee.Func
			if callee == nil || callee.Synthetic != "" {
				continue
			}
			pos := ref.MG.Prog.Fset.Position(callee.Pos())
			if !pos.IsValid() {
				continue
			}
			results = append(results, FuncInfo{
				Name: callee.Name(),
				File: pos.Filename,
				Line: pos.Line,
			})
		}
	}
	return results
}

// CallersToTests walks callers upward from the given functions through the
// call graph and returns all Test/Benchmark functions reached.
func (g *Graph) CallersToTests(refs []FuncRef) []FuncInfo {
	var results []FuncInfo

	for _, ref := range refs {
		visited := map[*ssa.Function]bool{ref.Fn: true}
		queue := []*ssa.Function{ref.Fn}

		for len(queue) > 0 {
			fn := queue[0]
			queue = queue[1:]

			name := fn.Name()
			if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") {
				pos := ref.MG.Prog.Fset.Position(fn.Pos())
				if pos.IsValid() {
					results = append(results, FuncInfo{
						Name: name,
						File: pos.Filename,
						Line: pos.Line,
					})
				}
				continue
			}

			node := ref.MG.CG.Nodes[fn]
			if node == nil {
				continue
			}
			for _, edge := range node.In {
				caller := edge.Caller.Func
				if caller != nil && !visited[caller] {
					visited[caller] = true
					queue = append(queue, caller)
				}
			}
		}
	}

	return results
}

// DirectCallerTests returns Test/Benchmark functions that directly call any
// of the given functions (one level up only, no transitive walk).
func (g *Graph) DirectCallerTests(refs []FuncRef) []FuncInfo {
	var results []FuncInfo
	for _, ref := range refs {
		if ref.Node == nil {
			continue
		}
		for _, edge := range ref.Node.In {
			caller := edge.Caller.Func
			if caller == nil || caller.Synthetic != "" {
				continue
			}
			name := caller.Name()
			if !strings.HasPrefix(name, "Test") && !strings.HasPrefix(name, "Benchmark") {
				continue
			}
			pos := ref.MG.Prog.Fset.Position(caller.Pos())
			if pos.IsValid() {
				results = append(results, FuncInfo{
					Name: name,
					File: pos.Filename,
					Line: pos.Line,
				})
			}
		}
	}
	return results
}

// FindTestsByName returns Test/Benchmark functions whose name contains any
// of the given substrings. For example, "GetLocation" matches "TestGetLocation",
// "Test_GetLocation", "TestGetLocation_NotFound".
func (g *Graph) FindTestsByName(names []string) []FuncInfo {
	var results []FuncInfo
	for _, mg := range g.modules {
		for fn := range mg.CG.Nodes {
			if fn == nil || fn.Synthetic != "" {
				continue
			}
			fnName := fn.Name()
			if !strings.HasPrefix(fnName, "Test") && !strings.HasPrefix(fnName, "Benchmark") {
				continue
			}
			for _, name := range names {
				if strings.Contains(fnName, name) {
					pos := mg.Prog.Fset.Position(fn.Pos())
					if pos.IsValid() {
						results = append(results, FuncInfo{
							Name: fnName,
							File: pos.Filename,
							Line: pos.Line,
						})
					}
					break
				}
			}
		}
	}
	return results
}

// TestsCovering finds functions defined in changedFiles whose name matches
// one of symbolNames, then walks callers upward through the call graph and
// collects Test/Benchmark functions that are in one of the affectedDirs.
// Returns dir → []testName.
func (g *Graph) TestsCovering(affectedDirs map[string]bool, changedFiles []string, symbolNames []string) map[string][]string {
	if len(symbolNames) == 0 || len(affectedDirs) == 0 {
		return nil
	}

	nameSet := make(map[string]bool, len(symbolNames))
	for _, n := range symbolNames {
		nameSet[n] = true
	}

	results := map[string][]string{}
	seen := map[string]map[string]bool{} // dir → test name set (dedup)

	for _, mg := range g.modules {
		visited := map[*ssa.Function]bool{}
		var queue []*ssa.Function

		for _, file := range changedFiles {
			for _, fn := range mg.ByFile[file] {
				if nameSet[fn.Name()] && !visited[fn] {
					visited[fn] = true
					queue = append(queue, fn)
				}
			}
		}

		if len(queue) == 0 {
			continue
		}

		for len(queue) > 0 {
			fn := queue[0]
			queue = queue[1:]

			name := fn.Name()
			if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") {
				pos := mg.Prog.Fset.Position(fn.Pos())
				if pos.IsValid() {
					dir := filepath.Dir(pos.Filename)
					if affectedDirs[dir] {
						if seen[dir] == nil {
							seen[dir] = map[string]bool{}
						}
						if !seen[dir][name] {
							seen[dir][name] = true
							results[dir] = append(results[dir], name)
						}
					}
				}
				continue
			}

			node := mg.CG.Nodes[fn]
			if node == nil {
				continue
			}
			for _, edge := range node.In {
				caller := edge.Caller.Func
				if caller != nil && !visited[caller] {
					visited[caller] = true
					queue = append(queue, caller)
				}
			}
		}
	}

	return results
}
