package swago

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"testing"
)

const (
	switchesFilePath = "./testdata/ast/switches.go"
)

func TestSearchForHttpMethodSwitch(t *testing.T) {
	f, err := os.Open(switchesFilePath)
	if err != nil {
		t.Fatalf("error opening file %s: %v\n", switchesFilePath, err)
	}
	defer f.Close()
	src, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("error reading file %v\n", err)
	}
	fset := token.NewFileSet() // positions are relative to fset
	parsed, err := parser.ParseFile(fset, switchesFilePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("error parsing ast %v\n", err)
	}
	for i := range parsed.Decls {
		declaration := parsed.Decls[i]
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if ok {
			t.Run(fmt.Sprintf("testing switch %d", i), func(t *testing.T) {
				searchForHTTPMethodSwitch(funcDecl.Body)
			})
		}
	}
}
