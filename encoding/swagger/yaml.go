package swagger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
)

var escapePropNameRegExp = regexp.MustCompile("^[a-zA-Z_]+$")

// MarshalYAML marshals a Swagger definition to YAML
func MarshalYAML(swagger openapi2.Swagger, w io.Writer) error {
	ew := &errorWriter{w: w}
	writeStringProp("swagger", "\"2.0\"", 0, ew)
	if !isEmptyInfo(swagger.Info) {
		marshalInfo(swagger.Info, ew)
	}
	if !isEmptyExternalDocs(swagger.ExternalDocs) {
		marshalExternalDocs(swagger.ExternalDocs, 0, ew)
	}
	if !isEmptyStrSlice(swagger.Schemes) {
		writeStrSlice("schemes", swagger.Schemes, 0, ew)
	}
	if !isEmptyString(swagger.Host) {
		writeStringProp("host", swagger.Host, 0, ew)
	}
	if !isEmptyString(swagger.BasePath) {
		writeStringProp("basePath", swagger.BasePath, 0, ew)
	}
	if !isEmptyParameters(swagger.Parameters) {
		writeObject("parameters", 0, ew)
		for def, param := range swagger.Parameters {
			writeObject(def, 1, ew)
			marshalParameter(param, 2, ew)
		}
	}
	if !isEmptyPaths(swagger.Paths) {
		writeObject("paths", 0, ew)
		for url, pathItem := range swagger.Paths {
			writeObject(url, 1, ew)
			marshalPath(pathItem, ew)
		}
	}
	if !isEmptyDefinitions(swagger.Definitions) {
		writeObject("definitions", 0, ew)
		for def, schemaRef := range swagger.Definitions {
			writeObject(def, 1, ew)
			marshalSchemaRef(schemaRef, 2, ew)
		}
	}
	if !isEmptyResponses(swagger.Responses) {
		writeObject("responses", 0, ew)
		for code, resp := range swagger.Responses {
			marshalResponse(code, resp, 1, ew)
		}
	}
	if !isEmptySecurityDefinitions(swagger.SecurityDefinitions) {
		writeObject("securityDefinitions", 0, ew)
		for name, scheme := range swagger.SecurityDefinitions {
			marshalSecurityScheme(name, scheme, 1, ew)
		}
	}
	if !isEmptySecurity(swagger.Security) {
		marshalInterface("security", swagger.Security, 0, ew)
	}
	if !isEmptyTags(swagger.Tags) {
		marshalTags(swagger.Tags, 0, ew)
	}
	return ew.err
}

func writeLn(line string, indent int, ew *errorWriter) {
	for i := 0; i < indent; i++ {
		ew.Write([]byte("  "))
	}
	ew.Write([]byte(line))
	ew.Write([]byte("\n"))
}

func escapePropName(name string) string {
	escapedName := name
	if !escapePropNameRegExp.MatchString(name) {
		escapedName = "\"" + strings.ReplaceAll(name, "\"", "\\\"") + "\""
	}
	return escapedName
}

func writeRawStringProp(name, value string, indent int, ew *errorWriter) {
	hasNewLine := strings.Contains(value, "\n")
	if hasNewLine {
		writeLn(name+": >", indent, ew)
		for _, v := range strings.Split(value, "\n") {
			writeLn(v, indent+1, ew)
		}
	} else {
		writeLn(fmt.Sprintf("%s: %s", name, value), indent, ew)
	}
}

func writeStringProp(name, value string, indent int, ew *errorWriter) {
	escapedName := escapePropName(name)
	writeRawStringProp(escapedName, value, indent, ew)
}

func writeStrSlice(name string, values []string, indent int, ew *errorWriter) {
	// TODO: if any char has any newline (which shouldn't) this func will break the yaml format
	escapedName := escapePropName(name)
	writeLn(escapedName+":", indent, ew)
	for _, v := range values {
		writeLn(fmt.Sprintf("- %s", v), indent+1, ew)
	}
}

func writeArrStringProp(name, value string, indent int, ew *errorWriter) {
	writeRawStringProp("- "+name, value, indent, ew)
}

func writeFloat64Prop(name string, value float64, indent int, ew *errorWriter) {
	escapedName := escapePropName(name)
	writeLn(fmt.Sprintf("%s: %f.ff", escapedName, value), indent, ew)
}

