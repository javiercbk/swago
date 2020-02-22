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
			Hierarchy: "http.*Request.Method",
			Name:      "m",
		},
		"put": Variable{
			Name:      "put",
			Hierarchy: "http.MethodPut",
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
	expected := map[string]FuncCall{
		"idFunc": FuncCall{
			Function: Function{
				Name:      "idFunc",
				Hierarchy: "identifierStruct",
			},
		},
		"HandleFunc": FuncCall{
			Function: Function{
				Name:      "HandleFunc",
				Hierarchy: "http",
			},
			Args: []Variable{
				Variable{
					Name: "coolPath",
				},
				Variable{},
			},
		},
		"ToLower": FuncCall{
			Function: Function{
				Name: "ToLower",
			},
			Args: []Variable{
				Variable{
					GoType: "string",
					Value:  "/cool",
				},
			},
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
			f := FuncCall{}
			extractFuncCall(callExpr, &f)
			expectedFuncCall, ok := expected[f.Function.Name]
			if ok {
				delete(expected, f.Function.Name)
				t.Run(fmt.Sprintf("function identifier for func %s", f.Function.Name), func(t *testing.T) {
					compareFuncCall(t, f, expectedFuncCall)
				})
			}
		}
		return true
	})
	if len(expected) > 0 {
		t.Fatalf("could not find identifiers %v", expected)
	}
}

func compareFuncCall(t *testing.T, call, expected FuncCall) {
	compareFunction(t, call.Function, expected.Function)
	lenArgs := len(call.Args)
	lenExpectedArgs := len(expected.Args)
	if lenArgs != lenExpectedArgs {
		t.Fatalf("expected %d arguments but got %d\n", lenArgs, lenExpectedArgs)
	}
	for i := range call.Args {
		compareVariables(t, call.Args[i], expected.Args[i])
	}
	lenRet := len(call.Return)
	lenExpectedRet := len(expected.Return)
	if lenRet != lenExpectedRet {
		t.Fatalf("expected %d returns but got %d\n", lenRet, lenExpectedRet)
	}
	for i := range call.Return {
		compareVariables(t, call.Return[i], expected.Return[i])
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
			Name: "put",
			Definition: &Variable{
				Name:      "MethodPut",
				Hierarchy: "http",
			},
		},
		"Delete": Variable{
			Name: "del",
			Definition: &Variable{
				GoType: "string",
				Value:  "Delete",
			},
		},
		"MethodGet": Variable{
			Name: "get",
			Definition: &Variable{
				Name: "identifiersGet",
			},
		},
		"cool": Variable{
			Value: "cool",
		},
		"MethodPatch": Variable{
			Name:      "MethodPatch",
			Hierarchy: "http",
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
						f := FuncCall{}
						extractFuncCall(x, &f)
						if len(f.Args) > 0 {
							arg := f.Args[0]
							if len(arg.Value) == 0 {
								var expectedVar Variable
								var ok bool
								var key string
								if arg.Definition != nil {
									if len(arg.Definition.Value) > 0 {
										expectedVar, ok = expected[arg.Definition.Value]
										key = arg.Definition.Value
									} else {
										lastVar := extractOriginalDefinition(arg)
										expectedVar, ok = expected[lastVar.Name]
										key = lastVar.Name
									}
								} else {
									lastVar := extractOriginalDefinition(arg)
									expectedVar, ok = expected[lastVar.Name]
									key = lastVar.Name
								}
								if ok {
									compareVariables(t, arg, expectedVar)
									delete(expected, key)
								} else {
									t.Fatalf("identifier not found %s", f.Function.Name)
								}
							} else {
								expectedVar, ok := expected[arg.Value]
								if ok {
									compareVariables(t, arg, expectedVar)
									delete(expected, arg.Value)
								} else {
									t.Fatalf("identifier not found %s", arg.Value)
								}
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
		t.Fatalf("not all values were found, missing %v\n", expected)
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
	if v.Hierarchy != expectedVar.Hierarchy {
		t.Fatalf("expected type %s but got %s", expectedVar.Hierarchy, v.Hierarchy)
	}
	if v.Value != expectedVar.Value {
		t.Fatalf("expected value %s but got %s", expectedVar.Value, v.Value)
	}
}

func compareFunction(t *testing.T, v, expected Function) {
	if v.Name != expected.Name {
		t.Fatalf("expected name %s but got %s", expected.Name, v.Name)
	}
	if v.Hierarchy != expected.Hierarchy {
		t.Fatalf("expected type %s but got %s", expected.Hierarchy, v.Hierarchy)
	}
	if len(v.Args) != len(expected.Args) {
		t.Fatalf("expected %d args got %d", len(v.Args), len(expected.Args))
	}
	for i := range v.Args {
		compareVariables(t, v.Args[i], expected.Args[i])
	}
	if len(v.Return) != len(expected.Return) {
		t.Fatalf("expected %d return got %d", len(v.Return), len(expected.Return))
	}
	for i := range v.Return {
		compareVariables(t, v.Return[i], expected.Return[i])
	}
}
