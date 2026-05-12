package callgraph_test

import (
	"path/filepath"
	"runtime"
	"testing"

	callgraph "github.com/urechandro/go-callgraph"
)

// fixtureDir returns the path to testdata/simple.
func fixtureDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "simple")
}

func TestBuild_RTA(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	if n := g.SymbolCount(); n == 0 {
		t.Error("expected non-zero symbol count")
	}
}

func TestBuild_CHA(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.CHA)
	if err != nil {
		t.Fatal(err)
	}
	if n := g.SymbolCount(); n == 0 {
		t.Error("expected non-zero symbol count")
	}
}

func TestBuildFromPackages(t *testing.T) {
	pkgs, err := callgraph.LoadPackages(fixtureDir())
	if err != nil {
		t.Fatal(err)
	}
	g, err := callgraph.BuildFromPackages(pkgs, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	if n := g.SymbolCount(); n == 0 {
		t.Error("expected non-zero symbol count after BuildFromPackages")
	}
}

func TestFindFunctions(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"A", "B", "C"} {
		refs := g.FindFunctions(name)
		if len(refs) == 0 {
			t.Errorf("FindFunctions(%q) returned no results", name)
		}
	}
}

func TestDirectCallers_chain(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}

	// B should be called by A.
	refs := g.FindFunctions("B")
	if len(refs) == 0 {
		t.Fatal("FindFunctions(B) returned nothing")
	}
	callers := g.DirectCallers(refs)
	if !containsName(callers, "A") {
		t.Errorf("expected A in callers of B, got %v", names(callers))
	}
}

func TestDirectCallees_chain(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}

	// A should call B.
	refs := g.FindFunctions("A")
	if len(refs) == 0 {
		t.Fatal("FindFunctions(A) returned nothing")
	}
	callees := g.DirectCallees(refs)
	if !containsName(callees, "B") {
		t.Errorf("expected B in callees of A, got %v", names(callees))
	}
}

func TestCallersToTests(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}

	// C is called transitively by TestA (TestA→A→B→C). CallersToTests should
	// walk up and find TestA.
	refs := g.FindFunctions("C")
	if len(refs) == 0 {
		t.Fatal("FindFunctions(C) returned nothing")
	}
	tests := g.CallersToTests(refs)
	if !containsName(tests, "TestA") {
		t.Errorf("expected TestA in transitive callers of C, got %v", names(tests))
	}
}

func TestDirectCallerTests(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}

	// A is directly called by TestA.
	refs := g.FindFunctions("A")
	if len(refs) == 0 {
		t.Fatal("FindFunctions(A) returned nothing")
	}
	tests := g.DirectCallerTests(refs)
	if !containsName(tests, "TestA") {
		t.Errorf("expected TestA in direct caller tests of A, got %v", names(tests))
	}
}

func TestFindTestsByName(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	results := g.FindTestsByName([]string{"Greeter"})
	if !containsName(results, "TestGreeter") {
		t.Errorf("expected TestGreeter, got %v", names(results))
	}
}

func TestForEachEdge(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	g.ForEachEdge(func(e callgraph.EdgeInfo) bool {
		count++
		return true
	})
	if count == 0 {
		t.Error("ForEachEdge yielded no edges")
	}
}

func TestQualifiedID(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}
	g.ForEachEdge(func(e callgraph.EdgeInfo) bool {
		id := callgraph.QualifiedID(e.Caller)
		if id == "" {
			// Synthetic functions are filtered out in ForEachEdge, but
			// QualifiedID can still return "" for builtins — skip those.
			return true
		}
		pkg := callgraph.FuncPkgPath(e.Caller)
		if pkg == "" {
			t.Errorf("FuncPkgPath returned empty for %q", id)
		}
		return true
	})
}

func TestDeadFunctions_RTA(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}

	dead := g.DeadFunctions(true)

	// D is never called — it must appear in the dead list.
	if !containsName(dead, "D") {
		t.Errorf("expected D in dead functions, got %v", names(dead))
	}
	// A, B, C are reachable via TestA → A → B → C.
	for _, name := range []string{"A", "B", "C"} {
		if containsName(dead, name) {
			t.Errorf("expected %s to be reachable, but it appeared in dead list", name)
		}
	}
}

func TestDeadFunctions_CHA_NoFalseNegatives(t *testing.T) {
	// CHA over-approximates indirect func() calls, so D may not appear as dead.
	// But functions that ARE called (A, B, C) must never be reported as dead.
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.CHA)
	if err != nil {
		t.Fatal(err)
	}

	dead := g.DeadFunctions(true)

	for _, name := range []string{"A", "B", "C"} {
		if containsName(dead, name) {
			t.Errorf("expected %s to be reachable (CHA), but it appeared in dead list", name)
		}
	}
}

func TestDeadFunctions_ExcludesExportedByDefault(t *testing.T) {
	g, err := callgraph.Build([]string{fixtureDir()}, callgraph.RTA)
	if err != nil {
		t.Fatal(err)
	}

	// With includeExported=false, exported D should not be reported.
	dead := g.DeadFunctions(false)
	if containsName(dead, "D") {
		t.Error("D should not appear when includeExported=false")
	}
}

// helpers

func containsName(fns []callgraph.FuncInfo, name string) bool {
	for _, f := range fns {
		if f.Name == name {
			return true
		}
	}
	return false
}

func names(fns []callgraph.FuncInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}
