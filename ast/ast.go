package ast

import (
	"bytes"
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

// Variable has all the information about a variable
type Variable struct {
	Name  string
	Type  string
	Value string
}

// Function is a function
type Function struct {
	MemberOf string
	Name     string
	Pkg      string
	Args     []Variable
	Return   []Variable
	block    *ast.BlockStmt
}

// IsAnalyzable returns true if the FuncDecl is ready to be analyzed
func (fd Function) IsAnalyzable() bool {
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
	Path          Variable
	Handler       Function
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
	Type     string
	IsStruct bool
	Tag      string
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
	FindValue(filePath string, id *Variable) error
	FindFuncDeclaration(filePath string, funcDecl *Function) error
	// FindCallsInFunc(funcDecl FuncDecl) []FuncCall
	FindCallCriteria(funcDecl Function, c criteria.CallCriteria, paramIdentifier *Variable) error
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
	f, err := m.astForFile(filePath, fset)
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
	f, err := m.astForFile(filePath, fset)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return routes, err
	}
	searchFileForRouteCriteria(filePath, fset, f, routeCriterias)
	return routes, nil
}

func (m naiveManager) FindValue(filePath string, v *Variable) error {
	fset := token.NewFileSet()
	f, err := m.astForFile(filePath, fset)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return err
	}
	for i := range f.Decls {
		genDecl, ok := f.Decls[i].(*ast.GenDecl)
		if ok && len(genDecl.Specs) > 0 {
			valSpecs, ok := genDecl.Specs[0].(*ast.ValueSpec)
			if ok && len(valSpecs.Names) > 0 && len(valSpecs.Values) > 0 {
				if valSpecs.Names[0].Name == v.Name {
					vVal := Variable{}
					extractVariable(valSpecs.Values[0], &vVal)
					if len(vVal.Value) > 0 {
						v.Value = vVal.Value
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

func (m naiveManager) FindFuncDeclaration(filePath string, decl *Function) error {
	fset := token.NewFileSet()
	f, err := m.astForFile(filePath, fset)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return err
	}
	err = ErrNotFound
	for _, d := range f.Decls {
		funcDecl, ok := d.(*ast.FuncDecl)
		if ok {
			fdecl := Function{}
			extractFunction(funcDecl.Recv.List[0].Type, &fdecl)
			if fdecl.Name == decl.Name && fdecl.MemberOf == decl.MemberOf {
				// TODO: think if it should check the arguments and the return type
				decl.block = funcDecl.Body
				err = nil
				break
			}
		}
	}
	return err
}

func (m naiveManager) FindCallCriteria(funcDecl Function, c criteria.CallCriteria, paramVar *Variable) error {
	var paramExpr ast.Expr
	ast.Inspect(funcDecl.block, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if ok {
			fDecl := Function{}
			extractFunction(callExpr, &fDecl)
			if fDecl.Name == c.FuncName && (len(c.VarType) > 0 && fDecl.MemberOf == c.VarType) || fDecl.Pkg == c.Pkg {
				if len(callExpr.Args) > c.ParamIndex {
					paramExpr = callExpr.Args[c.ParamIndex]
				}
			}
		}
		return paramExpr == nil
	})
	if paramExpr != nil {
		extractVariable(paramExpr, paramVar)
		if len(paramVar.Name) == 0 {
			return ErrNotFound
		}
	} else {
		return ErrNotFound
	}
	return nil
}

func (m naiveManager) FindStruct(filePath string, s *Struct) error {
	fset := token.NewFileSet()
	f, err := m.astForFile(filePath, fset)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return err
	}
	err = ErrNotFound
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if ok {
			st, ok := ts.Type.(*ast.StructType)
			if ok {
				if ts.Name.Name == s.Name {
					for _, f := range st.Fields.List {
						buf := bytes.Buffer{}
						extractType(f.Type, &buf)
						newField := Field{
							Tag:  f.Tag.Value,
							Name: f.Names[0].Name,
							Type: correctType(buf.String()),
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

func (m naiveManager) astForFile(filePath string, fset *token.FileSet) (*ast.File, error) {
	m.logger.Printf("parsing ast from file %s\n", filePath)
	return astForFile(filePath, fset)
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

// searchFileForRouteCriteria searches a file for routes matching some criterias
func searchFileForRouteCriteria(filePath string, fset *token.FileSet, file *ast.File, criterias []criteria.RouteCriteria) []Route {
	routesFound := make([]Route, 0)
	ast.Inspect(file, func(n ast.Node) bool {
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
	})
	return routesFound
}

func matchesRouteCriteria(callExpr *ast.CallExpr, routeCriteria criteria.RouteCriteria) bool {
	id := Function{}
	extractFunction(callExpr, &id)
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
	fdecl := Function{}
	extractFunction(callExpr, &fdecl)
	if len(routeCriteria.HTTPMethod) > 0 {
		route.HTTPMethod = routeCriteria.HTTPMethod
	} else {
		route.HTTPMethod = criteria.MatchHTTPMethod(fdecl.Name)
	}
	extractVariable(callExpr.Args[routeCriteria.PathIndex], &route.Path)
	extractFunction(callExpr.Args[routeCriteria.HandlerIndex], &route.Handler)
	route.FuncCall = FuncCall{
		callExpr: callExpr,
	}
}

func extractType(n ast.Node, buf *bytes.Buffer) {
	switch x := n.(type) {
	case *ast.StarExpr:
		extractType(x.X, buf)
		buf.WriteString("*")
	case *ast.SelectorExpr:
		extractType(x.X, buf)
		buf.WriteString(".")
		buf.WriteString(x.Sel.Name)
	case *ast.Field:
		extractType(x.Type, buf)
	case *ast.Ident:
		if buf.Len() > 0 {
			buf.WriteString(".")
		}
		if x.Obj != nil {
			f, ok := x.Obj.Decl.(*ast.Field)
			if ok {
				extractType(f, buf)
			}
		} else {
			buf.WriteString(x.Name)
		}
	}
}

func correctType(toCorrect string) string {
	corrected := toCorrect
	if strings.Contains(toCorrect, "*") {
		parts := strings.Split(toCorrect, ".")
		for i := range parts {
			str := parts[i]
			if strings.HasSuffix(str, "*") {
				parts[i] = "*" + str[:len(str)-1]
			}
		}
		corrected = strings.Join(parts, ".")
	}
	return corrected
}

func extractVariable(n ast.Node, v *Variable) {
	switch x := n.(type) {
	case *ast.Field:
		v.Name = x.Names[0].Name
		buf := bytes.Buffer{}
		extractType(x.Type, &buf)
		v.Type = correctType(buf.String())
	case *ast.AssignStmt:
		ident, ok := x.Lhs[0].(*ast.Ident)
		if ok {
			v.Name = ident.Name
			buf := bytes.Buffer{}
			extractType(x.Rhs[0], &buf)
			v.Type = correctType(buf.String())
		}
	case *ast.Ident:
		v.Name = x.Name
	case *ast.SelectorExpr:
		v.Name = x.Sel.Name
		buf := bytes.Buffer{}
		extractType(x.X, &buf)
		v.Type = correctType(buf.String())
	case *ast.CompositeLit:
		extractVariable(x.Type, v)
	case *ast.BasicLit:
		val := x.Value
		if x.Kind == token.STRING {
			v.Type = "string"
			val = strings.Trim(val, "\"")
		} else {
			v.Type = "number"
		}
		v.Value = val
	}
}

func extractFunction(n ast.Node, funcDecl *Function) {
	switch x := n.(type) {
	case *ast.FuncDecl:
		funcDecl.Name = x.Name.Name
		if x.Recv != nil && x.Recv.List != nil {
			if len(x.Recv.List) > 0 {
				field := x.Recv.List[0]
				switch ft := field.Type.(type) {
				case *ast.Ident:
					funcDecl.MemberOf = ft.Name
				case *ast.StarExpr:
					ident, ok := ft.X.(*ast.Ident)
					if ok {
						funcDecl.MemberOf = "*" + ident.Name
					}
				}
			}
		}
	case *ast.CallExpr:
		extractFunction(x.Fun, funcDecl)
	}
}

func findFuncDeclInFile(f *ast.File, name, memberOf string, funcDecl *ast.FuncDecl) bool {
	for i := range f.Decls {
		decl, ok := f.Decls[i].(*ast.FuncDecl)
		if ok {
			f := Function{}
			extractFunction(decl, &f)
			if f.Name == name && f.MemberOf == memberOf {
				funcDecl = decl
				return true
			}
		}
	}
	return false
}
