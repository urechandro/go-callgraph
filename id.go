package callgraph

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// QualifiedID returns a stable fully-qualified ID for an SSA function,
// in the form "pkg/path.FuncName" or "pkg/path.Type.MethodName".
// Returns "" for synthetic functions (wrappers, thunks) or builtins.
func QualifiedID(fn *ssa.Function) string {
	obj := fn.Object()
	if obj == nil {
		return "" // synthetic (wrappers, thunks)
	}
	if obj.Pkg() == nil {
		return "" // builtin
	}
	return QualifiedObjID(obj.Pkg().Path(), obj)
}

// FuncPkgPath returns the package path of an SSA function, or "".
func FuncPkgPath(fn *ssa.Function) string {
	if fn.Package() != nil && fn.Package().Pkg != nil {
		return fn.Package().Pkg.Path()
	}
	if obj := fn.Object(); obj != nil && obj.Pkg() != nil {
		return obj.Pkg().Path()
	}
	return ""
}

// QualifiedObjID builds a stable fully-qualified ID from a package path and
// types.Object. For methods, the format is "pkg/path.TypeName.MethodName".
// For functions, "pkg/path.FuncName". This can be used for both SSA functions
// and AST-level symbol indexing.
func QualifiedObjID(pkgPath string, obj types.Object) string {
	if obj.Parent() == nil {
		// Method — include receiver type name.
		fn, ok := obj.(*types.Func)
		if ok {
			sig := fn.Type().(*types.Signature)
			if sig.Recv() != nil {
				recv := receiverTypeName(sig.Recv().Type())
				return fmt.Sprintf("%s.%s.%s", pkgPath, recv, obj.Name())
			}
		}
	}
	return fmt.Sprintf("%s.%s", pkgPath, obj.Name())
}

// receiverTypeName returns just the type name (no package path) for a method receiver.
func receiverTypeName(t types.Type) string {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return types.TypeString(t, nil)
}
