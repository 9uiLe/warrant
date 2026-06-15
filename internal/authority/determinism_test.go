// @warrant-enforces RULE-DETERMINISTIC-GATE
package authority_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDeterministicGate は internal/check と internal/authority のソースに
// 非決定的判定（math/rand のインポート）がないことを検証する
func TestDeterministicGate(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")

	pkgDirs := []string{
		filepath.Join(repoRoot, "internal", "check"),
		filepath.Join(repoRoot, "internal", "authority"),
	}

	for _, dir := range pkgDirs {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse error in %s: %v", dir, err)
			continue
		}
		for _, pkg := range pkgs {
			for fname, f := range pkg.Files {
				for _, imp := range f.Imports {
					path := strings.Trim(imp.Path.Value, `"`)
					if path == "math/rand" || path == "math/rand/v2" {
						t.Errorf("非決定的インポート %q が %s に含まれている", path, fname)
					}
				}
			}
		}
	}
}

// TestDeterministicGate_NoRandUsage は check.go の AST をパースし
// math/rand セレクタ呼び出しがないことを確認する
func TestDeterministicGate_NoRandUsage(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")

	targetFile := filepath.Join(repoRoot, "internal", "check", "check.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, targetFile, nil, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "rand" {
			t.Errorf("check.go に rand セレクタ呼び出しが含まれている: %v", fset.Position(sel.Pos()))
		}
		return true
	})
}
