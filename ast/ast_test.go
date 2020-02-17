package ast

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
	expected := map[string]Identifier{
		"m": Identifier{
			Ptr:  true,
			Pkg:  "http",
			Name: "Request",
		},
		"put": Identifier{
			Ptr:  false,
			Pkg:  "http",
			Name: "MethodPut",
		},
		"get": Identifier{
			Ptr:  false,
			Pkg:  "http",
			Name: "MethodGet",
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
					id := Identifier{}
					identify(ident, &id)
					compareIdentifiers(t, id, expectedIdentifier)
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

func TestIdentifyFunc(t *testing.T) {
	expected := map[string]Identifier{
		"idFunc": Identifier{
			Name:     "idFunc",
			MemberOf: "identifierStruct",
		},
		"HandleFunc": Identifier{
			Name: "HandleFunc",
			Pkg:  "http",
		},
		"ToLower": Identifier{
			Name: "ToLower",
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
			id := Identifier{}
			identify(callExpr.Fun, &id)
			expectedIdentifier, ok := expected[id.Name]
			if ok {
				delete(expected, id.Name)
				t.Run(fmt.Sprintf("function identifier for func %s", id.Name), func(t *testing.T) {
					if id.Pkg != expectedIdentifier.Pkg {
						t.Fatalf("expected pkg %s but got %s", expectedIdentifier.Pkg, id.Pkg)
					}
					if id.MemberOf != expectedIdentifier.MemberOf {
						t.Fatalf("expected memberOf %s but got %s", expectedIdentifier.MemberOf, id.MemberOf)
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

func TestFuncDeclInFile(t *testing.T) {
	type funcType struct {
		name     string
		memberOf string
		found    bool
	}
	expected := []funcType{
		funcType{
			name:     "idFunc",
			memberOf: "identifierStruct",
			found:    true,
		},
		funcType{
			name:     "idFuncWithPtr",
			memberOf: "*identifierStruct",
			found:    true,
		},
		funcType{
			name:  "readIdentifiers",
			found: true,
		},
		funcType{
			name:  "ToLower",
			found: false,
		},
	}
	parsed := parseASTFromFile(t, identifiersFilePath)
	for i := range expected {
		expectedFuncType := expected[i]
		funcDeclFound := ast.FuncDecl{}
		found := findFuncDeclInFile(parsed, expectedFuncType.name, expectedFuncType.memberOf, &funcDeclFound)
		if expectedFuncType.found && !found {
			t.Fatalf("expected function declaration %s to be found but wasn't", expectedFuncType.name)
		} else if !expectedFuncType.found && found {
			t.Fatalf("expected function declaration %s to not be found but was", expectedFuncType.name)
		}
	}
}

func TestValueForNode(t *testing.T) {
	expected := map[string]Identifier{
		"MethodPut": Identifier{
			Name: "MethodPut",
			Pkg:  "http",
		},
		"Delete": Identifier{
			Value: "Delete",
		},
		"MethodGet": Identifier{
			Name: "MethodGet",
			Pkg:  "http",
		},
		"cool": Identifier{
			Value: "cool",
		},
		"MethodPatch": Identifier{
			Name: "MethodPatch",
			Pkg:  "http",
		},
	}
	parsed := parseASTFromFile(t, identifiersFilePath)
	for i := range parsed.Decls {
		decl, ok := parsed.Decls[i].(*ast.FuncDecl)
		if ok {
			if decl.Name.Name == "findValue" {
				ast.Inspect(decl, func(n ast.Node) bool {
					switch x := n.(type) {
					case *ast.CallExpr:
						id := Identifier{}
						identify(x.Args[0], &id)
						if len(id.Name) > 0 {
							expectedIdentifier, ok := expected[id.Name]
							if ok {
								compareIdentifiers(t, id, expectedIdentifier)
								delete(expected, id.Name)
							} else {
								t.Fatalf("identifier not found %s", id.Name)
							}
						} else {
							expectedIdentifier, ok := expected[id.Value]
							if ok {
								compareIdentifiers(t, id, expectedIdentifier)
								delete(expected, id.Value)
							} else {
								t.Fatalf("identifier not found %s", id.Value)
							}
						}

					}
					return true
				})
				break
			}
		}
	}
	if len(expected) > 0 {
		t.Fatal("not all values were found")
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

func compareIdentifiers(t *testing.T, id, expectedIdentifier Identifier) {
	if id.Ptr != expectedIdentifier.Ptr {
		t.Fatalf("expected ptr %v but got %v", expectedIdentifier.Ptr, id.Ptr)
	}
	if id.Name != expectedIdentifier.Name {
		t.Fatalf("expected name %s but got %s", expectedIdentifier.Name, id.Name)
	}
	if id.Pkg != expectedIdentifier.Pkg {
		t.Fatalf("expected pkg %s but got %s", expectedIdentifier.Pkg, id.Pkg)
	}
	if id.Value != expectedIdentifier.Value {
		t.Fatalf("expected value %s but got %s", expectedIdentifier.Value, id.Value)
	}
}
