package ast

import (
	"go/ast"
	"go/token"
	"log"

	"github.com/javiercbk/swago/criteria"
)

type FileImport struct {
	Name string
	Pkg  string
}

// FuncDecl is a function declaration
type FuncDecl struct {
	Pkg      string
	Name     string
	MemberOf string
	funcDecl *ast.FuncDecl
}

type FuncCall struct {
	callExpr *ast.CallExpr
}

// Route is a route handled found
type Route struct {
	File          string
	HTTPMethod    string
	Path          string
	HandlerPkg    string
	HandlerName   string
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

// Manager is an abstraction that can read ast for files
type Manager interface {
	GetFileImports(filePath string) ([]FileImport, error)
	ExtractRoutesFromFile(filePath string, criterias []criteria.RouteCriteria) ([]Route, error)
	FindFuncDeclaration(filePath string, funcCall FuncCall) (FuncDecl, error)
	FindCallsInFunc(funcDecl FuncDecl) []FuncCall
	FindCallCriteria(funcDecl FuncDecl, callCriteria []criteria.CallCriteria)
	FindStruct(filePath string, s *Struct) bool
}

type naiveManager struct {
	logger *log.Logger
}

func (m naiveManager) GetFileImports(filePath string) ([]FileImport, error) {
	var imports []FileImport
	f := ast.File{}
	err := m.astForFile(filePath, &f)
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

func (m naiveManager) ExtractRoutesFromFile(filePath string, criterias []criteria.RouteCriteria) ([]Route, error) {
	var routes []Route
	f := ast.File{}
	err := m.astForFile(filePath, &f)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return routes, err
	}
	return routes, nil
}
func (m naiveManager) FindFuncDeclaration(filePath string, funcCall FuncCall) (FuncDecl, error) {
	var decl FuncDecl
	f := ast.File{}
	err := m.astForFile(filePath, &f)
	if err != nil {
		m.logger.Printf("error parsing ast from file %s: %v\n", filePath, err)
		return decl, err
	}
	return decl, nil
	// searchFileForRouteCriteria()
}
func (m naiveManager) FindCallsInFunc(funcDecl FuncDecl) []FuncCall {
	return make([]FuncCall, 0)
}

func (m naiveManager) FindCallCriteria(funcDecl FuncDecl, callCriteria []criteria.CallCriteria) {

}

func (m naiveManager) FindStruct(filePath string, s *Struct) bool {
	return false
}

func (m naiveManager) astForFile(filePath string, f *ast.File) error {
	var err error
	m.logger.Printf("parsing ast from file %s\n", filePath)
	fset := token.NewFileSet()
	f, err = astForFile(filePath, fset)
	return err
}

// NewManager creates the default Manager
func NewManager(logger *log.Logger) Manager {
	return naiveManager{
		logger: logger,
	}
}
