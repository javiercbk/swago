package swago

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	"github.com/javiercbk/swago/ast"
	"github.com/javiercbk/swago/criteria"
)

const (
	structProjectPath string = "testdata/struct-project"
	structRouteName   string = "Route"
)

func TestInitStructRoute(t *testing.T) {
	expectedFields := []ast.Field{
		ast.Field{
			Name: "Name",
			Type: "string",
			Tag:  "tag:\"Name\"",
		},
		ast.Field{
			Name: "Method",
			Type: "string",
			Tag:  "another:\"tag\"",
		},
		ast.Field{
			Name: "Pattern",
			Type: "string",
		},
		ast.Field{
			Name: "HandlerFunc",
			Type: "http.HandlerFunc",
		},
		ast.Field{
			Name: "ValidateJwt",
			Type: "bool",
		},
		ast.Field{
			Name: "AddlMiddlewares",
			Type: "[]Middleware",
		},
		ast.Field{
			Name: "HandlerTimeout",
			Type: "int",
		},
	}
	projectPath, err := filepath.Abs(structProjectPath)
	if err != nil {
		t.Fatalf("error getting absolute path of %s: %v", structProjectPath, err)
	}
	sg, err := NewSwaggerGenerator(projectPath, projectPath, log.New(ioutil.Discard, "", log.LstdFlags))
	if err != nil {
		t.Fatalf("error creating a swagger generator: %v", err)
	}
	projectCriteria := criteria.Criteria{
		Routes: []criteria.RouteCriteria{
			criteria.RouteCriteria{
				StructRoute: &criteria.StructRoute{
					Name:            structRouteName,
					Hierarchy:       "r",
					PathField:       "Pattern",
					HandlerField:    "HandlerFunc",
					HTTPMethodField: "Method",
				},
			},
		},
	}
	err = sg.initStructRoute(projectCriteria.Routes[0].StructRoute)
	if err != nil {
		t.Fatalf("error initializing struct route: %v", err)
	}
	structDef, ok := sg.structs[structRouteName]
	if !ok {
		t.Fatalf("error initializing struct route: %v", err)
	}
	if structDef.Name != structRouteName {
		t.Fatalf("expected struct name %s but got %s", structRouteName, structDef.Name)
	}
	for i := range structDef.Fields {
		found := structDef.Fields[i]
		expected := expectedFields[i]
		compareFields(t, found, expected)
	}
}

func TestFindStructCompositeLiteral(t *testing.T) {
	projectPath, err := filepath.Abs(structProjectPath)
	if err != nil {
		t.Fatalf("error getting absolute path of %s: %v", structProjectPath, err)
	}
	sg, err := NewSwaggerGenerator(projectPath, projectPath, log.New(ioutil.Discard, "", log.LstdFlags))
	if err != nil {
		t.Fatalf("error creating a swagger generator: %v", err)
	}
	projectCriteria := criteria.Criteria{
		Routes: []criteria.RouteCriteria{
			criteria.RouteCriteria{
				StructRoute: &criteria.StructRoute{
					Name:            structRouteName,
					Hierarchy:       "r",
					PathField:       "Pattern",
					HandlerField:    "HandlerFunc",
					HTTPMethodField: "Method",
				},
			},
		},
	}
	err = sg.initStructRoute(projectCriteria.Routes[0].StructRoute)
	if err != nil {
		t.Fatalf("error initializing struct route: %v", err)
	}
	err = sg.findStructCompositeLiteral(projectCriteria.Routes[0].StructRoute)
	if err != nil {
		t.Fatalf("error finding composite literal: %v", err)
	}
	foundRoutesLen := len(sg.routes)
	if foundRoutesLen != 3 {
		t.Fatalf("expected 3 routes to be found but got %d", foundRoutesLen)
	}
}

func compareFields(t *testing.T, found, expected ast.Field) {
	if found.Name != expected.Name {
		t.Fatalf("expected name to be %s but was %s", expected.Name, found.Name)
	}
	if found.Type != expected.Type {
		t.Fatalf("expected type to be %s but was %s", expected.Type, found.Type)
	}
	if found.Tag != expected.Tag {
		t.Fatalf("expected tag to be %s but was %s", expected.Tag, found.Tag)
	}
}
