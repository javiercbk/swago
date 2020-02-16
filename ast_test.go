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
	readIdentifiers     = "readIdentifiers"
	callExprIdentifier  = "callExprIdentifier"
)

func TestDescribeIdentifier(t *testing.T) {
	expected := map[string]identifier{
		"m": identifier{
			ptr:  true,
			pkg:  "http",
			name: "Request",
		},
		"put": identifier{
			ptr:  false,
			pkg:  "http",
			name: "MethodPut",
		},
		"get": identifier{
			ptr:  false,
			pkg:  "http",
			name: "MethodGet",
		},
	}
	parsed := parseASTFromFile(t, switchesFilePath)
	var funcDecl *ast.FuncDecl
	var ok bool
	for _, d := range parsed.Decls {
		funcDecl, ok = d.(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == readIdentifiers {
			break
		}
	}
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if ok {
			expectedIdentifier, ok := expected[ident.Name]
			delete(expected, ident.Name)
			if ok {
				t.Run(fmt.Sprintf("identifier %s should match", ident.Name), func(t *testing.T) {
					id := identifier{
						name: ident.Name,
					}
					findTypeAndPkg(ident, &id)
					if id.ptr != expectedIdentifier.ptr {
						t.Fatalf("expected ptr %v but got %v", expectedIdentifier.ptr, id.ptr)
					}
					if id.name != expectedIdentifier.name {
						t.Fatalf("expected name %s but got %s", expectedIdentifier.name, id.name)
					}
					if id.pkg != expectedIdentifier.pkg {
						t.Fatalf("expected pkg %s but got %s", expectedIdentifier.pkg, id.pkg)
					}
				})
			}
			return false
		}
		return true
	})
	t.Run("found all identifiers", func(t *testing.T) {
		if len(expected) > 0 {
			t.Fatalf("expected the following identifiers to be found %v", expected)
		}
	})
}

func TestFindTypeAndPkgFunc(t *testing.T) {
	expected := map[string]identifier{
		"idFunc": identifier{
			name:     "idFunc",
			memberOf: "identifierStruct",
		},
		"HandleFunc": identifier{
			name: "HandleFunc",
			pkg:  "http",
		},
	}
	parsed := parseASTFromFile(t, identifiersFilePath)
	var funcDecl *ast.FuncDecl
	var ok bool
	for _, d := range parsed.Decls {
		funcDecl, ok = d.(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == callExprIdentifier {
			break
		}
	}
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if ok {
			id := identifier{}
			findTypeAndPkg(callExpr.Fun, &id)
			expectedIdentifier, ok := expected[id.name]
			if ok {
				delete(expected, id.name)
				t.Run(fmt.Sprintf("function identifier for func %s", id.name), func(t *testing.T) {
					if id.pkg != expectedIdentifier.pkg {
						t.Fatalf("expected pkg %s but got %s", expectedIdentifier.pkg, id.pkg)
					}
					if id.memberOf != expectedIdentifier.memberOf {
						t.Fatalf("expected memberOf %s but got %s", expectedIdentifier.memberOf, id.memberOf)
					}
				})
			}
		}
		return true
	})
	if len(expected) > 0 {
		t.Fatalf("could not find identifiers %v", expected)
	}
}

func parseASTFromFile(t *testing.T, filePath string) *ast.File {
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("error opening file %s: %v", switchesFilePath, err)
	}
	defer f.Close()
	src, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("error reading file %v", err)
	}
	fset := token.NewFileSet() // positions are relative to fset
	parsed, err := parser.ParseFile(fset, switchesFilePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("error parsing ast %v", err)
	}
	return parsed
}