func writeUint64Prop(name string, value uint64, indent int, ew *errorWriter) {
	escapedName := escapePropName(name)
	writeLn(fmt.Sprintf("%s: %d", escapedName, value), indent, ew)
}

func writeBoolProp(name string, value bool, indent int, ew *errorWriter) {
	escapedName := escapePropName(name)
	writeLn(fmt.Sprintf("%s: %v", escapedName, value), indent, ew)
}

func writeObject(name string, indent int, ew *errorWriter) {
	escapedName := escapePropName(name)
	writeLn(escapedName+":", indent, ew)
}

func marshalWithYAML(data map[string]interface{}, indent int, ew *errorWriter) {
	if ew.err == nil {
		marshaled, err := yaml.Marshal(data)
		if err != nil {
			ew.err = err
			return
		}
		s := bufio.NewScanner(bytes.NewReader(marshaled))
		for s.Scan() {
			writeLn(s.Text(), indent, ew)
		}
		if err := s.Err(); err != nil {
			ew.err = err
		}
	}
}

func marshalInterface(name string, generic interface{}, indent int, ew *errorWriter) {
	data := make(map[string]interface{})
	data[name] = generic
	marshalWithYAML(data, indent, ew)
}

func marshalInfo(info openapi3.Info, ew *errorWriter) {
	writeObject("info", 0, ew)
	if !isEmptyString(info.Title) {
		writeStringProp("title", info.Title, 1, ew)
	}
	if !isEmptyString(info.Description) {
		writeStringProp("description", info.Description, 1, ew)
	}
	if !isEmptyString(info.TermsOfService) {
		writeStringProp("termsOfService", info.TermsOfService, 1, ew)
	}
	if !isEmptyString(info.Version) {
		writeStringProp("version", info.Version, 1, ew)
	}
	if !isEmptyContactPtr(info.Contact) {
		writeObject("contact", 1, ew)
		marshalContactPtr(info.Contact, ew)
	}
	if !isEmptyLicensePtr(info.License) {
		writeObject("license", 1, ew)
		marshalLicensePtr(info.License, ew)
	}
}

func marshalContactPtr(contact *openapi3.Contact, ew *errorWriter) {
	if !isEmptyString(contact.Name) {
		writeStringProp("name", contact.Name, 1, ew)
	}
	if !isEmptyString(contact.URL) {
		writeStringProp("url", contact.URL, 1, ew)
	}
	if !isEmptyString(contact.Email) {
		writeStringProp("email", contact.Email, 1, ew)
	}
}

func marshalLicensePtr(license *openapi3.License, ew *errorWriter) {
	if !isEmptyString(license.Name) {
		writeStringProp("name", license.Name, 1, ew)
	}
	if !isEmptyString(license.URL) {
		writeStringProp("url", license.URL, 1, ew)
	}
}

func marshalPath(pathItem *openapi2.PathItem, ew *errorWriter) {
	if !isEmptyString(pathItem.Ref) {
		writeStringProp("$ref", pathItem.Ref, 2, ew)
	}
	if pathItem.Delete != nil {
		writeObject("delete", 2, ew)
		marshalOperation(pathItem.Delete, ew)
	} else if pathItem.Get != nil {
		writeObject("get", 2, ew)
		marshalOperation(pathItem.Get, ew)
	} else if pathItem.Head != nil {
		writeObject("head", 2, ew)
		marshalOperation(pathItem.Head, ew)
	} else if pathItem.Options != nil {
		writeObject("options", 2, ew)
		marshalOperation(pathItem.Options, ew)
	} else if pathItem.Patch != nil {
		writeObject("patch", 2, ew)
		marshalOperation(pathItem.Patch, ew)
	} else if pathItem.Post != nil {
		writeObject("post", 2, ew)
		marshalOperation(pathItem.Post, ew)
	} else if pathItem.Put != nil {
		writeObject("put", 2, ew)
		marshalOperation(pathItem.Put, ew)
	}
	if !isEmptyPathItemParameters(pathItem.Parameters) {
		writeObject("parameters", 2, ew)
		for _, p := range pathItem.Parameters {
			marshalParameterArr(p, 3, ew)
		}
	}
}

