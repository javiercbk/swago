package swago

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"
)

const (
	moduleRootPath = "./testdata/mod-project"
	moduleGoPath   = "./testdata/go-path"
)

func TestCodeExplorerConfig(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		goPath        string
		moduleName    string
		expectedError error
	}{
		{
			name:       "module project",
			path:       moduleRootPath,
			goPath:     moduleGoPath,
			moduleName: "modproj",
		}, {
			name:   "gopath project",
			path:   "./testdata/go-path/go-project",
			goPath: moduleGoPath,
		},
	}
	for i := range tests {
		t.Run(fmt.Sprintf("code navigator config: %s", tests[i].name), func(t *testing.T) {
			path, err := filepath.Abs(tests[i].path)
			if err != nil {
				t.Fatalf("error getting absolute path of %s: %v", tests[i].path, err)
			}
			goPath, err := filepath.Abs(tests[i].goPath)
			if err != nil {
				t.Fatalf("error getting absolute path of %s: %v", tests[i].goPath, err)
			}
			expectedError := tests[i].expectedError
			moduleName := tests[i].moduleName
			swaggerGenerator, err := NewSwaggerGenerator(path, goPath, log.New(ioutil.Discard, "", log.LstdFlags))
			if err != expectedError {
				t.Fatalf("expected error %v but got %v\n", expectedError, err)
			}
			if err == nil {
				if swaggerGenerator.goPath != goPath {
					t.Fatalf("expected go path %s but got %s\n", goPath, swaggerGenerator.goPath)
				}
				if swaggerGenerator.rootPath != path {
					t.Fatalf("expected root path %s but got %s\n", path, swaggerGenerator.rootPath)
				}
				if swaggerGenerator.moduleName != moduleName {
					t.Fatalf("expected module name %s but got %s\n", moduleName, swaggerGenerator.moduleName)
				}
			}
		})
	}
}
