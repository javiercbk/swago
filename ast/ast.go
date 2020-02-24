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
	Name       string
	Hierarchy  string
	GoType     string
	Value      string
	Definition *Variable
}

// Function is a function
type Function struct {
	File      string
	Hierarchy string
	Name      string
	Args      []Variable
	Return    []Variable
	fAST      fileAST
	block     *ast.BlockStmt
	callExpr  *ast.CallExpr
}

// FuncCall is a function call
type FuncCall struct {
	Function Function
	Args     []Variable
	Return   []Variable
	callExpr *ast.CallExpr
}

// Route is a route handled found
type Route struct {
	File          string
	HTTPMethod    string
	Path          Variable
	Middlewares   []string
	ChildRoutes   []Route
	Handler       Function
	RequestModel  Model
	ResponseModel Model
	FuncCall      *FuncCall
	Struct        *Struct
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
	Name string
	Type string
	Tag  string
}

// StructDef is a struct definition
type StructDef struct {
	Name   string
	Fields []Field
}

// Struct is a struct
type Struct struct {
	s      *StructDef
	Values map[string]Variable
}

const (
	// ErrNotFound is returned when a value was not found for an identifier
	ErrNotFound      notFoundErr = "value not found"
	selMethod                    = "Method"
	selRequest                   = "Request"
	goTypeBool                   = "bool"
	goTypeString                 = "string"
	goTypeInt                    = "int"
	goTypeInt8                   = "int8"
	goTypeInt16                  = "int16"
	goTypeInt32                  = "int32"
	goTypeInt64                  = "int64"
	goTypeUint                   = "uint"
	goTypeUint8                  = "uint8"
	goTypeUint16                 = "uint16"
	goTypeUint32                 = "uint32"
	goTypeUint64                 = "uint64"
	goTypeUintptr                = "uintptr"
	goTypeByte                   = "byte"
	goTypeRune                   = "rune"
	goTypeFloat32                = "float32"
	goTypeFloat64                = "float64"
	goTypeComplex64              = "complex64"
	goTypeComplex128             = "complex128"
)

// Manager is an abstraction that can read ast for files
type Manager interface {
	GetFileImports(filePath string) ([]FileImport, error)
	ExtractRoutes(f *Function, routeCriterias []criteria.RouteCriteria) []Route
	FindStructLiteral(filePath string, structRoute *criteria.StructRoute) ([]Route, error)
	ExtractFuncCalls(f Function) []Function
	FindValue(filePath string, id *Variable) error
	FindFuncDeclaration(filePath string, funcDecl *Function) error
	// FindCallsInFunc(funcDecl FuncDecl) []FuncCall
	FindCallCriteria(funcDecl Function, c criteria.CallCriteria, paramIdentifier *Variable) error
	FindStruct(filePath string, s *StructDef) error
}

type fileAST struct {
	file *ast.File
	fset *token.FileSet
}

type cacheManager struct {
	files  map[string]fileAST
	logger *log.Logger
}

type switchRouterHandler struct {
	HTTPMethod string
	RootNode   ast.Node
}

