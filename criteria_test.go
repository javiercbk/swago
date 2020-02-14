package swago

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
)

var (
	defaultCriteria = Criteria{
		Routes: []RouteCriteria{
			RouteCriteria{
				Pkg:          "packageName0",
				FuncName:     "funcName0",
				VarName:      "varName0",
				HTTPMethod:   "GET",
				PathIndex:    1,
				HandlerIndex: 2,
			},
		},
		Request: []CallCriteria{
			{
				Pkg:        "requestPackageName0",
				FuncName:   "requestFuncName0",
				VarName:    "requestVarName0",
				ParamIndex: 0,
			},
		},
		Response: []CallCriteria{
			{
				Pkg:        "responsePackageName0",
				FuncName:   "responseFuncName0",
				VarName:    "responseVarName0",
				ParamIndex: 0,
			},
		},
	}
	missingRouteHandlerIndexCriteria = Criteria{
		Routes: []RouteCriteria{
			RouteCriteria{
				Pkg:          "packageName0",
				FuncName:     "funcName0",
				VarName:      "varName0",
				HTTPMethod:   "GET",
				PathIndex:    1,
				HandlerIndex: 0,
			},
		},
		Request: []CallCriteria{
			{
				Pkg:        "requestPackageName0",
				FuncName:   "requestFuncName0",
				VarName:    "requestVarName0",
				ParamIndex: 0,
			},
		},
		Response: []CallCriteria{
			{
				Pkg:        "responsePackageName0",
				FuncName:   "responseFuncName0",
				VarName:    "responseVarName0",
				ParamIndex: 0,
			},
		},
	}
	missingRoutePathIndexCriteria = Criteria{
		Routes: []RouteCriteria{
			RouteCriteria{
				Pkg:          "packageName0",
				FuncName:     "funcName0",
				VarName:      "varName0",
				HTTPMethod:   "GET",
				PathIndex:    0,
				HandlerIndex: 2,
			},
		},
		Request: []CallCriteria{
			{
				Pkg:        "requestPackageName0",
				FuncName:   "requestFuncName0",
				VarName:    "requestVarName0",
				ParamIndex: 0,
			},
		},
		Response: []CallCriteria{
			{
				Pkg:        "responsePackageName0",
				FuncName:   "responseFuncName0",
				VarName:    "responseVarName0",
				ParamIndex: 0,
			},
		},
	}
	fullCriteria = Criteria{
		Routes: []RouteCriteria{
			RouteCriteria{
				Pkg:          "packageName0",
				FuncName:     "funcName0",
				VarName:      "varName0",
				HTTPMethod:   "GET",
				PathIndex:    1,
				HandlerIndex: 2,
			},
			RouteCriteria{
				FuncName:     "funcName1",
				VarName:      "varName1",
				HTTPMethod:   "POST",
				PathIndex:    1,
				HandlerIndex: 2,
			},
			RouteCriteria{
				FuncName:     "funcName2",
				HTTPMethod:   "PUT",
				PathIndex:    1,
				HandlerIndex: 2,
			},
			RouteCriteria{
				FuncName:     "funcName3",
				HTTPMethod:   "PATCH",
				PathIndex:    0,
				HandlerIndex: 2,
			},
			RouteCriteria{
				FuncName:     "funcName4",
				HTTPMethod:   "DELETE",
				PathIndex:    0,
				HandlerIndex: 1,
			},
			RouteCriteria{
				FuncName:     "funcName5",
				PathIndex:    0,
				HandlerIndex: 1,
			},
		},
		Request: []CallCriteria{
			{
				Pkg:        "requestPackageName0",
				FuncName:   "requestFuncName0",
				VarName:    "requestVarName0",
				ParamIndex: 0,
			},
			{
				FuncName:   "requestFuncName1",
				VarName:    "requestVarName1",
				ParamIndex: 1,
			},
			{
				FuncName:   "requestFuncName2",
				ParamIndex: 2,
			},
		},
		Response: []CallCriteria{
			{
				Pkg:        "responsePackageName0",
				FuncName:   "responseFuncName0",
				VarName:    "responseVarName0",
				ParamIndex: 0,
			},
			{
				FuncName:   "responseFuncName1",
				VarName:    "responseVarName1",
				ParamIndex: 1,
			},
			{
				FuncName:   "responseFuncName2",
				ParamIndex: 2,
			},
		},
	}
)

