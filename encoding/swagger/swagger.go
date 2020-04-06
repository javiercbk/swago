package swagger

import (
	"io"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
)

type errorWriter struct {
	w   io.Writer
	err error
}

func (ew *errorWriter) Write(p []byte) (int, error) {
	if ew.err == nil {
		n, err := ew.w.Write(p)
		if err != nil {
			ew.err = err
		}
		return n, err
	}
	return 0, ew.err
}

func isEmptyInterface(i interface{}) bool {
	return i == nil
}

func isEmptyString(str string) bool {
	return len(str) == 0
}

func isEmptyFloat64Ptr(number *float64) bool {
	return number == nil
}

func isEmptyUint64(number uint64) bool {
	return number == 0
}

func isEmptyUint64Ptr(number *uint64) bool {
	return number == nil
}

func isEmptyStrSlice(slice []string) bool {
	return slice == nil || len(slice) == 0
}

func isEmptyInterfaceSlice(slice []interface{}) bool {
	return slice == nil || len(slice) == 0
}

func isEmptyInterfaceMap(interfaceMap map[string]interface{}) bool {
	return interfaceMap == nil || len(interfaceMap) == 0
}

func isEmptySchemaRefSlice(slice []*openapi3.SchemaRef) bool {
	return slice == nil || len(slice) == 0
}

func isEmptySchemaRefMap(schemaRefMap map[string]*openapi3.SchemaRef) bool {
	return schemaRefMap == nil || len(schemaRefMap) == 0
}

func isEmptyStringMap(strMap map[string]string) bool {
	return strMap == nil || len(strMap) == 0
}

func isEmptyInfo(info openapi3.Info) bool {
	return isEmptyString(info.Title) &&
		isEmptyString(info.Description) &&
		isEmptyString(info.TermsOfService) &&
		isEmptyString(info.Version) &&
		isEmptyContactPtr(info.Contact) &&
		isEmptyLicensePtr(info.License)
}

func isEmptyContactPtr(contact *openapi3.Contact) bool {
	return contact == nil ||
		(isEmptyString(contact.Name) &&
			isEmptyString(contact.URL) &&
			isEmptyString(contact.Email))
}

func isEmptyLicensePtr(license *openapi3.License) bool {
	return license == nil || (isEmptyString(license.Name) && isEmptyString(license.URL))
}

func isEmptyPaths(paths map[string]*openapi2.PathItem) bool {
	return paths == nil || len(paths) == 0
}

func isEmptyDefinitions(definitions map[string]*openapi3.SchemaRef) bool {
	return definitions == nil || len(definitions) == 0
}

func isEmptyParameters(parameters map[string]*openapi2.Parameter) bool {
	return parameters == nil || len(parameters) == 0
}

func isEmptyPathItemParameters(parameters openapi2.Parameters) bool {
	return parameters == nil || len(parameters) == 0
}

func isEmptyResponses(responses map[string]*openapi2.Response) bool {
	return responses == nil || len(responses) == 0
}

func isEmptySecurityDefinitions(securityDefinitions map[string]*openapi2.SecurityScheme) bool {
	return securityDefinitions == nil || len(securityDefinitions) == 0
}

func isEmptySecurity(security openapi2.SecurityRequirements) bool {
	return security == nil || len(security) == 0
}

func isEmptySecurityPtr(security *openapi2.SecurityRequirements) bool {
	return security == nil || len(*security) == 0
}

func isEmptyHeaders(headers map[string]*openapi2.Header) bool {
	return headers == nil || len(headers) == 0
}

func isEmptyTags(tags openapi3.Tags) bool {
	return tags == nil || len(tags) == 0
}

func isEmptyExternalDocs(externalDocs *openapi3.ExternalDocs) bool {
	return externalDocs == nil || (isEmptyString(externalDocs.Description) && isEmptyString(externalDocs.URL))
}

func isEmptySchemaRef(schemaRef *openapi3.SchemaRef) bool {
	return schemaRef == nil || (isEmptyString(schemaRef.Ref) && isEmptySchema(schemaRef.Value))
}

func isEmptySchema(schema *openapi3.Schema) bool {
	if schema == nil {
		return true
	}
	return isEmptyString(schema.Title) &&
		isEmptyString(schema.Format) &&
		isEmptyString(schema.Type) &&
		isEmptyString(schema.Description) &&
		isEmptyInterface(schema.Default) &&
		isEmptyString(schema.Pattern) &&
		isEmptyInterfaceSlice(schema.Enum) &&
		isEmptyFloat64Ptr(schema.MultipleOf) &&
		isEmptyFloat64Ptr(schema.Max) &&
		!schema.ExclusiveMax &&
		isEmptyFloat64Ptr(schema.Min) &&
		!schema.ExclusiveMin &&
		isEmptyUint64Ptr(schema.MaxLength) &&
		isEmptyUint64(schema.MinLength) &&
		isEmptyUint64Ptr(schema.MaxItems) &&
		isEmptyUint64(schema.MinItems) &&
		!schema.UniqueItems &&
		isEmptyUint64Ptr(schema.MaxProps) &&
		isEmptyUint64(schema.MinProps) &&
		isEmptySchemaRef(schema.Items) &&
		isEmptySchemaRefSlice(schema.AllOf) &&
		isEmptySchemaRefMap(schema.Properties) &&
		isEmptySchemaRef(schema.AdditionalProperties) &&
		isEmptyDiscriminator(schema.Discriminator) &&
		!schema.ReadOnly &&
		isEmptyInterface(schema.XML) &&
		isEmptyExternalDocs(schema.ExternalDocs) &&
		isEmptyInterface(schema.Example)
}

func isEmptyDiscriminator(discriminator *openapi3.Discriminator) bool {
	return discriminator == nil || (isEmptyString(discriminator.PropertyName) && isEmptyStringMap(discriminator.Mapping))
}
