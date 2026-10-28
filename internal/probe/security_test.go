package probe

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInsecureSkipVerifyIsLimitedToProbe(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo localizar el archivo de prueba")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", ".."))
	fset := token.NewFileSet()
	count := 0
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			assignment, ok := node.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := assignment.Key.(*ast.Ident)
			if !ok || key.Name != "InsecureSkipVerify" {
				return true
			}
			value, ok := assignment.Value.(*ast.Ident)
			if !ok || value.Name != "true" {
				t.Errorf("InsecureSkipVerify solo puede aparecer con valor true en %s", path)
				return true
			}
			allowed := filepath.Base(path) == "probe.go"
			inProbe := false
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if ok && function.Name.Name == "Probe" && function.Body.Pos() <= assignment.Pos() && assignment.End() <= function.Body.End() {
					inProbe = true
				}
			}
			if !allowed || !inProbe {
				t.Errorf("InsecureSkipVerify fuera de probe.Probe en %s", path)
			}
			count++
			return true
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("encontrados %d usos de InsecureSkipVerify=true; se esperaba uno", count)
	}
}