func TestCriteriaParser(t *testing.T) {
	tests := []struct {
		yamlFile         string
		expectedCriteria Criteria
		expectedError    error
	}{
		{
			yamlFile:         "valid.yml",
			expectedCriteria: fullCriteria,
		}, {
			yamlFile:      "equal-route-path-handler-index.yml",
			expectedError: ErrInvalidRoute,
		}, {
			yamlFile:      "missing-request-func-name.yml",
			expectedError: ErrInvalidCallCriteria,
		}, {
			yamlFile:         "missing-request-param-index.yml",
			expectedCriteria: defaultCriteria,
		}, {
			yamlFile:      "missing-request.yml",
			expectedError: ErrMissingRequest,
		}, {
			yamlFile:      "missing-response-func-name.yml",
			expectedError: ErrInvalidCallCriteria,
		}, {
			yamlFile:         "missing-response-param-index.yml",
			expectedCriteria: defaultCriteria,
		}, {
			yamlFile:      "missing-response.yml",
			expectedError: ErrMissingResponse,
		}, {
			yamlFile:      "missing-route-func-name.yml",
			expectedError: ErrInvalidRoute,
		}, {
			yamlFile:         "missing-route-handler-index.yml",
			expectedCriteria: missingRouteHandlerIndexCriteria,
		}, {
			yamlFile:      "missing-route-path-handler-index.yml",
			expectedError: ErrInvalidRoute,
		}, {
			yamlFile:         "missing-route-path-index.yml",
			expectedCriteria: missingRoutePathIndexCriteria,
		}, {
			yamlFile:      "missing-routes.yml",
			expectedError: ErrMissingRoutes,
		}, {
			yamlFile:      "negative-request-param-index.yml",
			expectedError: ErrInvalidCallCriteria,
		}, {
			yamlFile:      "negative-response-param-index.yml",
			expectedError: ErrInvalidCallCriteria,
		}, {
			yamlFile:      "negative-route-handler-index.yml",
			expectedError: ErrInvalidRoute,
		}, {
			yamlFile:      "negative-route-path-index.yml",
			expectedError: ErrInvalidRoute,
		},
	}
	for i := range tests {
		t.Run(fmt.Sprintf("validation on file %s", tests[i].yamlFile), func(t *testing.T) {
			yamlFile := tests[i].yamlFile
			expectedError := tests[i].expectedError
			expectedCriteria := tests[i].expectedCriteria
			yamlFilePath := path.Join("testdata", "criterias", yamlFile)
			f, err := os.Open(yamlFilePath)
			if err != nil {
				t.Fatalf("error opening file %s: %v", yamlFilePath, err)
			}
			defer f.Close()
			criteriaDecoder := NewCriteriaDecoder(log.New(ioutil.Discard, "", log.LstdFlags))
			criteria := Criteria{}
			err = criteriaDecoder.ParseCriteriaFromYAML(f, &criteria)
			if err != expectedError {
				t.Fatalf("expected error %v but got %v", expectedError, err)
			}
			if err == nil {
				compareCriterias(t, criteria, expectedCriteria)
			}
		})
	}
}

func compareCriterias(t *testing.T, criteria Criteria, expected Criteria) {
	routesLen := len(criteria.Routes)
	expectedRoutesLen := len(expected.Routes)
	if routesLen != expectedRoutesLen {
		t.Fatalf("expected %d routes but got %d", expectedRoutesLen, routesLen)
	}
	requestLen := len(criteria.Request)
	expectedRequestLen := len(expected.Request)
	if requestLen != expectedRequestLen {
		t.Fatalf("expected %d routes but got %d", expectedRequestLen, requestLen)
	}
	responseLen := len(criteria.Response)
	expectedResponseLen := len(expected.Response)
	if responseLen != expectedResponseLen {
		t.Fatalf("expected %d routes but got %d", expectedResponseLen, responseLen)
	}
	compareRoutes(t, criteria.Routes, expected.Routes)
	compareCallCriterias(t, criteria.Request, expected.Request)
	compareCallCriterias(t, criteria.Response, expected.Response)
}

func compareRoutes(t *testing.T, routes, expected []RouteCriteria) {
	for i := range routes {
		compareRoute(t, routes[i], expected[i])
	}
}
func compareCallCriterias(t *testing.T, calls, expected []CallCriteria) {
	for i := range calls {
		compareCallCriteria(t, calls[i], expected[i])
	}
}

func compareRoute(t *testing.T, route, expected RouteCriteria) {
	if route.Pkg != expected.Pkg {
		t.Fatalf("expected Pkg %s but got %s", expected.Pkg, route.Pkg)
	}
	if route.FuncName != expected.FuncName {
		t.Fatalf("expected FuncName %s but got %s", expected.FuncName, route.FuncName)
	}
	if route.VarName != expected.VarName {
		t.Fatalf("expected VarName %s but got %s", expected.VarName, route.VarName)
	}
	if route.HTTPMethod != expected.HTTPMethod {
		t.Fatalf("expected HTTPMethod %s but got %s", expected.HTTPMethod, route.HTTPMethod)
	}
	if route.PathIndex != expected.PathIndex {
		t.Fatalf("expected PathIndex %d but got %d", expected.PathIndex, route.PathIndex)
	}
	if route.HandlerIndex != expected.HandlerIndex {
		t.Fatalf("expected HandlerIndex %d but got %d", expected.HandlerIndex, route.HandlerIndex)
	}
}

func compareCallCriteria(t *testing.T, call, expected CallCriteria) {
	if call.Pkg != expected.Pkg {
		t.Fatalf("expected Pkg %s but got %s", expected.Pkg, call.Pkg)
	}
	if call.FuncName != expected.FuncName {
		t.Fatalf("expected FuncName %s but got %s", expected.FuncName, call.FuncName)
	}
	if call.VarName != expected.VarName {
		t.Fatalf("expected VarName %s but got %s", expected.VarName, call.VarName)
	}
	if call.ParamIndex != expected.ParamIndex {
		t.Fatalf("expected ParamIndex %d but got %d", expected.ParamIndex, call.ParamIndex)
	}
}
