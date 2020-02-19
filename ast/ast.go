package ast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/javiercbk/swago/criteria"
)

type notFoundErr string

func (i notFoundErr) Error() string {
	return string(i)
}

// FileImport represent the imports of a go file
type FileImport struct {
	Name string
	Pkg  string
}

// Identifier containts all the information about an identifier
type Identifier struct {
	Ptr      bool
	MemberOf string
	Name     string
	Pkg      string
	Value    string
}

// FuncDecl is a function declaration
type FuncDecl struct {
	Identifier
	block *ast.BlockStmt
}

// IsAnalyzable returns true if the FuncDecl is ready to be analyzed
func (fd FuncDecl) IsAnalyzable() bool {
	return fd.block != nil
}

// FuncCall is a function call
type FuncCall struct {
	callExpr *ast.CallExpr
}

// Route is a route handled found
type Route struct {
	File          string
	HTTPMethod    string
	Path          Identifier
	Handler       FuncDecl
	RequestModel  Model
	ResponseModel Model
	FuncCall      FuncCall
}

// Model is a serializable struct found that can parse incoming requests or serialize outgoing responses
type Model struct {
	File string
	Line int
	Pos  int
	Pkg  string
	Type string
}

// Field is a struct field
type Field struct {
	Name     string
	Type     Identifier
	IsStruct bool
	Tag      string
}

// VarType encodes a variable type
type VarType struct {
	GoType string
	Struct Struct
}

// Struct is a struct
type Struct struct {
	Name   string
	Fields []Field
}

const (
	// ErrNotFound is returned when a value was not found for an identifier
	ErrNotFound notFoundErr = "value not found"
	selMethod               = "Method"
	selRequest              = "Request"
)

// Manager is an abstraction that can read ast for files
type Manager interface {
	GetFileImports(filePath string) ([]FileImport, error)
	ExtractRoutesFromFile(filePath string, criterias []criteria.RouteCriteria) ([]Route, error)
	FindValue(filePath string, id *Identifier) error
	FindFuncDeclaration(filePath string, id Identifier) (FuncDecl, error)
	// FindCallsInFunc(funcDecl FuncDecl) []FuncCall
	FindCallCriteria(funcDecl FuncDecl, callCriteria []criteria.CallCriteria, paramIdentifier *Identifier) error
	FindStruct(filePath string, s *Struct) error
}

type naiveManager struct {
	logger *log.Logger
}

type inspectorFunc = func(ast.Node) bool

// type nodeMatcher = func(ast.Node) ast.Node
type inspectorBuilderFunc func(*token.FileSet) inspectorFunc

type switchRouterHandler struct {
	HTTPMethod string
	RootNode   ast.Node
}

func (m naiveManager) GetFileImports(filePath string) ([]FileImport, error) {
	var imports []FileImport
	fset := token.NewFileSet()
	f := ast.File{}
	err := m.astForFile(filePath, fset, &f)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return imports, err
	}
	imports = make([]FileImport, len(f.Imports))
	for i := range f.Imports {
		fileImport := f.Imports[i]
		fi := FileImport{
			Pkg: fileImport.Path.Value,
		}
		if fileImport.Name != nil {
			fi.Name = fileImport.Name.Name
		}
		imports[i] = fi
	}
	return imports, nil
}

func (m naiveManager) ExtractRoutesFromFile(filePath string, routeCriterias []criteria.RouteCriteria) ([]Route, error) {
	var routes []Route
	fset := token.NewFileSet()
	f := ast.File{}
	err := m.astForFile(filePath, fset, &f)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return routes, err
	}
	searchFileForRouteCriteria(filePath, fset, &f, routeCriterias)
	return routes, nil
}

func (m naiveManager) FindValue(filePath string, id *Identifier) error {
	f := ast.File{}
	fset := token.NewFileSet()
	err := m.astForFile(filePath, fset, &f)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return err
	}
	for i := range f.Decls {
		genDecl, ok := f.Decls[i].(*ast.GenDecl)
		if ok && len(genDecl.Specs) > 0 {
			valSpecs, ok := genDecl.Specs[0].(*ast.ValueSpec)
			if ok && len(valSpecs.Names) > 0 && len(valSpecs.Values) > 0 {
				if valSpecs.Names[0].Name == id.Name {
					idVal := Identifier{}
					identify(valSpecs.Values[0], &idVal)
					if len(idVal.Value) > 0 {
						id.Value = idVal.Value
						return nil
					} else {
						//TODO: handle scenario where there this a variable
					}
				}
			}
		}
	}
	return ErrNotFound
}

