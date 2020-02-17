package ast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/javiercbk/swago/criteria"
)

const (
	selMethod  = "Method"
	selRequest = "Request"
)

type inspectorFunc = func(ast.Node) bool

// type nodeMatcher = func(ast.Node) ast.Node
type inspectorBuilderFunc func(*token.FileSet) inspectorFunc

type switchRouterHandler struct {
	HTTPMethod string
	RootNode   ast.Node
}

type Identifier struct {
	Ptr      bool
	MemberOf string
	Name     string
	Pkg      string
	Value    string
}

func astForFile(filePath string, fset *token.FileSet) (*ast.File, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return astForReader(filePath, f, fset)
}

func astForReader(filePath string, r io.Reader, fset *token.FileSet) (*ast.File, error) {
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parser.ParseFile(fset, filePath, src, parser.ParseComments)
}

// analyzeFile scans a file for patterns
func analyzeFile(fset *token.FileSet, file *ast.File, inspectorBuilder inspectorBuilderFunc) {
	inspector := inspectorBuilder(fset)
	ast.Inspect(file, inspector)
}

// searchFileForRouteCriteria searches a file for routes matching some criterias
func searchFileForRouteCriteria(filePath string, fset *token.FileSet, file *ast.File, criterias []criteria.RouteCriteria) []Route {
	routesFound := make([]Route, 0)
	inspectorBuilder := func(fset *token.FileSet) inspectorFunc {
		return func(n ast.Node) bool {
			if n != nil {
				switch x := n.(type) {
				case *ast.CallExpr:
					for i := range criterias {
						routeCriteria := criterias[i]
						if matchesRouteCriteria(x, routeCriteria) {
							foundRoute := Route{
								File: filePath,
							}
							callExprToRoute(fset, x, routeCriteria, &foundRoute)
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
	analyzeFile(fset, file, inspectorBuilder)
	return routesFound
}

func matchesRouteCriteria(callExpr *ast.CallExpr, routeCriteria criteria.RouteCriteria) bool {
	id := Identifier{}
	identify(callExpr, &id)
	matches := id.Name == routeCriteria.FuncName && id.Pkg == routeCriteria.Pkg && id.MemberOf == routeCriteria.VarType
	if matches {
		if len(routeCriteria.HTTPMethod) == 0 && !criteria.MatchesHTTPMethod(id.Name) {
			return false
		}
		if callExpr.Args == nil {
			return false
		}
		argsLen := len(callExpr.Args)
		if routeCriteria.PathIndex >= argsLen || routeCriteria.HandlerIndex >= argsLen {
			return false
		}
	}
	return matches
}

func callExprToRoute(fset *token.FileSet, callExpr *ast.CallExpr, routeCriteria criteria.RouteCriteria, route *Route) {
	id := Identifier{}
	identify(callExpr, &id)
	if len(routeCriteria.HTTPMethod) > 0 {
		route.HTTPMethod = routeCriteria.HTTPMethod
	} else {
		route.HTTPMethod = criteria.MatchHTTPMethod(id.Name)
	}
	pathID := Identifier{}
	identify(callExpr.Args[routeCriteria.PathIndex], &pathID)
	// route.Path = pathID
	handleID := Identifier{}
	identify(callExpr.Args[routeCriteria.HandlerIndex], &handleID)
	route.HandlerPkg = handleID.Pkg
	route.HandlerName = handleID.Name
	route.FuncCall = FuncCall{
		callExpr: callExpr,
	}
}

func identify(ident ast.Node, id *Identifier) {
	switch x := ident.(type) {
	case *ast.Ident:
		if x.Obj != nil {
			if x.Obj.Decl != nil {
				switch identX := x.Obj.Decl.(type) {
				case *ast.Field:
					identify(identX, id)
				case *ast.ValueSpec:
					if len(identX.Values) > 0 {
						// for the moment I only care about the first one
						identify(identX.Values[0], id)
					}
				case *ast.AssignStmt:
					identify(identX.Rhs[0], id)
				case *ast.TypeSpec:
					id.MemberOf = identX.Name.Name
				}
				field, ok := x.Obj.Decl.(*ast.Field)
				if ok {
					identify(field, id)
				}
			}
		} else {
			if len(id.Name) == 0 {
				id.Name = x.Name
			} else {
				id.Pkg = x.Name
			}
		}
	case *ast.Field:
		identify(x.Type, id)
	case *ast.SelectorExpr:
		if x.Sel.Obj == nil {
			id.Name = x.Sel.Name
			identify(x.X, id)
		}
	case *ast.StarExpr:
		id.Ptr = true
		identify(x.X, id)
	case *ast.CompositeLit:
		identify(x.Type, id)
	case *ast.FuncDecl:
		id.Name = x.Name.Name
		if x.Recv != nil && x.Recv.List != nil {
			if len(x.Recv.List) > 0 {
				field := x.Recv.List[0]
				switch ft := field.Type.(type) {
				case *ast.Ident:
					id.MemberOf = ft.Name
				case *ast.StarExpr:
					ident, ok := ft.X.(*ast.Ident)
					if ok {
						id.MemberOf = "*" + ident.Name
					}
				}
			}
		}
	case *ast.BasicLit:
		val := x.Value
		if x.Kind == token.STRING {
			val = strings.Trim(val, "\"")
		}
		id.Value = val

	}
}

func findFuncDeclInFile(f *ast.File, name, memberOf string, funcDecl *ast.FuncDecl) bool {
	for i := range f.Decls {
		decl, ok := f.Decls[i].(*ast.FuncDecl)
		if ok {
			id := Identifier{}
			identify(decl, &id)
			if id.Name == name && id.MemberOf == memberOf {
				funcDecl = decl
				return true
			}
		}
	}
	return false
}
