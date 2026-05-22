package callgraph_test

import (
	"bytes"
	"testing"

	callgraph "github.com/urechandro/go-callgraph"
)

func TestWriteTree(t *testing.T) {
	nodes := []callgraph.FuncInfo{
		{Name: "Foo", File: "foo.go", Line: 10},
		{Name: "Bar", File: "bar.go", Line: 20},
	}
	var buf bytes.Buffer
	callgraph.WriteTree(&buf, "Root", nodes)
	want := "Root\n├── Foo  foo.go:10\n└── Bar  bar.go:20\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestWriteTree_single(t *testing.T) {
	nodes := []callgraph.FuncInfo{{Name: "Only", File: "x.go", Line: 1}}
	var buf bytes.Buffer
	callgraph.WriteTree(&buf, "Root", nodes)
	want := "Root\n└── Only  x.go:1\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestWriteTree_empty(t *testing.T) {
	var buf bytes.Buffer
	callgraph.WriteTree(&buf, "Root", nil)
	if got := buf.String(); got != "Root\n" {
		t.Errorf("got %q, want %q", got, "Root\n")
	}
}