func marshalOperation(operation *openapi2.Operation, ew *errorWriter) {
	if !isEmptyString(operation.Summary) {
		writeStringProp("summary", operation.Summary, 3, ew)
	}
	if !isEmptyString(operation.Description) {
		writeStringProp("description", operation.Description, 3, ew)
	}
	if !isEmptyString(operation.OperationID) {
		writeStringProp("operationid", operation.OperationID, 3, ew)
	}
	if !isEmptyStrSlice(operation.Consumes) {
		writeStrSlice("consumes", operation.Consumes, 3, ew)
	}
	if !isEmptyStrSlice(operation.Produces) {
		writeStrSlice("produces", operation.Produces, 3, ew)
	}
	if !isEmptyStrSlice(operation.Tags) {
		writeStrSlice("tags", operation.Tags, 3, ew)
	}
	if !isEmptyExternalDocs(operation.ExternalDocs) {
		marshalExternalDocs(operation.ExternalDocs, 3, ew)
	}
	if !isEmptyPathItemParameters(operation.Parameters) {
		writeObject("parameters", 3, ew)
		for _, p := range operation.Parameters {
			marshalParameterArr(p, 4, ew)
		}
	}
	if !isEmptyResponses(operation.Responses) {
		writeObject("responses", 3, ew)
		keys := make([]string, 0, len(operation.Responses))
		for k := range operation.Responses {
			keys = append(keys, k)
		}
		// I could have inserted each key sorted...but well
		sort.Strings(keys)
		for _, code := range keys {
			marshalResponse(code, operation.Responses[code], 4, ew)
		}
	}
	if !isEmptySecurityPtr(operation.Security) {
		marshalInterface("security", *operation.Security, 3, ew)
	}
}

func marshalExternalDocs(externalDocs *openapi3.ExternalDocs, indent int, ew *errorWriter) {
	writeObject("externalDocs", indent, ew)
	if !isEmptyString(externalDocs.Description) {
		writeStringProp("description", externalDocs.Description, indent+1, ew)
	}
	if !isEmptyString(externalDocs.URL) {
		writeStringProp("url", externalDocs.URL, indent+1, ew)
	}
}

func marshalParameter(parameter *openapi2.Parameter, indent int, ew *errorWriter) {
	if !isEmptyString(parameter.In) {
		writeStringProp("in", parameter.In, indent, ew)
	}
	if !isEmptyString(parameter.Name) {
		writeStringProp("name", parameter.Name, indent, ew)
	}
	if !isEmptyString(parameter.Ref) {
		writeStringProp("$ref", parameter.Ref, indent, ew)
	}
	if !isEmptyString(parameter.Type) {
		writeStringProp("type", parameter.Type, indent, ew)
	}
	if !isEmptyString(parameter.Format) {
		writeStringProp("format", parameter.Format, indent, ew)
	}
	if !isEmptyString(parameter.Description) {
		writeStringProp("description", parameter.Description, indent, ew)
	}
	if !isEmptyString(parameter.Pattern) {
		writeStringProp("pattern", parameter.Pattern, indent, ew)
	}
	writeBoolProp("required", parameter.Required, indent, ew)
	if parameter.UniqueItems {
		writeBoolProp("uniqueItems", parameter.UniqueItems, indent, ew)
	}
	if parameter.ExclusiveMin {
		writeBoolProp("exclusiveMinimum", parameter.ExclusiveMin, indent, ew)
	}
	if parameter.ExclusiveMax {
		writeBoolProp("exclusiveMaximum", parameter.ExclusiveMax, indent, ew)
	}
	if !isEmptySchemaRef(parameter.Schema) {
		writeObject("schema", indent, ew)
		marshalSchemaRef(parameter.Schema, indent+1, ew)
	}
	if !isEmptyInterfaceSlice(parameter.Enum) {
		marshalInterface("enum", parameter.Enum, indent, ew)
	}
	if !isEmptyFloat64Ptr(parameter.Minimum) {
		writeFloat64Prop("minimum", *parameter.Minimum, indent, ew)
	}
	if !isEmptyFloat64Ptr(parameter.Maximum) {
		writeFloat64Prop("maximum", *parameter.Maximum, indent, ew)
	}
	if !isEmptyUint64(parameter.MinLength) {
		writeUint64Prop("minLength", parameter.MinLength, indent, ew)
	}
	if !isEmptyUint64Ptr(parameter.MaxLength) {
		writeUint64Prop("maxLength", *parameter.MaxLength, indent, ew)
	}
	if !isEmptySchemaRef(parameter.Items) {
		writeObject("items", indent, ew)
		marshalSchemaRef(parameter.Items, indent+1, ew)
	}
	if !isEmptyUint64(parameter.MinItems) {
		writeUint64Prop("minItems", parameter.MinItems, indent, ew)
	}
	if !isEmptyUint64Ptr(parameter.MaxItems) {
		writeUint64Prop("maxItems", *parameter.MaxItems, indent, ew)
	}
}

