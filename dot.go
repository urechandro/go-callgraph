package callgraph

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// WriteDOTSubgraph writes only the subgraph formed by the given functions in
// Graphviz DOT format. Edges where either end is not in refs are omitted.
//
// Typical use — blast radius of a symbol:
//
//	refs := g.FindFunctions("MyFunc")
//	all := append(refs, g.TransitiveCallers(refs)...)
//	callgraph.WriteDOTSubgraph(w, all)
//
// Forward slice (what a function reaches):
//
//	all := append(refs, g.TransitiveCallees(refs)...)
//	callgraph.WriteDOTSubgraph(w, all)
func WriteDOTSubgraph(w io.Writer, refs []FuncRef) error {
	inSet := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref.Fn != nil {
			inSet[ref.Fn.String()] = struct{}{}
		}
	}

	var rawEdges []dotEdge
	seen := map[string]struct{}{}
	seenEdge := map[string]struct{}{}

	for _, ref := range refs {
		if ref.Fn == nil || ref.Node == nil {
			continue
		}
		seen[ref.Fn.String()] = struct{}{}
		for _, edge := range ref.Node.Out {
			callee := edge.Callee.Func
			if callee == nil {
				continue
			}
			if _, ok := inSet[callee.String()]; !ok {
				continue
			}
			key := ref.Fn.String() + "\x00" + callee.String()
			if _, dup := seenEdge[key]; dup {
				continue
			}
			seenEdge[key] = struct{}{}
			rawEdges = append(rawEdges, dotEdge{ref.Fn.String(), callee.String()})
			seen[callee.String()] = struct{}{}
		}
	}

	return writeDOT(w, seen, rawEdges)
}

// WriteDOT writes the call graph in Graphviz DOT format to w.
// Render with: dot -Tsvg out.dot > out.svg
func WriteDOT(w io.Writer, g *Graph) error {
	var rawEdges []dotEdge
	seen := map[string]struct{}{}
	seenEdge := map[string]struct{}{}

	g.ForEachEdge(func(e EdgeInfo) bool {
		c, d := e.Caller.String(), e.Callee.String()
		key := c + "\x00" + d
		if _, dup := seenEdge[key]; dup {
			return true
		}
		seenEdge[key] = struct{}{}
		rawEdges = append(rawEdges, dotEdge{c, d})
		seen[c] = struct{}{}
		seen[d] = struct{}{}
		return true
	})

	return writeDOT(w, seen, rawEdges)
}

type dotEdge struct{ caller, callee string }

func writeDOT(w io.Writer, seen map[string]struct{}, rawEdges []dotEdge) error {
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	ids := make(map[string]int, len(names))
	for i, n := range names {
		ids[n] = i
	}

	sort.Slice(rawEdges, func(i, j int) bool {
		if rawEdges[i].caller != rawEdges[j].caller {
			return rawEdges[i].caller < rawEdges[j].caller
		}
		return rawEdges[i].callee < rawEdges[j].callee
	})

	var sb strings.Builder
	sb.WriteString("digraph callgraph {\n")
	sb.WriteString("\trankdir=LR;\n")
	sb.WriteString("\tnode [shape=box fontname=\"Courier\" fontsize=10];\n")
	for i, name := range names {
		fmt.Fprintf(&sb, "\t%d [label=%q];\n", i, name)
	}
	for _, e := range rawEdges {
		fmt.Fprintf(&sb, "\t%d -> %d;\n", ids[e.caller], ids[e.callee])
	}
	sb.WriteString("}\n")

	_, err := io.WriteString(w, sb.String())
	return err
}