func (m *cacheManager) GetFileImports(filePath string) ([]FileImport, error) {
	var imports []FileImport
	fAST, err := m.astForFile(filePath)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return imports, err
	}
	imports = make([]FileImport, len(fAST.file.Imports))
	for i := range fAST.file.Imports {
		fileImport := fAST.file.Imports[i]
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

func (m *cacheManager) ExtractRoutes(f *Function, routeCriterias []criteria.RouteCriteria) []Route {
	routesFound := make([]Route, 0)
	if f.block != nil {
		ast.Inspect(f.block, func(n ast.Node) bool {
			if n != nil {
				switch x := n.(type) {
				case *ast.CallExpr:
					for _, c := range routeCriterias {
						if c.FuncRoute != nil {
							if matchesRouteCriteria(x, c) {
								foundRoute := Route{
									File: f.File,
								}
								callExprToRoute(f.fAST.fset, x, c, &foundRoute)
								routesFound = append(routesFound, foundRoute)
								// if CallExpr matched one criteria, we don't want to compare it to other criterias
								break
							}
						}
					}
				}
			}
			return true
		})
	}
	return routesFound
}

func (m *cacheManager) ExtractFuncCalls(f Function) []Function {
	functions := make([]Function, 0)
	ast.Inspect(f.block, func(n ast.Node) bool {
		if n != nil {
			callExpr, ok := n.(*ast.CallExpr)
			if ok {
				function := Function{}
				extractFunction(callExpr, &function)
				functions = append(functions, function)
			}
		}
		return true
	})
	return functions
}

func (m *cacheManager) FindValue(filePath string, v *Variable) error {
	fAST, err := m.astForFile(filePath)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return err
	}
	for _, d := range fAST.file.Decls {
		genDecl, ok := d.(*ast.GenDecl)
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

func (m *cacheManager) FindFuncDeclaration(filePath string, decl *Function) error {
	fAST, err := m.astForFile(filePath)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return err
	}
	err = ErrNotFound
	for _, d := range fAST.file.Decls {
		funcDecl, ok := d.(*ast.FuncDecl)
		if ok {
			fdecl := Function{
				File: filePath,
				fAST: fAST,
			}
			extractFunction(d, &fdecl)
			if fdecl.Name == decl.Name && fdecl.Hierarchy == decl.Hierarchy {
				// TODO: think if it should check the arguments and the return type
				decl.block = funcDecl.Body
				err = nil
				break
			}
		}
	}
	return err
}

func (m *cacheManager) FindCallCriteria(funcDecl Function, c criteria.CallCriteria, paramVar *Variable) error {
	var paramExpr ast.Expr
	ast.Inspect(funcDecl.block, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if ok {
			fDecl := Function{}
			extractFunction(callExpr, &fDecl)
			if fDecl.Name == c.FuncName && fDecl.Hierarchy == c.Hierarchy {
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

func (m *cacheManager) FindStructLiteral(filePath string, structRoute *criteria.StructRoute) ([]Route, error) {
	routes := make([]Route, 0)
	fAST, err := m.astForFile(filePath)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return routes, err
	}
	ast.Inspect(fAST.file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CompositeLit:
			if structRoute != nil {
				if matchesStructRoute(x, *structRoute) {
					foundRoute := Route{
						File: filePath,
					}
					compositeLitToRoute(fAST.fset, x, &foundRoute)
					routes = append(routes, foundRoute)
				}
			}
		}
		return true
	})
	return routes, nil
}

func (m *cacheManager) FindStruct(filePath string, s *StructDef) error {
	fAST, err := m.astForFile(filePath)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return err
	}
	err = ErrNotFound
	found := false
	ast.Inspect(fAST.file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if ok {
			st, ok := ts.Type.(*ast.StructType)
			if ok {
				if ts.Name.Name == s.Name {
					for _, f := range st.Fields.List {
						buf := bytes.Buffer{}
						extractHierarchy(f.Type, &buf)
						typeStr := correctHierarchy(buf.String())
						tag := ""
						if f.Tag != nil {
							tag = strings.Trim(f.Tag.Value, "`")
						}
						newField := Field{
							Tag:  tag,
							Name: f.Names[0].Name,
							Type: typeStr,
						}
						s.Fields = append(s.Fields, newField)
					}
					found = true
					err = nil
					return false
				}
			}
		}
		return !found
	})
	return err
}

func (m *cacheManager) astForFile(filePath string) (fileAST, error) {
	m.logger.Printf("parsing ast from file %s\n", filePath)
	f, ok := m.files[filePath]
	if ok {
		return f, nil
	}
	var err error
	f = fileAST{
		fset: token.NewFileSet(),
	}
	f.file, err = astForFile(filePath, f.fset)
	if err != nil {
		return f, err
	}
	m.files[filePath] = f
	return f, nil
}

