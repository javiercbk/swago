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
	identifiersFilePath = "./testdata/ast/identifiers.go"
)

func TestDescribeIdentifier(t *testing.T) {
	f, err := os.Open(identifiersFilePath)
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
	ast.Inspect(parsed, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if ok {
			id := identifier{
				name: ident.Name,
			}
			findTypeAndPkg(ident, &id)
			fmt.Printf("%v\n", id)
			return false
		}
		return true
	})
}
