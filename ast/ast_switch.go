package ast

import (
	"go/ast"

	"github.com/javiercbk/swago/criteria"
)

func searchForHTTPMethodSwitch(rootNode ast.Node, switchVarType string) []switchRouterHandler {
	handlersFound := make([]switchRouterHandler, 0)
	done := false
	inspector := func(n ast.Node) bool {
		if n != nil {
			switch x := n.(type) {
			case *ast.SwitchStmt:
				selectorExpr, ok := x.Tag.(*ast.SelectorExpr)
				if ok {
					if isHTTPMethodSelectorSwitch(selectorExpr, switchVarType) {
						handlersFound = extractHTTPMethodsFromSwitch(x)
						done = true
					}
				} else {
					ident, ok := x.Tag.(*ast.Ident)
					if ok {
						v := Variable{}
						extractVariable(ident, &v)
						if v.Hierarchy == switchVarType {
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
					v := Variable{}
					extractVariable(x, &v)
					matched := criteria.MatchHTTPMethod(v.Name)
					if len(matched) > 0 {
						httpMethodsHandled = append(httpMethodsHandled, switchRouterHandler{
							HTTPMethod: matched,
							RootNode:   caseClause,
						})
					}
				case *ast.Ident:
					v := Variable{}
					extractVariable(x, &v)
					matched := criteria.MatchHTTPMethod(v.Name)
					if len(matched) > 0 {
						httpMethodsHandled = append(httpMethodsHandled, switchRouterHandler{
							HTTPMethod: matched,
							RootNode:   caseClause,
						})
					}
				}
			}
		}
	}
	return httpMethodsHandled
}

func isHTTPMethodSelectorSwitch(selectorExpr *ast.SelectorExpr, switchVarType string) bool {
	if selectorExpr.Sel.Name == selMethod {
		ident, ok := selectorExpr.X.(*ast.Ident)
		if ok {
			v := Variable{}
			extractVariable(ident, &v)
			if v.Hierarchy == switchVarType {
				return true
			}
		}
	}
	return false
}
