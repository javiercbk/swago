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
	var foundAt token.Pos = -1
	var ident *ast.Ident
	ast.Inspect(f.block, func(n ast.Node) bool {
		if n != nil {
			switch x := n.(type) {
			case *ast.CallExpr:
				fullName := flattenType(x.Fun, f.File.Pkg.Name, f.File.importMappings)
				pkg, name := TypeParts(fullName)
				if foundAt == -1 && pkg == callCriteria.Pkg && name == callCriteria.FuncName && len(x.Args) > callCriteria.ParamIndex {
					varAST := x.Args[callCriteria.ParamIndex]
					switch varExpr := varAST.(type) {
					case *ast.UnaryExpr:
						var ok bool
						foundAt = x.Pos()
						ident, ok = varExpr.X.(*ast.Ident)
						if !ok {
							foundAt = -1
							ident = nil
						}
					case *ast.Ident:
						foundAt = x.Pos()
						ident = varExpr
					}
				}
				return false
			default:
				return true
			}
		}
		return true
	})
	if foundAt == -1 {
		return "", swagoErrors.ErrNotFound
	}
	variables := f.ListVariablesUntil(foundAt)
	for _, v := range variables {
		if v.Name == ident.Name {
			return v.GoType, nil
		}
	}
	return "", swagoErrors.ErrNotFound
}

// ModelResponse is a response model
type ModelResponse struct {
	Type string
	Pos  token.Pos
	Code string
}

// FindResponseCallExpressionAfter given a call expression it finds the type of the argument past a position
func (f Function) FindResponseCallExpressionAfter(callCriteria criteria.ResponseCallCriteria, pos *token.Pos, modelResponse *ModelResponse) error {
	//FIXME: need to re/write this and FindArgTypeCallExpression function
	var foundAt token.Pos = -1
	var ident *ast.Ident
	var code string
	ast.Inspect(f.block, func(n ast.Node) bool {
		if n != nil {
			if n.Pos() > *pos {
				switch x := n.(type) {
				case *ast.CallExpr:
					fullName := flattenType(x.Fun, f.File.Pkg.Name, f.File.importMappings)
					pkg, name := TypeParts(fullName)
					// FIXME: should be allowed to continue and return a slice
					if foundAt == -1 && pkg == callCriteria.Pkg && name == callCriteria.FuncName && callCriteria.ParamIndex >= 0 && len(x.Args) > callCriteria.ParamIndex {
						varAST := x.Args[callCriteria.ParamIndex]
						switch varExpr := varAST.(type) {
						case *ast.UnaryExpr:
							var ok bool
							foundAt = x.Pos()
							ident, ok = varExpr.X.(*ast.Ident)
							if !ok {
								foundAt = -1
								ident = nil
							}
						case *ast.Ident:
							foundAt = x.Pos()
							ident = varExpr
						}
						codeAST := x.Args[callCriteria.CodeIndex]
						switch codeExpr := codeAST.(type) {
						case *ast.Ident:
							code = codeExpr.Name
						case *ast.SelectorExpr:
							code = flattenType(codeExpr, f.File.Pkg.Name, f.File.importMappings)
						case *ast.BasicLit:
							code = codeExpr.Value
						}
					}
					return false
				default:
					return true
				}
			}
		}
		return true
	})
	if foundAt == -1 {
		return swagoErrors.ErrNotFound
	}
	*pos = foundAt
	variables := f.ListVariablesUntil(foundAt)
	for _, v := range variables {
		if v.Name == ident.Name {
			modelResponse.Type = v.GoType
			modelResponse.Pos = foundAt
			modelResponse.Code = code
			return nil
		}
	}
	return swagoErrors.ErrNotFound
}

// FindErrorResponseCallExpressionAfter given a call expression it finds the type of the argument past a position
func (f Function) FindErrorResponseCallExpressionAfter(callCriteria criteria.ResponseCallCriteria, pos *token.Pos, modelResponse *ModelResponse) error {
	//FIXME: need to re/write this and FindArgTypeCallExpression function
	var foundAt token.Pos = -1
	var code string
	ast.Inspect(f.block, func(n ast.Node) bool {
		if n != nil {
			if n.Pos() > *pos {
				switch x := n.(type) {
				case *ast.CallExpr:
					fullName := flattenType(x.Fun, f.File.Pkg.Name, f.File.importMappings)
					pkg, name := TypeParts(fullName)
					// FIXME: should be allowed to continue and return a slice
					if foundAt == -1 && pkg == callCriteria.Pkg && name == callCriteria.FuncName && len(x.Args) > callCriteria.CodeIndex {
						foundAt = x.Pos()
						codeAST := x.Args[callCriteria.CodeIndex]
						switch codeExpr := codeAST.(type) {
						case *ast.Ident:
							code = codeExpr.Name
						case *ast.SelectorExpr:
							code = flattenType(codeExpr, f.File.Pkg.Name, f.File.importMappings)
						case *ast.BasicLit:
							code = codeExpr.Value
						}
					}
					return false
				default:
					return true
				}
			}
		}
		return true
	})
	if foundAt == -1 {
		return swagoErrors.ErrNotFound
	}
	*pos = foundAt
	modelResponse.Pos = foundAt
	modelResponse.Code = code
	return nil
}
