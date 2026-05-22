package callgraph_test

import (
	"bytes"
	"strings"
	"testing"

	callgraph "github.com/urechandro/go-callgraph"
)

func TestWriteDOT_structure(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := callgraph.WriteDOT(&buf, g); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		"digraph callgraph {",
		"rankdir=LR;",
		"->",
		`"`,
		"}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestWriteDOT_deterministic(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	var a, b bytes.Buffer
	if err := callgraph.WriteDOT(&a, g); err != nil {
		t.Fatal(err)
	}
	if err := callgraph.WriteDOT(&b, g); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Error("WriteDOT output is not deterministic")
	}
}

func TestWriteDOTSubgraph(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	// B is called by A and calls C; blast radius should include A.
	refs := g.FindFunctions("B")
	if len(refs) == 0 {
		t.Fatal("function B not found")
	}
	all := append(refs, g.TransitiveCallers(refs)...)

	var buf bytes.Buffer
	if err := callgraph.WriteDOTSubgraph(&buf, all); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "digraph callgraph {") {
		t.Error("missing digraph header")
	}
	if !strings.Contains(out, "A") {
		t.Error("expected transitive caller A in subgraph output")
	}
	if !strings.Contains(out, "B") {
		t.Error("expected root B in subgraph output")
	}
}

func TestWriteDOT_knownEdge(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := callgraph.WriteDOT(&buf, g); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// fixture has A → B → C; all three must appear as labels
	for _, fn := range []string{"A", "B", "C"} {
		if !strings.Contains(out, fn) {
			t.Errorf("expected function %q in DOT output", fn)
		}
	}
}
