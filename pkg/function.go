package pkg

import (
	"go/ast"

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

// ListVariableNames returns a list of all the variables names in this function
func (f Function) ListVariableNames() []Variable {
	vars := make([]Variable, 0)
	for _, a := range f.Args {
		vars = append(vars, a)
	}
	ast.Inspect(f.block, func(n ast.Node) bool {
		if n != nil {
			switch x := n.(type) {
			case *ast.AssignStmt:
				for _, l := range x.Lhs {
					v := &Variable{}
					v.Extract(l)
					vars = append(vars, *v)
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
	found := false
	ast.Inspect(f.block, func(n ast.Node) bool {
		if n != nil {
			switch x := n.(type) {
			case *ast.CallExpr:
				fullName := flattenType(x.Fun, f.File.Pkg.Name, f.File.importMappings)
				pkg, name := TypeParts(fullName)
				if !found && pkg == callCriteria.Pkg && name == callCriteria.FuncName && len(x.Args) > callCriteria.ParamIndex {
					foundArg = x.Args[callCriteria.ParamIndex]
					found = true
				}
				return false
			default:
				return true
			}
		}
		return true
	})
	if !found {
		return structType, swagoErrors.ErrNotFound
	}
	return flattenType(foundArg, f.File.Pkg.Name, f.File.importMappings), nil
}