// NewManager creates the default Manager
func NewManager(logger *log.Logger) Manager {
	return &cacheManager{
		logger: logger,
		files:  make(map[string]fileAST),
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

func matchesRouteCriteria(callExpr *ast.CallExpr, routeCriteria criteria.RouteCriteria) bool {
	id := Function{}
	extractFunction(callExpr, &id)
	matches := id.Name == routeCriteria.FuncRoute.FuncName && id.Hierarchy == routeCriteria.FuncRoute.Hierarchy
	if matches {
		if len(routeCriteria.FuncRoute.HTTPMethod) == 0 && !criteria.MatchesHTTPMethod(id.Name) {
			return false
		}
		if callExpr.Args == nil {
			return false
		}
		argsLen := len(callExpr.Args)
		if routeCriteria.FuncRoute.PathIndex >= argsLen || routeCriteria.FuncRoute.HandlerIndex >= argsLen {
			return false
		}
	}
	return matches
}

func matchesStructRoute(com *ast.CompositeLit, structRoute criteria.StructRoute) bool {
	v := Variable{}
	extractVariable(com.Type, &v)
	return structRoute.Name == v.Name && v.Hierarchy == structRoute.Hierarchy
}

func callExprToRoute(fset *token.FileSet, callExpr *ast.CallExpr, routeCriteria criteria.RouteCriteria, route *Route) {
	fdecl := Function{}
	extractFunction(callExpr, &fdecl)
	if len(routeCriteria.FuncRoute.HTTPMethod) > 0 {
		route.HTTPMethod = routeCriteria.FuncRoute.HTTPMethod
	} else {
		route.HTTPMethod = criteria.MatchHTTPMethod(fdecl.Name)
	}
	extractVariable(callExpr.Args[routeCriteria.FuncRoute.PathIndex], &route.Path)
	extractFunction(callExpr.Args[routeCriteria.FuncRoute.HandlerIndex], &route.Handler)
	route.FuncCall = &FuncCall{
		callExpr: callExpr,
	}
}

func compositeLitToRoute(fset *token.FileSet, com *ast.CompositeLit, route *Route) {
	v := Variable{}
	extractVariable(com.Type, &v)
	route.Struct = &Struct{
		Values: make(map[string]Variable),
	}
	for _, e := range com.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if ok {
			ident, ok := kv.Key.(*ast.Ident)
			if ok {
				v := Variable{}
				extractVariable(kv.Value, &v)
				route.Struct.Values[ident.Name] = v
			}
		}
	}
}

func extractHierarchy(n ast.Node, buf *bytes.Buffer) {
	switch x := n.(type) {
	case *ast.StarExpr:
		switch stX := x.X.(type) {
		case *ast.Ident:
			buf.WriteString("*")
			buf.WriteString(stX.Name)
		case *ast.SelectorExpr:
			extractHierarchy(stX.X, buf)
			buf.WriteString(".")
			buf.WriteString("*")
			buf.WriteString(stX.Sel.Name)
		}
	case *ast.SelectorExpr:
		extractHierarchy(x.X, buf)
		buf.WriteString(".")
		buf.WriteString(x.Sel.Name)
	case *ast.Field:
		extractHierarchy(x.Type, buf)
	case *ast.ArrayType:
		buf.WriteString("[]")
		extractHierarchy(x.Elt, buf)
	case *ast.MapType:
		buf.WriteString("map[")
		extractHierarchy(x.Key, buf)
		buf.WriteString("]")
		extractHierarchy(x.Value, buf)
	case *ast.Ident:
		if buf.Len() > 0 && !strings.HasSuffix(buf.String(), "[]") {
			buf.WriteString(".")
		}
		if x.Obj != nil {
			switch f := x.Obj.Decl.(type) {
			case *ast.Field:
				extractHierarchy(f, buf)
			case *ast.AssignStmt:
				extractHierarchy(f.Rhs[0], buf)
			case *ast.TypeSpec:
				buf.WriteString(f.Name.Name)
			}
		} else {
			buf.WriteString(x.Name)
		}
	case *ast.CompositeLit:
		extractHierarchy(x.Type, buf)
	}
}

func oldExtractHierarchy(n ast.Node, buf *bytes.Buffer) {
	switch x := n.(type) {
	case *ast.StarExpr:
		extractHierarchy(x.X, buf)
		buf.WriteString("*")
	case *ast.SelectorExpr:
		extractHierarchy(x.X, buf)
		buf.WriteString(".")
		buf.WriteString(x.Sel.Name)
	case *ast.Field:
		extractHierarchy(x.Type, buf)
	case *ast.ArrayType:
		extractHierarchy(x.Elt, buf)
		buf.WriteString("[]")
	case *ast.Ident:
		if buf.Len() > 0 {
			buf.WriteString(".")
		}
		if x.Obj != nil {
			switch f := x.Obj.Decl.(type) {
			case *ast.Field:
				extractHierarchy(f, buf)
			case *ast.AssignStmt:
				extractHierarchy(f.Rhs[0], buf)
			case *ast.TypeSpec:
				buf.WriteString(f.Name.Name)
			}
		} else {
			buf.WriteString(x.Name)
		}
	case *ast.CompositeLit:
		extractHierarchy(x.Type, buf)
	}
}

func correctHierarchy(toCorrect string) string {
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
		ident, ok := x.Type.(*ast.Ident)
		if ok {
			if isGoType(ident.Name) {
				v.GoType = ident.Name
			} else {
				v.Hierarchy = ident.Name
			}
		} else {
			buf := bytes.Buffer{}
			extractHierarchy(x.Type, &buf)
			v.Hierarchy = correctHierarchy(buf.String())
		}
	case *ast.AssignStmt:
		ident, ok := x.Lhs[0].(*ast.Ident)
		if ok {
			v.Name = ident.Name
			buf := bytes.Buffer{}
			extractHierarchy(x.Rhs[0], &buf)
			v.Hierarchy = correctHierarchy(buf.String())
		}
	case *ast.Ident:
		if isGoType(x.Name) {
			v.GoType = x.Name
		} else {
			v.Name = x.Name
		}
		if x.Obj != nil {
			v.Definition = &Variable{}
			switch o := x.Obj.Decl.(type) {
			case *ast.AssignStmt:
				extractVariable(o.Rhs[0], v.Definition)
			case *ast.ValueSpec:
				extractVariable(o.Values[0], v.Definition)
			case *ast.Field:
				extractVariable(o.Type, v.Definition)
			}
		}
	case *ast.SelectorExpr:
		v.Name = x.Sel.Name
		buf := bytes.Buffer{}
		extractHierarchy(x.X, &buf)
		v.Hierarchy = correctHierarchy(buf.String())
	case *ast.CompositeLit:
		extractVariable(x.Type, v)
	case *ast.BasicLit:
		val := x.Value
		if x.Kind == token.STRING {
			v.GoType = "string"
			val = strings.Trim(val, "\"")
		} else {
			v.GoType = "number"
		}
		v.Value = val
	}
}