func marshalParameterArr(parameter *openapi2.Parameter, indent int, ew *errorWriter) {
	// in is a required property so it MUST be present
	writeArrStringProp("in", parameter.In, indent, ew)
	indent++
	if !isEmptyString(parameter.Name) {
		writeStringProp("name", parameter.Name, indent, ew)
	}
	if !isEmptyString(parameter.Ref) {
		writeStringProp("$ref", parameter.Ref, indent, ew)
	}
	if !isEmptyString(parameter.Type) {
		writeStringProp("type", parameter.Type, indent, ew)
	}
	if !isEmptyString(parameter.Format) {
		writeStringProp("format", parameter.Format, indent, ew)
	}
	if !isEmptyString(parameter.Description) {
		writeStringProp("description", parameter.Description, indent, ew)
	}
	if !isEmptyString(parameter.Pattern) {
		writeStringProp("pattern", parameter.Pattern, indent, ew)
	}
	writeBoolProp("required", parameter.Required, indent, ew)
	if parameter.UniqueItems {
		writeBoolProp("uniqueItems", parameter.UniqueItems, indent, ew)
	}
	if parameter.ExclusiveMin {
		writeBoolProp("exclusiveMinimum", parameter.ExclusiveMin, indent, ew)
	}
	if parameter.ExclusiveMax {
		writeBoolProp("exclusiveMaximum", parameter.ExclusiveMax, indent, ew)
	}
	if !isEmptySchemaRef(parameter.Schema) {
		writeObject("schema", indent, ew)
		marshalSchemaRef(parameter.Schema, indent+1, ew)
	}
	if !isEmptyInterfaceSlice(parameter.Enum) {
		marshalInterface("enum", parameter.Enum, indent, ew)
	}
	if !isEmptyFloat64Ptr(parameter.Minimum) {
		writeFloat64Prop("minimum", *parameter.Minimum, indent, ew)
	}
	if !isEmptyFloat64Ptr(parameter.Maximum) {
		writeFloat64Prop("maximum", *parameter.Maximum, indent, ew)
	}
	if !isEmptyUint64(parameter.MinLength) {
		writeUint64Prop("minLength", parameter.MinLength, indent, ew)
	}
	if !isEmptyUint64Ptr(parameter.MaxLength) {
		writeUint64Prop("maxLength", *parameter.MaxLength, indent, ew)
	}
	if !isEmptySchemaRef(parameter.Items) {
		writeObject("items", indent, ew)
		marshalSchemaRef(parameter.Items, indent+1, ew)
	}
	if !isEmptyUint64(parameter.MinItems) {
		writeUint64Prop("minItems", parameter.MinItems, indent, ew)
	}
	if !isEmptyUint64Ptr(parameter.MaxItems) {
		writeUint64Prop("maxItems", *parameter.MaxItems, indent, ew)
	}
}

func marshalResponse(code string, response *openapi2.Response, indent int, ew *errorWriter) {
	writeObject(code, indent, ew)
	if !isEmptyString(response.Ref) {
		writeStringProp("$ref", response.Ref, indent+1, ew)
	}
	if !isEmptyString(response.Description) {
		writeStringProp("description", response.Description, indent+1, ew)
	}
	if !isEmptySchemaRef(response.Schema) {
		writeObject("schema", indent+1, ew)
		marshalSchemaRef(response.Schema, indent+2, ew)
	}
	if !isEmptyHeaders(response.Headers) {
		marshalHeaders(response.Headers, indent+1, ew)
	}
	if !isEmptyInterfaceMap(response.Examples) {
		marshalInterface("examples", response.Examples, indent+1, ew)
	}

}

