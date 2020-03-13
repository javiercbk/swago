package pkg

import (
	"go/ast"
	"go/token"
	"strings"
)

// Array is a go array
type Array struct {
	Type      Variable
	StrValues []string
}

// Variable is a variable
type Variable struct {
	Name        string
	GoType      string
	StrValue    string
	MapValue    map[string]*Variable
	ArrayValue  *Array
	SubVariable *Variable
}

func (v *Variable) getLastVar() *Variable {
	var aux *Variable
	aux = v
	for aux.SubVariable != nil {
		aux = aux.SubVariable
	}
	return aux
}

// AssignValue given the ast node assigns the variable a value
func (v *Variable) AssignValue(n ast.Node) {
	h := &variableHelper{}
	h.assign(n, v)
}

// Extract its structure from an ast node
func (v *Variable) Extract(n ast.Node) {
	ve := &variableHelper{}
	ve.extract(n, v)
}

type variableHelper struct {
	level int
}

func (e *variableHelper) assign(n ast.Node, v *Variable) {
	lastVar := v.getLastVar()
	switch x := n.(type) {
	case *ast.BasicLit:
		val := x.Value
		if x.Kind == token.STRING {
			lastVar.GoType = "string"
			val = strings.Trim(val, "\"")
		} else {
			lastVar.GoType = "number"
		}
		lastVar.StrValue = val
	case *ast.Ident:
		if isGoType(x.Name) {
			lastVar.GoType = x.Name
			lastVar.StrValue = x.Name
		} else {
			lastVar.Name = x.Name
		}
	case *ast.CompositeLit:
		array, ok := x.Type.(*ast.ArrayType)
		if ok {
			// this is an array
			arrTypeVar := &Variable{}
			arrTypeVar.Extract(array.Elt)
			lastVar.ArrayValue = &Array{
				Type: *arrTypeVar,
			}
		} else {
			if len(x.Elts) > 0 {
				switch x.Elts[0].(type) {
				case *ast.CompositeLit:
					// slice
					arrTypeVar := &Variable{}
					// arrTypeVar.Extract(x.Elts)
					lastVar.ArrayValue = &Array{
						Type: *arrTypeVar,
					}
				case *ast.KeyValueExpr:
					lastVar.MapValue = make(map[string]*Variable)
					for _, el := range x.Elts {
						kv := el.(*ast.KeyValueExpr)
						ident := kv.Key.(*ast.Ident)
						valVar := &Variable{}
						valVar.Extract(kv.Value)
						lastVar.MapValue[ident.Name] = valVar
					}
				}
			}
		}
	}
}

func (e *variableHelper) extract(n ast.Node, v *Variable) {
	switch x := n.(type) {
	case *ast.Ident:
		levelV := e.getLevel(v)
		levelV.Name = x.Name
		e.level++
	case *ast.SelectorExpr:
		e.extract(x.X, v)
		levelV := e.getLevel(v)
		levelV.Name = x.Sel.Name
	case *ast.BasicLit:
		e.assign(n, v)
	}
}

func (e *variableHelper) getLevel(v *Variable) *Variable {
	var aux *Variable
	aux = v
	for i := 0; i > e.level; i++ {
		if aux.SubVariable == nil {
			aux.SubVariable = &Variable{}
			aux = aux.SubVariable
		} else {
			aux = aux.SubVariable
		}
	}
	return aux
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

func flattenType(n ast.Node, fallbackPkg string, importMappings map[string]string) string {
	flattenedType := rawFlattenType(n, importMappings)
	if !strings.Contains(flattenedType, ".") && !isGoType(flattenedType) {
		flattenedType = fallbackPkg + "." + flattenedType
	}
	return flattenedType
}

func rawFlattenType(n ast.Node, importMappings map[string]string) string {
	switch x := n.(type) {
	case *ast.StarExpr:
		return "*" + rawFlattenType(x.X, importMappings)
	case *ast.Ident:
		mapped, ok := importMappings[x.Name]
		if ok {
			return mapped
		}
		return x.Name
	case *ast.UnaryExpr:
		if x.Op == token.AND {
			return "*" + rawFlattenType(x.X, importMappings)
		}
		return rawFlattenType(x.X, importMappings)
	case *ast.SelectorExpr:
		return rawFlattenType(x.X, importMappings) + "." + x.Sel.Name
	case *ast.CompositeLit:
		return rawFlattenType(x.Type, importMappings)
	default:
		return ""
	}
}

// TypeParts returns the package and the name of a flattened type
func TypeParts(selector string) (string, string) {
	parts := strings.Split(selector, ".")
	if len(parts) > 1 {
		return parts[0], parts[1]
	}
	return "", selector
}

func swaggerType(t string) (string, string) {
	switch t {
	case EmptyInterface:
		return "object", ""
	case goTypeBool:
		return "boolean", ""
	case goTypeString:
		return "string", ""
	case goTypeInt:
		return "integer", "int32"
	case goTypeInt8:
		return "integer", "int32"
	case goTypeInt16:
		return "integer", "int32"
	case goTypeInt32:
		return "integer", "int32"
	case goTypeInt64:
		return "integer", "int64"
	case goTypeUint:
		return "integer", "int32"
	case goTypeUint8:
		return "integer", "int32"
	case goTypeUint16:
		return "integer", "int32"
	case goTypeUint32:
		return "integer", "int32"
	case goTypeUint64:
		return "integer", "int64"
	case goTypeUintptr:
		return "integer", "int64"
	case goTypeByte:
		return "integer", "int32"
	case goTypeRune:
		return "string", ""
	case goTypeFloat32:
		return "number", "float"
	case goTypeFloat64:
		return "number", "double"
	case goTypeComplex64:
		return "string", ""
	case goTypeComplex128:
		return "string", ""
	case goTypeTime:
		return "string", "date-time"
	default:
		return "", ""
	}
}
