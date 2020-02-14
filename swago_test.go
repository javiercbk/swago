package swago

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"
)

func TestCodeNavigatorConfig(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		goPath        string
		moduleName    string
		expectedError error
	}{
		{
			name:       "module project",
			path:       "./testdata/mod-project",
			goPath:     "./testdata/go-path",
			moduleName: "mod-project",
		}, {
			name:   "gopath project",
			path:   "./testdata/go-path/go-project",
			goPath: "./testdata/go-path",
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
			codeNavigator, err := NewCodeNavigator(path, goPath, log.New(ioutil.Discard, "", log.LstdFlags))
			if err != expectedError {
				t.Fatalf("expected error %v but got %v\n", expectedError, err)
			}
			if err == nil {
				if codeNavigator.goPath != goPath {
					t.Fatalf("expected go path %s but got %s\n", goPath, codeNavigator.goPath)
				}
				if codeNavigator.rootPath != path {
					t.Fatalf("expected root path %s but got %s\n", path, codeNavigator.rootPath)
				}
				if codeNavigator.moduleName != moduleName {
					t.Fatalf("expected module name %s but got %s\n", moduleName, codeNavigator.moduleName)
				}
			}
		})
	}
}
