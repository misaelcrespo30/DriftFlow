package helpers

import (
	"fmt"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/interp"
	"golang.org/x/tools/go/ssa/ssautil"
)

// LoadModels loads a Go package located at dir and executes the Models
// function inside it using the `go/packages` and `ssa` interpreter. The
// function must return a `[]interface{}` describing all models.
func LoadModels(dir string) ([]interface{}, error) {
	cfg := &packages.Config{Dir: dir, Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 || len(pkgs) == 0 {
		return nil, fmt.Errorf("failed to load models package")
	}
	prog, ssaPkgs := ssautil.AllPackages(pkgs, ssa.SanityCheckFunctions)
	prog.Build()

	pkgSSA := ssaPkgs[0]
	modelsFn := pkgSSA.Func("Models")
	if modelsFn == nil {
		return nil, fmt.Errorf("Models function not found")
	}

	i := interp.New(interp.Config{}, prog)
	v, err := i.Eval(modelsFn)
	if err != nil {
		return nil, err
	}
	if res, ok := v.Interface().([]interface{}); ok {
		return res, nil
	}
	return nil, fmt.Errorf("invalid Models return type")
}
