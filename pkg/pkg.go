package pkg

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/javiercbk/swago/criteria"
	swagoErrors "github.com/javiercbk/swago/errors"

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
	EmptyInterface    = "interface{}"
	goTypeBool        = "bool"
	goTypeString      = "string"
	goTypeInt         = "int"
	goTypeInt8        = "int8"
	goTypeInt16       = "int16"
	goTypeInt32       = "int32"
	goTypeInt64       = "int64"
	goTypeUint        = "uint"
	goTypeUint8       = "uint8"
	goTypeUint16      = "uint16"
	goTypeUint32      = "uint32"
	goTypeUint64      = "uint64"
	goTypeUintptr     = "uintptr"
	goTypeByte        = "byte"
	goTypeRune        = "rune"
	goTypeFloat32     = "float32"
	goTypeFloat64     = "float64"
	goTypeComplex64   = "complex64"
	goTypeComplex128  = "complex128"
	goTypeTime        = "time.Time"
	definitionPrefix  = "#/definitions/"
	swaggerObjectType = "object"
)

var (
	errFileNotInPackage error = errors.New("file does not belong to the package")
	defaultBlackList          = []*regexp.Regexp{
		regexp.MustCompile(".*_test\\.go"),
		regexp.MustCompile(".*" + string(os.PathSeparator) + "testdata" + string(os.PathSeparator) + ".*"),
		regexp.MustCompile(".*" + string(os.PathSeparator) + "vendor" + string(os.PathSeparator) + ".*"),
	}
)

// Import represent the imports of a go file
type Import struct {
	Name string
	Pkg  string
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
					Pkg:                   file.Pkg.Name,
					File:                  file.Name,
					NamedPathVarExtractor: structRoute.NamedPathVarExtractorRegexp,
				}
				file.compositeLitToRoute(x, &foundRoute, structRoute)
				routes = append(routes, foundRoute)
			}
		}
		return true
	})
	return routes
}

// FindFunc find a function in a file
func (file *File) FindFunc(fun *Function) error {
	for _, funcInFile := range file.Functions {
		if funcInFile.Name == fun.Name {
			*fun = funcInFile
			return nil
		}
	}
	return swagoErrors.ErrNotFound
}

// FindStruct find a struct in a file
func (file *File) FindStruct(str *Struct) error {
	for _, s := range file.Structs {
		if s.Name == str.Name {
			*str = s
			str.File = file
			return nil
		}
	}
	return swagoErrors.ErrNotFound
}

func (file *File) extractType(genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		x := spec.(*ast.TypeSpec)
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
				// if field name is empty then it this is an embed struct
				fieldName := ""
				if len(f.Names) > 0 {
					fieldName = f.Names[0].Name
				}
				newField := Field{
					Tag:  tag,
					Name: fieldName,
					Type: typeStr,
				}
				s.Fields = append(s.Fields, newField)
			}
			file.Structs = append(file.Structs, s)
		}
	}
}

func (file *File) matchesStructRoute(com *ast.CompositeLit, structRoute criteria.StructRoute) bool {
	flattenedType := flattenType(com, file.Pkg.Name, file.importMappings)
	return flattenedType == structRoute.Pkg+"."+structRoute.Name
}

