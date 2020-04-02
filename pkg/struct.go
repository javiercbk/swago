package pkg

import (
	"strconv"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/javiercbk/swago/criteria"
)

// Field is a struct field
type Field struct {
	Name string
	Type string
	Tag  string
}

// Struct is a struct
type Struct struct {
	File         *File
	PkgName      string
	Name         string
	Fields       []Field
	CallCriteria criteria.CallCriteria
	Schema       *openapi3.Schema
}

// ToSwaggerSchema populates a given swagger schema with the data from the struct
func (s *Struct) ToSwaggerSchema(parameter *openapi2.Parameter) error {
	properties := make(map[string]*openapi3.SchemaRef)
	requiredProps := make([]string, 0)
	s.addEmbeddedStruct()
	for _, f := range s.Fields {
		t, format := swaggerType(f.Type)
		if extractBooleanValidation(criteria.RequiredValidation, f.Tag, s.CallCriteria) {
			requiredProps = append(requiredProps, f.Name)
		}
		if len(t) > 0 {
			sch := &openapi3.Schema{
				Type:         t,
				Format:       format,
				ExclusiveMin: extractBooleanValidation(criteria.ExclusiveMinValidation, f.Tag, s.CallCriteria),
				ExclusiveMax: extractBooleanValidation(criteria.ExclusiveMaxValidation, f.Tag, s.CallCriteria),
				Enum:         matchesInterfaceSlice(criteria.EnumValidation, f.Tag, s.CallCriteria),
			}
			schRef := &openapi3.SchemaRef{
				Value: sch,
			}
			min, minOk := extractFloat64(criteria.MinimumValidation, f.Tag, s.CallCriteria)
			max, maxOk := extractFloat64(criteria.MaximumValidation, f.Tag, s.CallCriteria)
			minLength, minLengthOk := extractUint64(criteria.MinLengthValidation, f.Tag, s.CallCriteria)
			maxLength, maxLengthOk := extractUint64(criteria.MaxLengthValidation, f.Tag, s.CallCriteria)
			pattern, patternOk := extractString(criteria.PatternValidation, f.Tag, s.CallCriteria)
			if minOk {
				sch.Min = &min
			}
			if maxOk {
				sch.Max = &max
			}
			if minLengthOk {
				sch.MinLength = minLength
			}
			if maxLengthOk {
				sch.MaxLength = &maxLength
			}
			if patternOk {
				sch.Pattern = pattern
			}
			properties[f.Name] = schRef
		} else {
			subStruct := Struct{}
			err := s.File.Pkg.Project.FindStruct(&subStruct)
			if err != nil {
				return err
			}
			subStruct.CallCriteria = s.CallCriteria
			sch := &openapi3.Schema{}
			err = subStruct.ToSwaggerSchema(nil)
			if err != nil {
				return err
			}
			schRef := &openapi3.SchemaRef{
				Value: sch,
			}
			properties[f.Name] = schRef
		}
	}
	s.Schema = &openapi3.Schema{}
	s.Schema.Type = swaggerObjectType
	s.Schema.Required = requiredProps
	s.Schema.Properties = properties
	if parameter != nil {
		parameter.Schema = &openapi3.SchemaRef{
			Value: s.Schema,
		}
		parameter.Name = s.Name
		parameter.Required = true
	}
	return nil
}

type embeddedStruct struct {
	index int
	pkg   string
	name  string
}

func (s *Struct) addEmbeddedStruct() error {
	embeddedStructs := make([]embeddedStruct, 0)
	for i := range s.Fields {
		if len(s.Fields[i].Name) == 0 {
			pkg, name := TypeParts(s.Fields[i].Type)
			embeddedStructs = append(embeddedStructs, embeddedStruct{
				index: i,
				pkg:   pkg,
				name:  name,
			})
		}
	}
	for i := len(embeddedStructs) - 1; i > 0; i-- {
		idx := embeddedStructs[i].index
		s.Fields = append(s.Fields[:idx], s.Fields[idx+1:]...)
	}
	for _, es := range embeddedStructs {
		embeddedStruct := Struct{
			Name: es.name,
		}
		err := s.File.Pkg.Project.FindStruct(&embeddedStruct)
		if err != nil {
			return err
		}
		embeddedStruct.addEmbeddedStruct()
		s.Fields = append(s.Fields, embeddedStruct.Fields...)
	}
	return nil
}

func extractBooleanValidation(validationName string, tag string, callCriteria criteria.CallCriteria) bool {
	e, ok := callCriteria.Validations[validationName]
	if ok {
		for _, r := range e.TagRegexp {
			if r.MatchString(tag) {
				return true
			}
		}
	}
	return false
}

func matchesInterfaceSlice(validationName string, tag string, callCriteria criteria.CallCriteria) []interface{} {
	enumItems := make([]interface{}, 0)
	e, ok := callCriteria.Validations[validationName]
	if ok {
		for _, r := range e.TagRegexp {
			found := r.FindStringSubmatch(tag)
			for _, f := range found {
				enumItems = append(enumItems, f)
			}
		}
	}
	return enumItems
}

func extractFloat64(validationName string, tag string, callCriteria criteria.CallCriteria) (float64, bool) {
	e, ok := callCriteria.Validations[validationName]
	if ok {
		for _, r := range e.TagRegexp {
			found := r.FindStringSubmatch(tag)
			if len(found) > 0 {
				parsed, err := strconv.ParseFloat(found[0], 64)
				if err == nil {
					return parsed, true
				}
			}
		}
	}
	return 0, false
}

func extractUint64(validationName string, tag string, callCriteria criteria.CallCriteria) (uint64, bool) {
	e, ok := callCriteria.Validations[validationName]
	if ok {
		for _, r := range e.TagRegexp {
			found := r.FindStringSubmatch(tag)
			if len(found) > 0 {
				parsed, err := strconv.ParseUint(found[0], 10, 64)
				if err == nil {
					return parsed, true
				}
			}
		}
	}
	return 0, false
}

func extractString(validationName string, tag string, callCriteria criteria.CallCriteria) (string, bool) {
	e, ok := callCriteria.Validations[validationName]
	if ok {
		for _, r := range e.TagRegexp {
			found := r.FindStringSubmatch(tag)
			if len(found) > 0 {
				return found[0], true
			}
		}
	}
	return "", false
}
