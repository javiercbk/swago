package pkg

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/javiercbk/swago/criteria"

	"github.com/javiercbk/swago/folder"
)

const (
	// ImportKeyWord is the keyword "import"
	ImportKeyWord string = "import"
	// VarKeyWord is the keyword "var"
	VarKeyWord string = "var"
	// ConstKeyWord is the keyword "const"
	ConstKeyWord string = "const"
	// TypeKeyWord is the keyword "type"
	TypeKeyWord string = "type"
	// EmptyInterface is the empty interface keyword
	EmptyInterface string = "interface{}"

	goTypeBool       string = "bool"
	goTypeString     string = "string"
	goTypeInt        string = "int"
	goTypeInt8       string = "int8"
	goTypeInt16      string = "int16"
	goTypeInt32      string = "int32"
	goTypeInt64      string = "int64"
	goTypeUint       string = "uint"
	goTypeUint8      string = "uint8"
	goTypeUint16     string = "uint16"
	goTypeUint32     string = "uint32"
	goTypeUint64     string = "uint64"
	goTypeUintptr    string = "uintptr"
	goTypeByte       string = "byte"
	goTypeRune       string = "rune"
	goTypeFloat32    string = "float32"
	goTypeFloat64    string = "float64"
	goTypeComplex64  string = "complex64"
	goTypeComplex128 string = "complex128"
)

var (
	defaultBlackList = []*regexp.Regexp{
		regexp.MustCompile(".*_test\\.go"),
		regexp.MustCompile(".*" + string(os.PathSeparator) + "testdata" + string(os.PathSeparator) + ".*"),
	}
)

// Import represent the imports of a go file
type Import struct {
	Name string
	Pkg  string
}

// Field is a struct field
type Field struct {
	Name string
	Type string
	Tag  string
}

// Struct is a struct
type Struct struct {
	Name   string
	Fields []Field
}

// File is a go file
type File struct {
	Pkg            *Pkg
	Name           string
	FSet           *token.FileSet
	File           *ast.File
	Structs        []Struct
	Imports        []Import
	GlobalVars     []Variable
	GlobalConst    []Variable
	Functions      []Function
	importMappings map[string]string
}

// SearchForStructRoutes searches for struct routes inside a file
func (file *File) SearchForStructRoutes(structRoute criteria.StructRoute) []Route {
	routes := make([]Route, 0)
	ast.Inspect(file.File, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CompositeLit:
			if file.matchesStructRoute(x, structRoute) {
				foundRoute := Route{
					File: file.Name,
				}
				file.compositeLitToRoute(x, &foundRoute, structRoute)
				routes = append(routes, foundRoute)
			}
		}
		return true
	})
	return routes
}

func (file *File) extractType(genDecl *ast.GenDecl) {
	x := genDecl.Specs[0].(*ast.TypeSpec)
	st, ok := x.Type.(*ast.StructType)
	if ok {
		s := Struct{
			Name:   x.Name.Name,
			Fields: make([]Field, 0),
		}
		for _, f := range st.Fields.List {
			typeStr := flattenType(f.Type, file.Pkg.Name, file.importMappings)
			tag := ""
			if f.Tag != nil {
				tag = strings.Trim(f.Tag.Value, "`")
				tag = strings.ReplaceAll(tag, "\\\"", "\"")
			}
			newField := Field{
				Tag:  tag,
				Name: f.Names[0].Name,
				Type: typeStr,
			}
			s.Fields = append(s.Fields, newField)
		}
		file.Structs = append(file.Structs, s)
	}
}

func (file *File) matchesStructRoute(com *ast.CompositeLit, structRoute criteria.StructRoute) bool {
	flattenedType := flattenType(com, file.Pkg.Name, file.importMappings)
	return flattenedType == structRoute.Name
}

func (file *File) compositeLitToRoute(com *ast.CompositeLit, route *Route, structRoute criteria.StructRoute) {
	route.Struct = make(map[string]string)
	for _, e := range com.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if ok {
			ident, ok := kv.Key.(*ast.Ident)
			if ok {
				v := &Variable{}
				v.Extract(com)
				var val string
				if len(v.StrValue) > 0 {
					val = v.StrValue
				} else {
					flattenedType := flattenType(com, file.Pkg.Name, file.importMappings)
					val = flattenedType
				}
				route.Struct[ident.Name] = val
				switch ident.Name {
				case structRoute.PathField:
					route.Path = val
				case structRoute.HandlerField:

					route.HandlerType = val
				case structRoute.HTTPMethodField:
					route.HTTPMethod = criteria.MatchHTTPMethod(val)
				}
			}
		}
	}
}