func marshalHeaders(headers map[string]*openapi2.Header, indent int, ew *errorWriter) {
	writeObject("headers", indent, ew)
	for name, header := range headers {
		writeObject(name, indent+1, ew)
		if !isEmptyString(header.Ref) {
			writeStringProp("$ref", header.Ref, indent+2, ew)
		}
		if !isEmptyString(header.Description) {
			writeStringProp("description", header.Description, indent+2, ew)
		}
		if !isEmptyString(header.Type) {
			writeStringProp("type", header.Type, indent+2, ew)
		}
	}
}

func marshalSecurityScheme(name string, scheme *openapi2.SecurityScheme, indent int, ew *errorWriter) {
	writeObject(name, indent, ew)
	if !isEmptyString(scheme.Ref) {
		writeStringProp("$ref", scheme.Ref, indent+1, ew)
	}
	if !isEmptyString(scheme.Description) {
		writeStringProp("description", scheme.Description, indent+1, ew)
	}
	if !isEmptyString(scheme.Type) {
		writeStringProp("type", scheme.Type, indent+1, ew)
	}
	if !isEmptyString(scheme.In) {
		writeStringProp("in", scheme.In, indent+1, ew)
	}
	if !isEmptyString(scheme.Name) {
		writeStringProp("name", scheme.Name, indent+1, ew)
	}
	if !isEmptyString(scheme.Flow) {
		writeStringProp("flow", scheme.Flow, indent+1, ew)
	}
	if !isEmptyString(scheme.AuthorizationURL) {
		writeStringProp("authorizationUrl", scheme.AuthorizationURL, indent+1, ew)
	}
	if !isEmptyString(scheme.TokenURL) {
		writeStringProp("tokenUrl", scheme.TokenURL, indent+1, ew)
	}
	if !isEmptyStringMap(scheme.Scopes) {
		marshalInterface("scopes", scheme.Scopes, indent+1, ew)
	}
	if !isEmptyTags(scheme.Tags) {
		marshalTags(scheme.Tags, indent+1, ew)
	}
}

func marshalSchemaRef(schemaRef *openapi3.SchemaRef, indent int, ew *errorWriter) {
	if !isEmptyString(schemaRef.Ref) {
		writeStringProp("$ref", schemaRef.Ref, indent, ew)
	} else if schemaRef.Value != nil {
		marshalSchema(schemaRef.Value, indent, ew)
	}
}

func marshalSchemaRefArr(schemaRef *openapi3.SchemaRef, indent int, ew *errorWriter) {
	if !isEmptyString(schemaRef.Ref) {
		writeArrStringProp("$ref", schemaRef.Ref, indent, ew)
	} else if schemaRef.Value != nil {
		marshalSchema(schemaRef.Value, indent, ew)
	}
}

