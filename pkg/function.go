package pkg

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

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
			modelResponse.Code = parseCode(code)
			return nil
		}
	}
	return swagoErrors.ErrNotFound
}

func parseCode(code string) string {
	// FIXME: this function should be configurable
	_, err := strconv.Atoi(code)
	if err == nil {
		// code is already a number, no change is needed
		return code
	}
	if strings.Contains(code, "StatusContinue") {
		return "100"
	}
	if strings.Contains(code, "StatusSwitchingProtocols") {
		return "101"
	}
	if strings.Contains(code, "StatusProcessing") {
		return "102"
	}
	if strings.Contains(code, "StatusEarlyHints") {
		return "103"
	}
	if strings.Contains(code, "StatusOK") {
		return "200"
	}
	if strings.Contains(code, "StatusCreated") {
		return "201"
	}
	if strings.Contains(code, "StatusAccepted") {
		return "202"
	}
	if strings.Contains(code, "StatusNonAuthoritativeInfo") {
		return "203"
	}
	if strings.Contains(code, "StatusNoContent") {
		return "204"
	}
	if strings.Contains(code, "StatusResetContent") {
		return "205"
	}
	if strings.Contains(code, "StatusPartialContent") {
		return "206"
	}
	if strings.Contains(code, "StatusMultiStatus") {
		return "207"
	}
	if strings.Contains(code, "StatusAlreadyReported") {
		return "208"
	}
	if strings.Contains(code, "StatusIMUsed") {
		return "226"
	}
	if strings.Contains(code, "StatusMultipleChoices") {
		return "300"
	}
	if strings.Contains(code, "StatusMovedPermanently") {
		return "301"
	}
	if strings.Contains(code, "StatusFound") {
		return "302"
	}
	if strings.Contains(code, "StatusSeeOther") {
		return "303"
	}
	if strings.Contains(code, "StatusNotModified") {
		return "304"
	}
	if strings.Contains(code, "StatusUseProxy") {
		return "305"
	}
	if strings.Contains(code, "StatusTemporaryRedirect") {
		return "307"
	}
	if strings.Contains(code, "StatusPermanentRedirect") {
		return "308"
	}
	if strings.Contains(code, "StatusBadRequest") {
		return "400"
	}
	if strings.Contains(code, "StatusUnauthorized") {
		return "401"
	}
	if strings.Contains(code, "StatusPaymentRequired") {
		return "402"
	}
	if strings.Contains(code, "StatusForbidden") {
		return "403"
	}
	if strings.Contains(code, "StatusNotFound") {
		return "404"
	}
	if strings.Contains(code, "StatusMethodNotAllowed") {
		return "405"
	}
	if strings.Contains(code, "StatusNotAcceptable") {
		return "406"
	}
	if strings.Contains(code, "StatusProxyAuthRequired") {
		return "407"
	}
	if strings.Contains(code, "StatusRequestTimeout") {
		return "408"
	}
	if strings.Contains(code, "StatusConflict") {
		return "409"
	}
	if strings.Contains(code, "StatusGone") {
		return "410"
	}
	if strings.Contains(code, "StatusLengthRequired") {
		return "411"
	}
	if strings.Contains(code, "StatusPreconditionFailed") {
		return "412"
	}
	if strings.Contains(code, "StatusRequestEntityTooLarge") {
		return "413"
	}
	if strings.Contains(code, "StatusRequestURITooLong") {
		return "414"
	}
	if strings.Contains(code, "StatusUnsupportedMediaType") {
		return "415"
	}
	if strings.Contains(code, "StatusRequestedRangeNotSatisfiable") {
		return "416"
	}
	if strings.Contains(code, "StatusExpectationFailed") {
		return "417"
	}
	if strings.Contains(code, "StatusTeapot") {
		return "418"
	}
	if strings.Contains(code, "StatusMisdirectedRequest") {
		return "421"
	}
	if strings.Contains(code, "StatusUnprocessableEntity") {
		return "422"
	}
	if strings.Contains(code, "StatusLocked") {
		return "423"
	}
	if strings.Contains(code, "StatusFailedDependency") {
		return "424"
	}
	if strings.Contains(code, "StatusTooEarly") {
		return "425"
	}
	if strings.Contains(code, "StatusUpgradeRequired") {
		return "426"
	}
	if strings.Contains(code, "StatusPreconditionRequired") {
		return "428"
	}
	if strings.Contains(code, "StatusTooManyRequests") {
		return "429"
	}
	if strings.Contains(code, "StatusRequestHeaderFieldsTooLarge") {
		return "431"
	}
	if strings.Contains(code, "StatusUnavailableForLegalReasons") {
		return "451"
	}
	if strings.Contains(code, "StatusInternalServerError") {
		return "500"
	}
	if strings.Contains(code, "StatusNotImplemented") {
		return "501"
	}
	if strings.Contains(code, "StatusBadGateway") {
		return "502"
	}
	if strings.Contains(code, "StatusServiceUnavailable") {
		return "503"
	}
	if strings.Contains(code, "StatusGatewayTimeout") {
		return "504"
	}
	if strings.Contains(code, "StatusHTTPVersionNotSupported") {
		return "505"
	}
	if strings.Contains(code, "StatusVariantAlsoNegotiates") {
		return "506"
	}
	if strings.Contains(code, "StatusInsufficientStorage") {
		return "507"
	}
	if strings.Contains(code, "StatusLoopDetected") {
		return "508"
	}
	if strings.Contains(code, "StatusNotExtended") {
		return "510"
	}
	if strings.Contains(code, "StatusNetworkAuthenticationRequired") {
		return "511"
	}
	return code
}
