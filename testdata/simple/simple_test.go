package simple_test

import (
	"testing"

	"github.com/urechandro/go-callgraph/testdata/simple"
)

// TestA exercises the A→B→C call chain. RTA uses this as a root.
func TestA(t *testing.T) {
	simple.A()
}

// TestGreeter exercises the interface dispatch path.
func TestGreeter(t *testing.T) {
	h := simple.Hello{Name: "world"}
	got := simple.CallGreeter(h)
	if got != "Hello, world" {
		t.Errorf("got %q", got)
	}
}
