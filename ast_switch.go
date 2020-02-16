package swago

import (
	"go/ast"
	"strings"
)

func searchForHTTPMethodSwitch(rootNode ast.Node, pkg string) []switchRouterHandler {
	handlersFound := make([]switchRouterHandler, 0)
	done := false
	inspector := func(n ast.Node) bool {
		if n != nil {
			switch x := n.(type) {
			case *ast.SwitchStmt:
				selectorExpr, ok := x.Tag.(*ast.SelectorExpr)
				if ok {
					if isHTTPMethodSelectorSwitch(selectorExpr, pkg) {
						handlersFound = extractHTTPMethodsFromSwitch(x)
						done = true
					}
				} else {
					ident, ok := x.Tag.(*ast.Ident)
					if ok {
						id := identifier{
							name: ident.Name,
						}
						findTypeAndPkg(ident, &id)
						if id.pkg == pkg && id.name == selRequest {
							handlersFound = extractHTTPMethodsFromSwitch(x)
							done = true
						}
						// we need to check if the identifier is an http.Request
					}
				}
			}
		}
		return !done
	}
	ast.Inspect(rootNode, inspector)
	return handlersFound
}

func extractHTTPMethodsFromSwitch(switchStmt *ast.SwitchStmt) []switchRouterHandler {
	httpMethodsHandled := make([]switchRouterHandler, 0, 2)
	for _, c := range switchStmt.Body.List {
		caseClause, ok := c.(*ast.CaseClause)
		if ok {
			for _, l := range caseClause.List {
				switch x := l.(type) {
				case *ast.SelectorExpr:
					id := identifier{}
					findTypeAndPkg(x, &id)
					httpMethodName := strings.ToUpper(id.name)
					for i := range httpMethods {
						if strings.Contains(httpMethodName, httpMethods[i]) {
							httpMethodsHandled = append(httpMethodsHandled, switchRouterHandler{
								HTTPMethod: httpMethods[i],
								RootNode:   caseClause,
							})
						}
					}
				case *ast.Ident:
					id := identifier{}
					findTypeAndPkg(x, &id)
					httpMethodName := strings.ToUpper(id.name)
					for i := range httpMethods {
						if strings.Contains(httpMethodName, httpMethods[i]) {
							httpMethodsHandled = append(httpMethodsHandled, switchRouterHandler{
								HTTPMethod: httpMethods[i],
								RootNode:   caseClause,
							})
						}
					}
				}
			}
		}
	}
	return httpMethodsHandled
}

func isHTTPMethodSelectorSwitch(selectorExpr *ast.SelectorExpr, pkg string) bool {
	if selectorExpr.Sel.Name == selMethod {
		ident, ok := selectorExpr.X.(*ast.Ident)
		if ok {
			id := identifier{
				name: ident.Name,
			}
			findTypeAndPkg(ident, &id)
			if id.pkg == pkg && id.name == selRequest {
				return true
			}
		}
	}
	return false
}
