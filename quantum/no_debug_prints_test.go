package quantum_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestNoDebugPrintsInTests(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	var violations []string
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".trees", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}

			if !isPrintPackage(ident.Name) || !isPrintSelector(selector.Sel.Name) {
				return true
			}

			pos := fset.Position(call.Lparen)
			violations = append(violations, formatViolation(path, pos.Line, pos.Column))
			return true
		})

		return nil
	})
	if err != nil {
		t.Fatalf("scan tests: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("debug prints detected in tests:\n%s", strings.Join(violations, "\n"))
	}
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(wd, "..")), nil
}

func isPrintPackage(name string) bool {
	return name == "fmt" || name == "log"
}

func isPrintSelector(name string) bool {
	switch name {
	case "Print", "Printf", "Println", "Fprint", "Fprintf", "Fprintln":
		return true
	default:
		return false
	}
}

func formatViolation(path string, line, column int) string {
	return filepath.Clean(path) + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(column)
}
