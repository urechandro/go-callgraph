package callgraph

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// LoadMode is the packages.Config.Mode required for SSA construction.
// Exported so consumers can compose it with additional flags if needed.
const LoadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo

// LoadPackages loads Go packages from dir with the standard mode flags
// required for SSA call graph construction. Tests are always included.
// If no patterns are given, "./..." is used.
func LoadPackages(dir string, patterns ...string) ([]*packages.Package, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := &packages.Config{
		Mode:  LoadMode,
		Dir:   dir,
		Tests: true,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}

	return pkgs, nil
}
