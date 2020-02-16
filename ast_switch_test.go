package swago

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

const (
	switchesFilePath        = "./testdata/ast/switches.go"
	switchesNonHTTPFilePath = "./testdata/ast/switches_non_http.go"
)

func TestSearchForHttpMethodSwitch(t *testing.T) {
	expected := map[string][]string{
		"idealSwitch": []string{
			strings.ToUpper(http.MethodGet),
			strings.ToUpper(http.MethodPost),
			strings.ToUpper(http.MethodPut),
			strings.ToUpper(http.MethodDelete),
			strings.ToUpper(http.MethodPatch),
		},
		"notCompleteSwitch": []string{
			strings.ToUpper(http.MethodGet),
			strings.ToUpper(http.MethodPost),
		},
		"switchWithVars": []string{
			strings.ToUpper(http.MethodGet),
			strings.ToUpper(http.MethodPost),
			strings.ToUpper(http.MethodPut),
			strings.ToUpper(http.MethodDelete),
		},
	}
	testSearchHTTPMethodSwithForPkg(t, switchesFilePath, "http", expected)
}

func TestSearchForNonHttpMethodSwitch(t *testing.T) {
	expected := map[string][]string{
		"coolIdealSwitch": []string{
			strings.ToUpper(http.MethodGet),
			strings.ToUpper(http.MethodPost),
			strings.ToUpper(http.MethodPut),
			strings.ToUpper(http.MethodDelete),
			strings.ToUpper(http.MethodPatch),
		},
		"coolNotCompleteSwitch": []string{
			strings.ToUpper(http.MethodGet),
			strings.ToUpper(http.MethodPost),
		},
		"coolSwitchWithVars": []string{
			strings.ToUpper(http.MethodGet),
			strings.ToUpper(http.MethodPost),
			strings.ToUpper(http.MethodPut),
			strings.ToUpper(http.MethodDelete),
		},
	}
	testSearchHTTPMethodSwithForPkg(t, switchesNonHTTPFilePath, "cool", expected)
}

func testSearchHTTPMethodSwithForPkg(t *testing.T, filePath, pkg string, expected map[string][]string) {
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("error opening file %s: %v\n", filePath, err)
	}
	defer f.Close()
	src, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("error reading file %v\n", err)
	}
	fset := token.NewFileSet() // positions are relative to fset
	parsed, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("error parsing ast %v\n", err)
	}
	for i := range parsed.Decls {
		declaration := parsed.Decls[i]
		funcDecl, ok := declaration.(*ast.FuncDecl)
		if ok {
			funcName := funcDecl.Name.Name
			t.Run(fmt.Sprintf("testing switch %s", funcName), func(t *testing.T) {
				methods, ok := expected[funcName]
				if !ok {
					t.Fatalf("func %s was not found in map\n", funcName)
				}
				methodsFound := searchForHTTPMethodSwitch(funcDecl.Body, pkg)
				methodsLen := len(methods)
				methodsFoundLen := len(methodsFound)
				if methodsLen != methodsFoundLen {
					t.Fatalf("expected to find %d method but found %d\n", methodsLen, methodsFoundLen)
				}
				for j := range methods {
					expectedMethod := methods[j]
					methodFound := methodsFound[j].HTTPMethod
					if methodFound != expectedMethod {
						t.Fatalf("expected method %d to be %s but was %s\n", j, expectedMethod, methodFound)
					}
				}
			})
		}
	}
}
