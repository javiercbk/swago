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

func TestExtractVariable(t *testing.T) {
	expected := map[string]Variable{
		"m": Variable{
			Type: "http.*Request.Method",
			Name: "m",
		},
		"put": Variable{
			Name: "put",
			Type: "http.MethodPut",
		},
		"get": Variable{
			Name: "get",
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
		if n != nil {
			v := Variable{}
			switch x := n.(type) {
			case *ast.BlockStmt:
				return true
			case *ast.SwitchStmt:
				return true
			case *ast.Ident:
				return true
			case *ast.AssignStmt:
				extractVariable(x, &v)
				assertVariable(t, v, expected)
				return false
			case *ast.CaseClause:
				if len(x.List) > 0 {
					extractVariable(x.List[0], &v)
					assertVariable(t, v, expected)
				}
				return false
			default:
				return false
			}
		}
		return false
	})
	t.Run("found all identifiers", func(t *testing.T) {
		if len(expected) > 0 {
			t.Fatalf("expected the following identifiers to be found %v", expected)
		}
	})
}

func assertVariable(t *testing.T, v Variable, expected map[string]Variable) {
	expectedVariable, ok := expected[v.Name]
	if ok {
		delete(expected, v.Name)
		t.Run(fmt.Sprintf("identifier %s should match", v.Name), func(t *testing.T) {
			compareVariables(t, v, expectedVariable)
		})
	}
}

func TestIdentifyFunc(t *testing.T) {
	expected := map[string]Variable{
		"idFunc": Variable{
			Name: "idFunc",
			Type: "identifierStruct",
		},
		"HandleFunc": Variable{
			Name: "HandleFunc",
			Type: "http",
		},
		"ToLower": Variable{
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
			v := Variable{}
			extractVariable(callExpr.Fun, &v)
			expectedIdentifier, ok := expected[v.Name]
			if ok {
				delete(expected, v.Name)
				t.Run(fmt.Sprintf("function identifier for func %s", v.Name), func(t *testing.T) {
					if v.Type != expectedIdentifier.Type {
						t.Fatalf("expected type %s but got %s", expectedIdentifier.Type, v.Type)
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
	expected := map[string]Variable{
		"MethodPut": Variable{
			Name: "MethodPut",
			Type: "http",
		},
		"Delete": Variable{
			Name:  "del",
			Value: "Delete",
		},
		"MethodGet": Variable{
			Name: "MethodGet",
			Type: "http",
		},
		"cool": Variable{
			Value: "cool",
		},
		"MethodPatch": Variable{
			Name: "MethodPatch",
			Type: "http",
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
						v := Variable{}
						extractVariable(x.Args[0], &v)
						if len(v.Name) > 0 {
							expectedIdentifier, ok := expected[v.Name]
							if ok {
								compareVariables(t, v, expectedIdentifier)
								delete(expected, v.Name)
							} else {
								t.Fatalf("identifier not found %s", v.Name)
							}
						} else {
							expectedIdentifier, ok := expected[v.Value]
							if ok {
								compareVariables(t, v, expectedIdentifier)
								delete(expected, v.Value)
							} else {
								t.Fatalf("identifier not found %s", v.Value)
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

func compareVariables(t *testing.T, v, expectedVar Variable) {
	if v.Name != expectedVar.Name {
		t.Fatalf("expected name %s but got %s", expectedVar.Name, v.Name)
	}
	if v.Type != expectedVar.Type {
		t.Fatalf("expected type %s but got %s", expectedVar.Type, v.Type)
	}
	if v.Value != expectedVar.Value {
		t.Fatalf("expected value %s but got %s", expectedVar.Value, v.Value)
	}
}
