package callgraph

import (
	"fmt"
	"io"
)

// WriteTree prints root and its direct neighbors as an ASCII tree.
// Intended for use with DirectCallers / DirectCallees output:
//
//	callgraph.WriteTree(os.Stdout, "MyFunc", g.DirectCallers(refs))
func WriteTree(w io.Writer, root string, nodes []FuncInfo) {
	fmt.Fprintln(w, root)
	for i, n := range nodes {
		connector := "├── "
		if i == len(nodes)-1 {
			connector = "└── "
		}
		fmt.Fprintf(w, "%s%s  %s:%d\n", connector, n.Name, n.File, n.Line)
	}
}