func (file *File) extractFunction(x *ast.FuncDecl, f *Function) {
	f.Name = x.Name.Name
	if x.Recv != nil && x.Recv.List != nil {
		if len(x.Recv.List) > 0 {
			switch ft := x.Recv.List[0].Type.(type) {
			case *ast.Ident:
				f.MemberOf = ft.Name
			case *ast.StarExpr:
				typeIdent := ft.X.(*ast.Ident)
				f.MemberOf = "*" + typeIdent.Name
			}
		}
	}
	if x.Type.Params != nil && len(x.Type.Params.List) > 0 {
		for _, pl := range x.Type.Params.List {
			goType := flattenType(pl.Type, file.Pkg.Name, file.importMappings)
			for _, n := range pl.Names {
				f.Args = append(f.Args, Variable{
					Name:   n.Name,
					GoType: goType,
				})
			}

		}
	}
	if x.Type.Results != nil && len(x.Type.Results.List) > 0 {
		for _, r := range x.Type.Results.List {
			f.Return = append(f.Return, flattenType(r.Type, file.Pkg.Name, file.importMappings))
		}
	}
}

// Pkg is a package
type Pkg struct {
	Name      string
	Path      string
	Files     []File
	Logger    *log.Logger
	BlackList []*regexp.Regexp
}

// NewPkgWithoutTest creates a new package with the default blacklist
func NewPkgWithoutTest(path string, logger *log.Logger) *Pkg {
	return &Pkg{Path: path, Logger: logger, BlackList: defaultBlackList}
}

// Analyze a package
func (p *Pkg) Analyze() error {
	goFiles, err := folder.ListGoFiles(p.Path, p.BlackList)
	if err != nil {
		p.Logger.Printf("error listing go files for path %s: %v\n", p.Path, err)
		return err
	}
	for _, goFile := range goFiles {
		f := File{
			Pkg:  p,
			Name: goFile,
		}
		err = p.analyzeFile(&f)
		if err != nil {
			p.Logger.Printf("error analyzing file %s: %v\n", f.Name, err)
			return err
		}
	}
	return nil
}

// SearchForStructRoutes searches for struct routes
func (p *Pkg) SearchForStructRoutes(structRoute criteria.StructRoute) []Route {
	routes := make([]Route, 0)
	for _, f := range p.Files {
		routes = append(routes, f.SearchForStructRoutes(structRoute)...)
	}
	return routes
}

func (p *Pkg) analyzeFile(file *File) error {
	var err error
	file.FSet, file.File, err = p.astForFile(file.Name)
	if err != nil {
		return err
	}
	// First parse imports
	for _, d := range file.File.Decls {
		switch x := d.(type) {
		case *ast.GenDecl:
			switch x.Tok.String() {
			case ImportKeyWord:
				extractImport(file, x)
			}
		}
	}
	// only after parsing imports, parse this
	for _, d := range file.File.Decls {
		switch x := d.(type) {
		case *ast.GenDecl:
			switch x.Tok.String() {
			case VarKeyWord:
				extractValueSpec(file, x, false)
			case ConstKeyWord:
				extractValueSpec(file, x, true)
			case TypeKeyWord:
				file.extractType(x)
			}
		case *ast.FuncDecl:
			f := Function{}
			file.extractFunction(x, &f)
		}
	}
	return nil
}

func (p *Pkg) astForFile(filePath string) (*token.FileSet, *ast.File, error) {
	p.Logger.Printf("parsing ast from file %s\n", filePath)
	fset := token.NewFileSet()
	file, err := astForFile(filePath, fset)
	if err != nil {
		p.Logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
	}
	return fset, file, err
}

func extractImport(file *File, genDecl *ast.GenDecl) {
	for _, s := range genDecl.Specs {
		i, ok := s.(*ast.ImportSpec)
		if ok {
			imp := Import{
				Pkg: strings.Trim(i.Path.Value, "\""),
			}
			if i.Name != nil {
				imp.Name = i.Name.Name
			}
			file.Imports = append(file.Imports, imp)
		}
	}
}

func extractValueSpec(file *File, genDecl *ast.GenDecl, isConst bool) {
	for _, s := range genDecl.Specs {
		i, ok := s.(*ast.ValueSpec)
		if ok {
			for n := range i.Names {
				v := &Variable{}
				v.Extract(i.Names[n])
				v.AssignValue(i.Values[n])
				if isConst {
					file.GlobalConst = append(file.GlobalConst, *v)
				} else {
					file.GlobalVars = append(file.GlobalVars, *v)
				}
			}
		}
	}
}

// generateImportMappings generates a mapping of package name to package given name
func generateImportMappings(f *File) {
	if f.importMappings == nil {
		f.importMappings = make(map[string]string)
		for _, i := range f.Imports {
			splitted := strings.Split(i.Pkg, "/")
			packageName := splitted[len(splitted)-1]
			if len(i.Name) > 0 {
				f.importMappings[packageName] = i.Name
			} else {
				f.importMappings[packageName] = packageName
			}
		}
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
