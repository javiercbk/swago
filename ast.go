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

// AnalyzeDirectory scans all go files for patterns
func AnalyzeDirectory(dir string) error {
	goFiles, err := listGoFiles(dir)
	if err != nil {
		return nil
	}
	for i := range goFiles {
		if err = AnalyzeFile(goFiles[i]); err != nil {
			return err
		}
	}
	return nil
}

// AnalyzeFile scans a file for patterns
func AnalyzeFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return AnalyzeReader(filePath, f)
}

// AnalyzeReader scans a reader for patterns
func AnalyzeReader(filePath string, r io.Reader) error {
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
	ast.Inspect(parsed, func(n ast.Node) bool {
		if n != nil {
			switch x := n.(type) {
			case *ast.FuncDecl:
				analyzeFuncDecl(fset, x)
			case *ast.FuncLit:
				analyzeFuncLit(fset, x)
			case *ast.CallExpr:
				analyzeCallExpr(fset, x)
			}
		}

		return true
	})
	return nil
}

func analyzeFuncDecl(fset *token.FileSet, decl *ast.FuncDecl) {
	x := decl.Type
	if len(x.Params.List) == 2 {
		params := x.Params.List
		for paramI := range x.Params.List {
			p, ok := params[paramI].Type.(*ast.SelectorExpr)
			if ok {
				id, ok := p.X.(*ast.Ident)
				if ok && id.Name == "http" && p.Sel.Name == "ResponseWriter" {
					fmt.Printf("%s=>%s\t%v\n", fset.Position(decl.Pos()), decl.Name, params[paramI].Type)
				}
			}
		}
	}
}

func analyzeFuncLit(fset *token.FileSet, lit *ast.FuncLit) {
	x := lit.Type
	if len(x.Params.List) == 2 {
		params := x.Params.List
		for paramI := range x.Params.List {
			p, ok := params[paramI].Type.(*ast.SelectorExpr)
			if ok {
				id, ok := p.X.(*ast.Ident)
				if ok && id.Name == "http" && p.Sel.Name == "ResponseWriter" {
					fmt.Printf("%s=>%s\t%v\n", fset.Position(lit.Pos()), "FuncLit", params[paramI].Type)
				}
			}
		}
	}
}

func analyzeCallExpr(fset *token.FileSet, callExp *ast.CallExpr) {
	fmt.Printf("%s=>\t%v\n", fset.Position(callExp.Pos()), callExp)
	selectorExpression, ok := callExp.Fun.(*ast.SelectorExpr)
	if ok {
		httpMethod := selectorExpression.Sel.Name
		fmt.Printf("httpMethod %s\n", httpMethod)
		ast.Inspect(selectorExpression.X, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				analyzeFuncDecl(fset, x)
			case *ast.FuncLit:
				analyzeFuncLit(fset, x)
			case *ast.CallExpr:
				analyzeCallExpr(fset, x)
			}
			return true
		})
		ident, ok := selectorExpression.X.(*ast.Ident)
		if ok {
			field, _ := ident.Obj.Decl.(*ast.Field)
			fmt.Printf("field: %v\n", field)
			// if ok {
			// starExpr, ok := field.Type.(*ast.StarExpr)
			/*
								a:<*go/ast.SelectorExpr>(0xc00000c1e0)
				:<go/ast.SelectorExpr>
				X:<go/ast.Expr>
				Sel:<*go/ast.Ident>(0xc00000c1c0)
				:<go/ast.Ident>
				NamePos:84
				Name:"Group"
				Obj:nil <*go/ast.Object>
				data:<*go/ast.Ident>(0xc00000c1a0)
				:<go/ast.Ident>
				NamePos:79
				Name:"echo"
				Obj:nil <*go/ast.Object>

			*/
			// if ok {
			// 	starExpr.X
			// } else {

			// }
			// }
		}
	}
}