func marshalSchema(schema *openapi3.Schema, indent int, ew *errorWriter) {
	if !isEmptyString(schema.Title) {
		writeStringProp("title", schema.Title, indent, ew)
	}
	if !isEmptyString(schema.Format) {
		writeStringProp("format", schema.Format, indent, ew)
	}
	if !isEmptyString(schema.Type) {
		writeStringProp("type", schema.Type, indent, ew)
	}
	if !isEmptyString(schema.Description) {
		writeStringProp("description", schema.Description, indent, ew)
	}
	if !isEmptyInterface(schema.Default) {
		marshalInterface("default", schema.Default, indent, ew)
	}
	if !isEmptyString(schema.Pattern) {
		writeStringProp("pattern", schema.Pattern, indent, ew)
	}
	if !isEmptyInterfaceSlice(schema.Enum) {
		marshalInterface("enum", schema.Enum, indent, ew)
	}
	if !isEmptyFloat64Ptr(schema.MultipleOf) {
		writeFloat64Prop("multipleOf", *schema.MultipleOf, indent, ew)
	}
	if !isEmptyFloat64Ptr(schema.Max) {
		writeFloat64Prop("maximum", *schema.Max, indent, ew)
	}
	if schema.ExclusiveMax {
		writeBoolProp("exclusiveMaximum", schema.ExclusiveMax, indent, ew)
	}
	if !isEmptyFloat64Ptr(schema.Min) {
		writeFloat64Prop("minimum", *schema.Min, indent, ew)
	}
	if schema.ExclusiveMin {
		writeBoolProp("exclusiveMinimum", schema.ExclusiveMin, indent, ew)
	}
	if !isEmptyUint64Ptr(schema.MaxLength) {
		writeUint64Prop("maxLength", *schema.MaxLength, indent, ew)
	}
	if !isEmptyUint64(schema.MinLength) {
		writeUint64Prop("minLength", schema.MinLength, indent, ew)
	}
	if !isEmptyUint64Ptr(schema.MaxItems) {
		writeUint64Prop("maxItems", *schema.MaxItems, indent, ew)
	}
	if !isEmptyUint64(schema.MinItems) {
		writeUint64Prop("minItems", schema.MinItems, indent, ew)
	}
	if schema.UniqueItems {
		writeBoolProp("uniqueItems", schema.UniqueItems, indent, ew)
	}
	if !isEmptyUint64Ptr(schema.MaxProps) {
		writeUint64Prop("maxProperties", *schema.MaxProps, indent, ew)
	}
	if !isEmptyUint64(schema.MinProps) {
		writeUint64Prop("minProperties", schema.MinProps, indent, ew)
	}
	if !isEmptySchemaRef(schema.Items) {
		writeObject("items", indent, ew)
		marshalSchemaRef(schema.Items, indent+1, ew)
	}
	if !isEmptySchemaRefSlice(schema.AllOf) {
		writeObject("allOf", indent, ew)
		for _, allOf := range schema.AllOf {
			marshalSchemaRefArr(allOf, indent+1, ew)
		}
	}
	if !isEmptySchemaRefMap(schema.Properties) {
		writeObject("properties", indent, ew)
		for name, property := range schema.Properties {
			writeObject(name, indent+1, ew)
			marshalSchemaRef(property, indent+2, ew)
		}
	}
	if !isEmptySchemaRef(schema.AdditionalProperties) {
		writeObject("additionalProperties", indent, ew)
		marshalSchemaRef(schema.AdditionalProperties, indent+1, ew)
	}
	if !isEmptyDiscriminator(schema.Discriminator) {
		marshalDiscriminator(schema.Discriminator, indent, ew)
	}
	if schema.ReadOnly {
		writeBoolProp("readOnly", schema.ReadOnly, indent, ew)
	}
	if !isEmptyInterface(schema.XML) {
		marshalInterface("xml", schema.XML, indent, ew)
	}
	if !isEmptyExternalDocs(schema.ExternalDocs) {
		marshalExternalDocs(schema.ExternalDocs, indent, ew)
	}
	if !isEmptyInterface(schema.Example) {
		marshalInterface("example", schema.Example, indent, ew)
	}
	if !isEmptyStrSlice(schema.Required) {
		writeStrSlice("required", schema.Required, indent, ew)
	}
}

