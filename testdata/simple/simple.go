// Package simple is a minimal fixture for go-callgraph tests.
// Call chain: A → B → C. D is unreachable from tests.
package simple

// A calls B.
func A() { B() }

// B calls C.
func B() { C() }

// C is a leaf — called by B, calls nothing.
func C() {}

// D is unreachable from any test; used to verify it doesn't appear in
// RTA results (RTA prunes unreachable code).
func D() {}

// Greeter is an interface with a single method.
type Greeter interface {
	Greet() string
}

// Hello implements Greeter.
type Hello struct{ Name string }

// Greet satisfies Greeter.
func (h Hello) Greet() string { return "Hello, " + h.Name }

// CallGreeter calls g.Greet() via the interface — useful for verifying
// that RTA/CHA correctly resolves the dispatch to Hello.Greet.
func CallGreeter(g Greeter) string { return g.Greet() }
