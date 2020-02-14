package swago

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
)

const (
	selMethod  = "Method"
	selRequest = "Request"
	selHTTP    = "http"
)

type inspectorFunc = func(ast.Node) bool

// type nodeMatcher = func(ast.Node) ast.Node
type inspectorBuilderFunc func(*token.FileSet) inspectorFunc

type switchRouterHandler struct {
	HTTPMethod string
	RootNode   ast.Node
}

type identifier struct {
	ptr  bool
	name string
	pkg  string
}

// searchFileForRouteCriteria searches a file for routes matching some criterias
func searchFileForRouteCriteria(filePath string, criterias []RouteCriteria) ([]Route, error) {
	routesFound := make([]Route, 0)
	inspectorBuilder := func(fset *token.FileSet) inspectorFunc {
		return func(n ast.Node) bool {
			if n != nil {
				switch x := n.(type) {
				case *ast.CallExpr:
					for i := range criterias {
						routeCriteria := criterias[i]
						if matchesRouteCriteria(x, routeCriteria) {
							foundRoute := Route{}
							callExprToRoute(fset, x, &foundRoute)
							routesFound = append(routesFound, foundRoute)
							// if CallExpr matched one criteria, we don't want to compare it to other criterias
							break
						}
					}
				}
			}
			return true
		}
	}
	err := analyzeFile(filePath, inspectorBuilder)
	return routesFound, err
}

// analyzeFile scans a file for patterns
func analyzeFile(filePath string, inspectorBuilder inspectorBuilderFunc) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return analyzeReader(filePath, f, inspectorBuilder)
}

// analyzeReader scans a reader for patterns
func analyzeReader(filePath string, r io.Reader, inspectorBuilder inspectorBuilderFunc) error {
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	fset := token.NewFileSet() // positions are relative to fset
	parsed, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return err
	}
	for i := range parsed.Decls {
		d := parsed.Decls[i]
		fmt.Printf("%s\t%v\n", fset.Position(d.Pos()), d)
	}
	inspector := inspectorBuilder(fset)
	ast.Inspect(parsed, inspector)
	return nil
}

func matchesRouteCriteria(callExpr *ast.CallExpr, criteria RouteCriteria) bool {
	return false
}

func callExprToRoute(fset *token.FileSet, callExpr *ast.CallExpr, route *Route) {

}

// func findChildASTNode(n ast.Node, retrieveChildNode bool, matcher nodeMatcher) ast.Node {
// 	var astNodeFound ast.Node
// 	ast.Inspect(n, func(inspecting ast.Node) bool {
// 		nodeMatched := matcher(inspecting)
// 		if nodeMatched != nil {
// 			if retrieveChildNode {
// 				astNodeFound = nodeMatched
// 			} else {
// 				astNodeFound = inspecting
// 			}
// 		}
// 		return astNodeFound == nil
// 	})
// 	return astNodeFound
// }

func findTypeAndPkg(ident ast.Node, id *identifier) {
	switch x := ident.(type) {
	case *ast.Ident:
		if x.Obj != nil {
			if x.Obj.Decl != nil {
				switch identX := x.Obj.Decl.(type) {
				case *ast.Field:
					findTypeAndPkg(identX, id)
				case *ast.ValueSpec:
					if len(identX.Values) > 0 {
						// for the moment I only care about the first one
						findTypeAndPkg(identX.Values[0], id)
					}
				case *ast.AssignStmt:
					findTypeAndPkg(identX.Rhs[0], id)
				}
				field, ok := x.Obj.Decl.(*ast.Field)
				if ok {
					findTypeAndPkg(field, id)
				}
			}
		} else {
			id.pkg = x.Name
		}
	case *ast.Field:
		findTypeAndPkg(x.Type, id)
	case *ast.SelectorExpr:
		if x.Sel.Obj == nil {
			id.name = x.Sel.Name
			findTypeAndPkg(x.X, id)
		}
	case *ast.StarExpr:
		id.ptr = true
		findTypeAndPkg(x.X, id)
	}
}

// func analyzeFuncDecl(fset *token.FileSet, decl *ast.FuncDecl) {
// 	x := decl.Type
// 	if len(x.Params.List) == 2 {
// 		params := x.Params.List
// 		for paramI := range x.Params.List {
// 			p, ok := params[paramI].Type.(*ast.SelectorExpr)
// 			if ok {
// 				id, ok := p.X.(*ast.Ident)
// 				if ok && id.Name == "http" && p.Sel.Name == "ResponseWriter" {
// 					fmt.Printf("%s=>%s\t%v\n", fset.Position(decl.Pos()), decl.Name, params[paramI].Type)
// 				}
// 			}
// 		}
// 	}
// }

// func analyzeFuncLit(fset *token.FileSet, lit *ast.FuncLit) {
// 	x := lit.Type
// 	if len(x.Params.List) == 2 {
// 		params := x.Params.List
// 		for paramI := range x.Params.List {
// 			p, ok := params[paramI].Type.(*ast.SelectorExpr)
// 			if ok {
// 				id, ok := p.X.(*ast.Ident)
// 				if ok && id.Name == "http" && p.Sel.Name == "ResponseWriter" {
// 					fmt.Printf("%s=>%s\t%v\n", fset.Position(lit.Pos()), "FuncLit", params[paramI].Type)
// 				}
// 			}
// 		}
// 	}
// }

// func analyzeCallExpr(fset *token.FileSet, callExp *ast.CallExpr) {
// 	fmt.Printf("%s=>\t%v\n", fset.Position(callExp.Pos()), callExp)
// 	selectorExpression, ok := callExp.Fun.(*ast.SelectorExpr)
// 	if ok {
// 		httpMethod := selectorExpression.Sel.Name
// 		fmt.Printf("httpMethod %s\n", httpMethod)
// 		ast.Inspect(selectorExpression.X, func(n ast.Node) bool {
// 			switch x := n.(type) {
// 			case *ast.FuncDecl:
// 				analyzeFuncDecl(fset, x)
// 			case *ast.FuncLit:
// 				analyzeFuncLit(fset, x)
// 			case *ast.CallExpr:
// 				analyzeCallExpr(fset, x)
// 			}
// 			return true
// 		})
// 		ident, ok := selectorExpression.X.(*ast.Ident)
// 		if ok {
// 			field, _ := ident.Obj.Decl.(*ast.Field)
// 			fmt.Printf("field: %v\n", field)
// 			// if ok {
// 			// starExpr, ok := field.Type.(*ast.StarExpr)
// 			/*
// 								a:<*go/ast.SelectorExpr>(0xc00000c1e0)
// 				:<go/ast.SelectorExpr>
// 				X:<go/ast.Expr>
// 				Sel:<*go/ast.Ident>(0xc00000c1c0)
// 				:<go/ast.Ident>
// 				NamePos:84
// 				Name:"Group"
// 				Obj:nil <*go/ast.Object>
// 				data:<*go/ast.Ident>(0xc00000c1a0)
// 				:<go/ast.Ident>
// 				NamePos:79
// 				Name:"echo"
// 				Obj:nil <*go/ast.Object>

// 			*/
// 			// if ok {
// 			// 	starExpr.X
// 			// } else {

// 			// }
// 			// }
// 		}
// 	}
// }