// func marshalSchemaArr(schema *openapi3.Schema, indent int, ew *errorWriter) {
// 	// tipe is required, so it must exists
// 	writeArrStringProp("title", schema.Title, indent, ew)
// 	indent++
// 	if !isEmptyString(schema.Format) {
// 		writeStringProp("format", schema.Format, indent, ew)
// 	}
// 	if !isEmptyString(schema.Type) {
// 		writeStringProp("type", schema.Type, indent, ew)
// 	}
// 	if !isEmptyString(schema.Description) {
// 		writeStringProp("description", schema.Description, indent, ew)
// 	}
// 	if !isEmptyInterface(schema.Default) {
// 		marshalInterface("default", schema.Default, indent, ew)
// 	}
// 	if !isEmptyString(schema.Pattern) {
// 		writeStringProp("pattern", schema.Pattern, indent, ew)
// 	}
// 	if !isEmptyInterfaceSlice(schema.Enum) {
// 		marshalInterface("enum", schema.Enum, indent, ew)
// 	}
// 	if !isEmptyFloat64Ptr(schema.MultipleOf) {
// 		writeFloat64Prop("multipleOf", *schema.MultipleOf, indent, ew)
// 	}
// 	if !isEmptyFloat64Ptr(schema.Max) {
// 		writeFloat64Prop("maximum", *schema.Max, indent, ew)
// 	}
// 	if schema.ExclusiveMax {
// 		writeBoolProp("exclusiveMaximum", schema.ExclusiveMax, indent, ew)
// 	}
// 	if !isEmptyFloat64Ptr(schema.Min) {
// 		writeFloat64Prop("minimum", *schema.Min, indent, ew)
// 	}
// 	if schema.ExclusiveMin {
// 		writeBoolProp("exclusiveMinimum", schema.ExclusiveMin, indent, ew)
// 	}
// 	if !isEmptyUint64Ptr(schema.MaxLength) {
// 		writeUint64Prop("maxLength", *schema.MaxLength, indent, ew)
// 	}
// 	if !isEmptyUint64(schema.MinLength) {
// 		writeUint64Prop("minLength", schema.MinLength, indent, ew)
// 	}
// 	if !isEmptyUint64Ptr(schema.MaxItems) {
// 		writeUint64Prop("maxItems", *schema.MaxItems, indent, ew)
// 	}
// 	if !isEmptyUint64(schema.MinItems) {
// 		writeUint64Prop("minItems", schema.MinItems, indent, ew)
// 	}
// 	if schema.UniqueItems {
// 		writeBoolProp("uniqueItems", schema.UniqueItems, indent, ew)
// 	}
// 	if !isEmptyUint64Ptr(schema.MaxProps) {
// 		writeUint64Prop("maxProperties", *schema.MaxProps, indent, ew)
// 	}
// 	if !isEmptyUint64(schema.MinProps) {
// 		writeUint64Prop("minProperties", schema.MinProps, indent, ew)
// 	}
// 	if !isEmptySchemaRef(schema.Items) {
// 		writeObject("items", indent, ew)
// 		marshalSchemaRef(schema.Items, indent+1, ew)
// 	}
// 	if !isEmptySchemaRefSlice(schema.AllOf) {
// 		writeObject("allOf", indent, ew)
// 		for _, allOf := range schema.AllOf {
// 			marshalSchemaRefArr(allOf, indent+1, ew)
// 		}
// 	}
// 	if !isEmptySchemaRefMap(schema.Properties) {
// 		writeObject("properties", indent, ew)
// 		for name, property := range schema.Properties {
// 			writeObject(name, indent+1, ew)
// 			marshalSchemaRef(property, indent+2, ew)
// 		}
// 	}
// 	if !isEmptySchemaRef(schema.AdditionalProperties) {
// 		writeObject("additionalProperties", indent, ew)
// 		marshalSchemaRef(schema.AdditionalProperties, indent+1, ew)
// 	}
// 	if !isEmptyDiscriminator(schema.Discriminator) {
// 		marshalDiscriminator(schema.Discriminator, indent, ew)
// 	}
// 	if schema.ReadOnly {
// 		writeBoolProp("readOnly", schema.ReadOnly, indent, ew)
// 	}
// 	if !isEmptyInterface(schema.XML) {
// 		marshalInterface("xml", schema.XML, indent, ew)
// 	}
// 	if !isEmptyExternalDocs(schema.ExternalDocs) {
// 		marshalExternalDocs(schema.ExternalDocs, indent, ew)
// 	}
// 	if !isEmptyInterface(schema.Example) {
// 		marshalInterface("example", schema.Example, indent, ew)
// 	}
// 	if !isEmptyStrSlice(schema.Required) {
// 		writeStrSlice("required", schema.Required, indent, ew)
// 	}
// }

func marshalDiscriminator(discriminator *openapi3.Discriminator, indent int, ew *errorWriter) {
	writeObject("discriminator", indent, ew)
	if !isEmptyString(discriminator.PropertyName) {
		writeStringProp("propertyName", discriminator.PropertyName, indent+1, ew)
	}
	if !isEmptyStringMap(discriminator.Mapping) {
		marshalInterface("mapping", discriminator.Mapping, indent+1, ew)
	}
}

func marshalTags(tags []*openapi3.Tag, indent int, ew *errorWriter) {
	writeObject("tags", indent, ew)
	for _, tag := range tags {
		// name is a required property for tags, so it must exists
		writeArrStringProp("name", tag.Name, indent+1, ew)
		if !isEmptyString(tag.Description) {
			writeStringProp("description", tag.Description, indent+1, ew)
		}
		if !isEmptyExternalDocs(tag.ExternalDocs) {
			marshalExternalDocs(tag.ExternalDocs, indent+1, ew)
		}
	}
}