func extractOriginalDefinition(v Variable) Variable {
	if v.Definition != nil {
		return extractOriginalDefinition(*v.Definition)
	}
	return v
}

func isGoType(t string) bool {
	switch t {
	case goTypeBool:
		return true
	case goTypeString:
		return true
	case goTypeInt:
		return true
	case goTypeInt8:
		return true
	case goTypeInt16:
		return true
	case goTypeInt32:
		return true
	case goTypeInt64:
		return true
	case goTypeUint:
		return true
	case goTypeUint8:
		return true
	case goTypeUint16:
		return true
	case goTypeUint32:
		return true
	case goTypeUint64:
		return true
	case goTypeUintptr:
		return true
	case goTypeByte:
		return true
	case goTypeRune:
		return true
	case goTypeFloat32:
		return true
	case goTypeFloat64:
		return true
	case goTypeComplex64:
		return true
	case goTypeComplex128:
		return true
	default:
		return false
	}
}

func extractFuncCall(n ast.Node, funcCall *FuncCall) {
	switch x := n.(type) {
	case *ast.AssignStmt:
		extractFunction(x.Rhs[0], &funcCall.Function)
		if funcCall.callExpr != nil {
			for _, l := range x.Lhs {
				v := Variable{}
				extractVariable(l, &v)
				funcCall.Return = append(funcCall.Return, v)
			}
			extractFuncCallArg(funcCall.callExpr, funcCall)
		}
	case *ast.CallExpr:
		extractFunction(x, &funcCall.Function)
		extractFuncCallArg(x, funcCall)
		funcCall.callExpr = x
	}
}

func extractFuncCallArg(call *ast.CallExpr, funcCall *FuncCall) {
	for _, a := range call.Args {
		v := Variable{}
		extractVariable(a, &v)
		funcCall.Args = append(funcCall.Args, v)
	}
}

func extractFunction(n ast.Node, funcDecl *Function) {
	switch x := n.(type) {
	case *ast.FuncDecl:
		funcDecl.Name = x.Name.Name
		if x.Recv != nil && x.Recv.List != nil {
			if len(x.Recv.List) > 0 {
				switch ft := x.Recv.List[0].Type.(type) {
				case *ast.Ident:
					buf := bytes.Buffer{}
					extractHierarchy(ft, &buf)
					funcDecl.Hierarchy = correctHierarchy(buf.String())
				case *ast.StarExpr:
					buf := bytes.Buffer{}
					extractHierarchy(ft, &buf)
					funcDecl.Hierarchy = correctHierarchy(buf.String())
				}
			}
		}
		if x.Type.Params != nil && len(x.Type.Params.List) > 0 {
			for _, p := range x.Type.Params.List {
				v := Variable{}
				extractVariable(p, &v)
				funcDecl.Args = append(funcDecl.Args, v)
			}
		}
		if x.Type.Results != nil && len(x.Type.Results.List) > 0 {
			for _, r := range x.Type.Results.List {
				v := Variable{}
				extractVariable(r.Type, &v)
				funcDecl.Return = append(funcDecl.Return, v)
			}
		}
	case *ast.SelectorExpr:
		funcDecl.Name = x.Sel.Name
		buf := bytes.Buffer{}
		extractHierarchy(x.X, &buf)
		funcDecl.Hierarchy = correctHierarchy(buf.String())
	case *ast.Ident:
		funcDecl.Name = x.Name
		if x.Obj != nil {
			fd, ok := x.Obj.Decl.(*ast.FuncDecl)
			if ok {
				extractFunction(fd, funcDecl)
			}
		}
	case *ast.CallExpr:
		funcDecl.callExpr = x
		extractFunction(x.Fun, funcDecl)
	}
}

func findFuncDeclInFile(f *ast.File, name, hierarchy string, funcDecl *ast.FuncDecl) bool {
	for i := range f.Decls {
		decl, ok := f.Decls[i].(*ast.FuncDecl)
		if ok {
			f := Function{}
			extractFunction(decl, &f)
			if f.Name == name && f.Hierarchy == hierarchy {
				funcDecl = decl
				return true
			}
		}
	}
	return false
}
