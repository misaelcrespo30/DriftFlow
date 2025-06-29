package helpers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"github.com/misaelcrespo30/DriftFlow/internal/models"
)

// LoadModels validates the directory defined by the MODELS_PATH environment
// variable and returns the compiled model instances. It ensures at least one
// exported struct exists in that directory.
func LoadModels() ([]interface{}, error) {
	modelPath := os.Getenv("MODELS_PATH")
	if modelPath == "" {
		modelPath = "internal/models"
	}

	info, err := os.Stat(modelPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrNotExist
	}

	files, err := filepath.Glob(filepath.Join(modelPath, "*.go"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, os.ErrNotExist
	}

	hasStruct := false
	fset := token.NewFileSet()
	for _, f := range files {
		astFile, err := parser.ParseFile(fset, f, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				if _, ok := ts.Type.(*ast.StructType); ok {
					hasStruct = true
					break
				}
			}
			if hasStruct {
				break
			}
		}
		if hasStruct {
			break
		}
	}
	if !hasStruct {
		return nil, os.ErrNotExist
	}

	// Return the compiled models slice. At the moment models are defined in
	// the internal/models package.
	return models.Models(), nil
}
