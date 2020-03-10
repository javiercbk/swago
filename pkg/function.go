package pkg

import (
	"go/ast"
	"go/token"

	"github.com/javiercbk/swago/criteria"
	swagoErrors "github.com/javiercbk/swago/errors"
)

// Function is a function in a package
type Function struct {
	File     *File
	Name     string
	MemberOf string
	Args     []Variable
	Return   []string
	block    *ast.BlockStmt
	callExpr *ast.CallExpr
}

// ListVariablesUntil returns a list of all the variables until a position
func (f Function) ListVariablesUntil(until token.Pos) []Variable {
	vars := make([]Variable, 0)
	for _, a := range f.Args {
		vars = append(vars, a)
	}
	ast.Inspect(f.block, func(n ast.Node) bool {
		if n != nil {
			switch x := n.(type) {
			case *ast.AssignStmt:
				if x.Pos() < until {
					varsInAssignment := make([]Variable, 0)
					for _, l := range x.Lhs {
						v := &Variable{}
						ident, ok := l.(*ast.Ident)
						if ok {
							v.Name = ident.Name
							varsInAssignment = append(varsInAssignment, *v)
						}
					}
					if len(varsInAssignment) > 0 {
						switch r := x.Rhs[0].(type) {
						case *ast.CompositeLit:
							varsInAssignment[0].GoType = flattenType(r.Type, f.File.Pkg.Name, f.File.importMappings)
						}
						vars = append(vars, varsInAssignment...)
					}
				}
				return false
			default:
				return true
			}
		}
		return true
	})
	return vars
}

// FindArgTypeCallExpression given a call expression it finds the type of the argument
func (f Function) FindArgTypeCallExpression(callCriteria criteria.CallCriteria) (string, error) {
	var foundArg ast.Expr
	var structType string
	var foundAt token.Pos = -1
	ast.Inspect(f.block, func(n ast.Node) bool {
		if n != nil {
			switch x := n.(type) {
			case *ast.CallExpr:
				fullName := flattenType(x.Fun, f.File.Pkg.Name, f.File.importMappings)
				pkg, name := TypeParts(fullName)
				if foundAt == -1 && pkg == callCriteria.Pkg && name == callCriteria.FuncName && len(x.Args) > callCriteria.ParamIndex {
					foundArg = x.Args[callCriteria.ParamIndex]
					foundAt = x.Pos()
				}
				return false
			default:
				return true
			}
		}
		return true
	})
	if foundAt == -1 {
		return structType, swagoErrors.ErrNotFound
	}

	varName := flattenType(foundArg, f.File.Pkg.Name, f.File.importMappings)
	variables := f.ListVariablesUntil(foundAt)
	for _, v := range variables {
		if v.Name == varName {
			return v.GoType, nil
		}
	}
	return "", swagoErrors.ErrNotFound
}
