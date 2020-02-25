package pkg

import (
	"go/ast"
)

// Function is a function in a package
type Function struct {
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