func (file *File) compositeLitToRoute(com *ast.CompositeLit, route *Route, structRoute criteria.StructRoute) {
	route.Struct = make(map[string]string)
	route.MatchedParameters = make(map[string]bool)
	route.MatchedSecurityDefinitions = make(map[string]bool)
	for _, e := range com.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if ok {
			ident, ok := kv.Key.(*ast.Ident)
			if ok {
				v := &Variable{}
				v.Extract(kv.Value)
				var val string
				if len(v.StrValue) > 0 {
					val = v.StrValue
				} else {
					flattenedType := flattenType(kv.Value, file.Pkg.Name, file.importMappings)
					val = flattenedType
				}
				route.Struct[ident.Name] = val
				for key, p := range structRoute.Parameters {
					if p.Always || (p.Field == ident.Name && p.MatchesRegExp.MatchString(val)) {
						route.MatchedParameters[key] = true
					}
				}
				for key, p := range structRoute.Security {
					if p.Always || (p.Field == ident.Name && p.MatchesRegExp.MatchString(val)) {
						route.MatchedSecurityDefinitions[key] = true
					}
				}
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

func (file *File) extractFunction(x *ast.FuncDecl) {
	f := Function{
		File:  file,
		Name:  x.Name.Name,
		block: x.Body,
	}
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
	file.Functions = append(file.Functions, f)
}

// AnalizeProject reads a project and returns a list of packages
func AnalizeProject(path string, logger *log.Logger) ([]*Pkg, error) {
	return AnalizeProjectWithBlacklist(path, logger, defaultBlackList)
}

// AnalizeProjectWithBlacklist reads a project excluding some files and returns a list of packages
func AnalizeProjectWithBlacklist(path string, logger *log.Logger, blacklist []*regexp.Regexp) ([]*Pkg, error) {
	pkgs := make([]*Pkg, 0)
	goFiles, err := folder.ListGoFilesRecursively(path, blacklist)
	if err != nil {
		return pkgs, err
	}
	for _, goFile := range goFiles {
		goFilePath := filepath.Dir(goFile)
		fset := token.NewFileSet()
		file, err := astForFile(goFile, fset)
		if err != nil {
			logger.Printf("error reading ast for file %s: %v\n", goFile, err)
			return pkgs, err
		}
		packageName := readFilePackage(file)
		found := false
		for i := range pkgs {
			if pkgs[i].Name == packageName && pkgs[i].Path == goFilePath {
				found = true
				break
			}
		}
		if !found {
			pkg := &Pkg{Name: packageName, Path: goFilePath, Logger: logger, BlackList: blacklist}
			err = pkg.Analyze()
			if err != nil {
				logger.Printf("error analyzing package %s in path %s: %v", packageName, goFilePath, err)
				return pkgs, err
			}
			pkgs = append(pkgs, pkg)
		}
	}
	return pkgs, nil
}

// NewPkgWithoutTest creates a new package with the default blacklist
func NewPkgWithoutTest(name, path string, logger *log.Logger) *Pkg {
	return &Pkg{Name: name, Path: path, Logger: logger, BlackList: defaultBlackList}
}

// Project is a golang project
type Project struct {
	RootPath  string
	Blacklist []*regexp.Regexp
	Pkgs      []*Pkg
}

// SearchForStructRoutes searches for struct routes
func (p *Project) SearchForStructRoutes(structRoute criteria.StructRoute) []Route {
	routes := make([]Route, 0)
	for _, p := range p.Pkgs {
		routes = append(routes, p.SearchForStructRoutes(structRoute)...)
	}
	return routes
}

// FindFunc attempts to find a function in every file of the package
func (p *Project) FindFunc(fun *Function) error {
	for _, p := range p.Pkgs {
		err := p.FindFunc(fun)
		if err == nil {
			return nil
		} else if err != swagoErrors.ErrNotFound {
			return err
		}
	}
	return swagoErrors.ErrNotFound
}

// FindStruct find a struct in a package
func (p *Project) FindStruct(str *Struct) error {
	for _, p := range p.Pkgs {
		err := p.FindStruct(str)
		if err == nil {
			return nil
		}
		if err != nil && err != swagoErrors.ErrNotFound {
			return err
		}
	}
	return swagoErrors.ErrNotFound
}

// Pkg is a package
type Pkg struct {
	Project   *Project
	Name      string
	Path      string
	Files     []File
	Logger    *log.Logger
	BlackList []*regexp.Regexp
}

// Analyze a package
func (p *Pkg) Analyze() error {
	p.Files = make([]File, 0)
	goFiles, err := folder.ListGoFiles(p.Path, p.BlackList)
	if err != nil {
		p.Logger.Printf("error listing go files for path %s: %v\n", p.Path, err)
		return err
	}
	for _, goFile := range goFiles {
		f := File{
			Pkg:  p,
			Name: path.Join(p.Path, goFile),
		}
		err = p.analyzeFile(&f)
		if err != nil {
			if err != errFileNotInPackage {
				p.Logger.Printf("error analyzing file %s: %v\n", f.Name, err)
				return err
			}
		}
		p.Files = append(p.Files, f)
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

// FindFunc attempts to find a function in every file of the package
func (p *Pkg) FindFunc(fun *Function) error {
	for _, f := range p.Files {
		err := f.FindFunc(fun)
		if err == nil {
			return nil
		} else if err != swagoErrors.ErrNotFound {
			return err
		}
	}
	return swagoErrors.ErrNotFound
}

// FindStruct find a struct in a package
func (p *Pkg) FindStruct(str *Struct) error {
	for _, f := range p.Files {
		err := f.FindStruct(str)
		if err == nil {
			return nil
		}
		if err != nil && err != swagoErrors.ErrNotFound {
			return err
		}
	}
	return swagoErrors.ErrNotFound
}

func (p *Pkg) analyzeFile(file *File) error {
	var err error
	file.FSet, file.File, err = p.astForFile(file.Name)
	if err != nil {
		return err
	}
	if p.Name != readFilePackage(file.File) {
		return errFileNotInPackage
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
			file.extractFunction(x)
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
	generateImportMappings(file)
}

func extractValueSpec(file *File, genDecl *ast.GenDecl, isConst bool) {
	for _, s := range genDecl.Specs {
		i, ok := s.(*ast.ValueSpec)
		if ok {
			for n := range i.Names {
				v := &Variable{}
				v.Extract(i.Names[n])
				// if len(i.Values) is n then this is a variable assignment with another variable
				if len(i.Values) > 0 && len(i.Values) >= n+1 {
					v.AssignValue(i.Values[n])
				}
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
				f.importMappings[i.Name] = packageName
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

func readFilePackage(f *ast.File) string {
	return f.Name.Name
}