func (m naiveManager) FindFuncDeclaration(filePath string, targetID Identifier) (FuncDecl, error) {
	var decl FuncDecl
	fset := token.NewFileSet()
	f := ast.File{}
	err := m.astForFile(filePath, fset, &f)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return decl, err
	}
	err = ErrNotFound
	for _, d := range f.Decls {
		funcDecl, ok := d.(*ast.FuncDecl)
		if ok {
			id := Identifier{}
			identify(funcDecl.Recv.List[0].Type, &id)
			if id.Name == targetID.Name && id.MemberOf == targetID.MemberOf {
				decl.Identifier = id
				decl.block = funcDecl.Body
				err = nil
				break
			}
		}
	}
	return decl, err
}

// func (m naiveManager) FindCallsInFunc(funcDecl FuncDecl) []FuncCall {
// 	calls := make([]FuncCall, 0)
// 	if funcDecl.block != nil {
// 		ast.Inspect(funcDecl.block, func(n ast.Node) bool {
// 			if n != nil {
// 				callExpr, ok := n.(*ast.CallExpr)
// 				if ok {
// 					calls = append(calls, FuncCall{
// 						callExpr: callExpr,
// 					})
// 				}
// 			}
// 			return true
// 		})
// 	}
// 	return calls
// }

func (m naiveManager) FindCallCriteria(funcDecl FuncDecl, callCriteria []criteria.CallCriteria, paramIdentifier *Identifier) error {
	var paramExpr ast.Expr
	ast.Inspect(funcDecl.block, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if ok {
			id := Identifier{}
			identify(callExpr, &id)
			for _, c := range callCriteria {
				if id.Name == c.FuncName && (len(c.VarType) > 0 && id.MemberOf == c.VarType) || id.Pkg == c.Pkg {
					if len(callExpr.Args) > c.ParamIndex {
						paramExpr = callExpr.Args[c.ParamIndex]
					}
				}
				break
			}
		}
		return paramExpr == nil
	})
	if paramExpr != nil {
		identify(paramExpr, paramIdentifier)
		if len(paramIdentifier.Name) == 0 {
			return ErrNotFound
		}
	} else {
		return ErrNotFound
	}
	return nil
}

func (m naiveManager) FindStruct(filePath string, s *Struct) error {
	fset := token.NewFileSet()
	f := ast.File{}
	err := m.astForFile(filePath, fset, &f)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return err
	}
	err = ErrNotFound
	found := false
	ast.Inspect(&f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if ok {
			st, ok := ts.Type.(*ast.StructType)
			if ok {
				if ts.Name.Name == s.Name {
					for _, f := range st.Fields.List {
						typeIdentifier := Identifier{}
						typeID, ok := f.Type.(*ast.Ident)
						if ok {
							typeIdentifier.Name = typeID.Name
						} else {
							typeSelector, ok := f.Type.(*ast.SelectorExpr)
							if ok {
								identify(typeSelector, &typeIdentifier)
							}
						}
						newField := Field{
							Tag:  f.Tag.Value,
							Name: f.Names[0].Name,
							Type: typeIdentifier,
						}
						s.Fields = append(s.Fields, newField)
					}
					found = true
					return false
				}
			}
		}
		return !found
	})
	return err
}

func (m naiveManager) astForFile(filePath string, fset *token.FileSet, f *ast.File) error {
	var err error
	m.logger.Printf("parsing ast from file %s\n", filePath)
	f, err = astForFile(filePath, fset)
	return err
}

// NewManager creates the default Manager
func NewManager(logger *log.Logger) Manager {
	return naiveManager{
		logger: logger,
	}
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
	route.Path = pathID
	handleID := Identifier{}
	identify(callExpr.Args[routeCriteria.HandlerIndex], &handleID)
	route.Handler = FuncDecl{
		Identifier: handleID,
	}
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
