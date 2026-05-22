package callgraph

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// WriteDOT writes the call graph in Graphviz DOT format to w.
// Render with: dot -Tsvg out.dot > out.svg
func WriteDOT(w io.Writer, g *Graph) error {
	type rawEdge struct{ caller, callee string }
	var rawEdges []rawEdge
	seen := map[string]struct{}{}
	seenEdge := map[string]struct{}{}

	g.ForEachEdge(func(e EdgeInfo) bool {
		c, d := e.Caller.String(), e.Callee.String()
		key := c + "\x00" + d
		if _, dup := seenEdge[key]; dup {
			return true
		}
		seenEdge[key] = struct{}{}
		rawEdges = append(rawEdges, rawEdge{c, d})
		seen[c] = struct{}{}
		seen[d] = struct{}{}
		return true
	})

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
