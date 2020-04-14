package criteria

import (
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v2"
)

// ParserErr is returned when there is an error parsing a criteria
type ParserErr string

func (p ParserErr) Error() string {
	return string(p)
}

const (
	// ErrMissingRoutes is returned when a Criteria does not have any route criteria
	ErrMissingRoutes ParserErr = "missing routes matching criteria array"
	// ErrMissingRequest is returned when a Criteria does not have any request criteria
	ErrMissingRequest ParserErr = "missing request matching criteria array"
	// ErrMissingResponse is returned when a Criteria does not have any response criteria
	ErrMissingResponse ParserErr = "missing response matching criteria array"
	// ErrInvalidRoute is returned when a Criteria contains an invalid route criteria
	ErrInvalidRoute ParserErr = "invalid route criteria"
	// MIMEApplicationJSON is the application/json mime
	MIMEApplicationJSON = "application/json"
	// RequiredValidation is the swagger required validation
	RequiredValidation = "required"
	// ExclusiveMinValidation is the swagger exclusiveMin validation
	ExclusiveMinValidation = "exclusiveMin"
	// ExclusiveMaxValidation is the swagger exclusiveMax validation
	ExclusiveMaxValidation = "exclusiveMax"
	// EnumValidation is the swagger enum validation
	EnumValidation = "enum"
	// MinimumValidation is the swagger minimum validation
	MinimumValidation = "minimum"
	// MaximumValidation is the swagger maximum validation
	MaximumValidation = "maximum"
	// MinLengthValidation is the swagger minLength validation
	MinLengthValidation = "minLength"
	// MaxLengthValidation is the swagger maxLength validation
	MaxLengthValidation = "maxLength"
	// PatternValidation is the swagger pattern validation
	PatternValidation = "pattern"
	// ErrInvalidCallCriteria is returned when a Criteria contains an invalid callCriteria
	ErrInvalidCallCriteria ParserErr = "invalid response criteria"
	requestCallCriteria    string    = "request"
	responseCallCriteria   string    = "response"
)

var (
	defaultURLNamedPathVarExtractor = regexp.MustCompile("\\{([a-zA-Z0-9]+)\\}")
	// httpMethods
	httpMethods = [...]string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
)

// Criteria contains all the information to match a Handler, a request Parser and a Response marshaler
type Criteria struct {
	DefinitionPrefix    string                              `yaml:"definitionPrefix"`
	BasePath            string                              `yaml:"basePath"`
	Host                string                              `yaml:"host"`
	Info                Info                                `yaml:"info"`
	Parameters          map[string]*openapi2.Parameter      `yaml:"parameters,omitempty"`
	SecurityDefinitions map[string]*openapi2.SecurityScheme `yaml:"securityDefinitions,omitempty"`
	Routes              []RouteCriteria                     `yaml:"routes"`
	Request             []CallCriteria                      `yaml:"request"`
	Response            []CallCriteria                      `yaml:"response"`
	StaticModels        map[string]*openapi3.Schema         `yaml:"staticModels"`
	VendorFolders       []string                            `yaml:"vendorFolders"`
}

// Info is the info swagger mapping
type Info struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// FuncRoute matches a route that is defined as a function call
type FuncRoute struct {
	FuncName                    string         `yaml:"funcName"`
	Pkg                         string         `yaml:"pkg"`
	HTTPMethod                  string         `yaml:"httpMethod"`
	NamedPathVarExtractor       string         `yaml:"namedPathVarExtractor"`
	NamedPathVarExtractorRegexp *regexp.Regexp `yaml:"-"`
	PathIndex                   int            `yaml:"pathIndex"`
	HandlerIndex                int            `yaml:"handlerIndex"`
	ChildRoute                  *FuncRoute     `yaml:"childRoute,omitempty"`
}

// ParameterMatcher matches parameters in a struct route
type ParameterMatcher struct {
	Always        bool           `yaml:"always"`
	Field         string         `yaml:"field"`
	Matches       string         `yaml:"matches"`
	MatchesRegExp *regexp.Regexp `yaml:"-"`
}

// StructRoute matches a route that is defined as a struct
type StructRoute struct {
	Name                        string                      `yaml:"name"`
	Pkg                         string                      `yaml:"pkg"`
	PathField                   string                      `yaml:"pathField"`
	NamedPathVarExtractor       string                      `yaml:"namedPathVarExtractor"`
	NamedPathVarExtractorRegexp *regexp.Regexp              `yaml:"-"`
	HandlerField                string                      `yaml:"handlerField"`
	HTTPMethodField             string                      `yaml:"httpMethodField"`
	Parameters                  map[string]ParameterMatcher `yaml:"parameters"`
	Security                    map[string]ParameterMatcher `yaml:"security"`
}

// RouteCriteria contains all the information to find a Route declaration
type RouteCriteria struct {
	StructRoute *StructRoute `yaml:"structRoute"`
	FuncRoute   *FuncRoute   `yaml:"funcRoute"`
}

// ValidationExtractor are slices of regular expression that matches validations
type ValidationExtractor struct {
	Validation string `yaml:"validation"`
	// Tag        []string         `yaml:"tag"`
	TagRegexp []*regexp.Regexp `yaml:"-"`
}

// ModelExtractor defines where to extract a model
type ModelExtractor struct {
	ParamIndex int    `yaml:"paramIndex"`
	Name       string `yaml:"name"`
}

// CallCriteria contains all the information to match a function call with an argument
type CallCriteria struct {
	Pkg            string                         `yaml:"pkg"`
	FuncName       string                         `yaml:"funcName"`
	ModelExtractor ModelExtractor                 `yaml:"modelExtractor"`
	CodeIndex      int                            `yaml:"codeIndex"`
	Validations    map[string]ValidationExtractor `yaml:"validations"`
	Consumes       string                         `yaml:"consumes"`
	Produces       string                         `yaml:"produces"`
}

// Decoder is able to decode and validate a Criteria
type Decoder struct {
	Logger *log.Logger
}

// ParseCriteriaFromYAML parses a Criteria from a YAML reader
func (decoder Decoder) ParseCriteriaFromYAML(r io.Reader, c *Criteria) error {
	decoder.Logger.Printf("parsing criteria from reader\n")
	err := yaml.NewDecoder(r).Decode(c)
	if err != nil {
		decoder.Logger.Printf("error decoding criteria from reader: %v\n", err)
		return err
	}
	for i := range c.Routes {
		if c.Routes[i].StructRoute != nil {
			namedPathVarExtractor := defaultURLNamedPathVarExtractor
			if len(c.Routes[i].StructRoute.NamedPathVarExtractor) > 0 {
				namedPathVarExtractor, err = regexp.Compile(c.Routes[i].StructRoute.NamedPathVarExtractor)
				if err != nil {
					return err
				}
			}
			c.Routes[i].StructRoute.NamedPathVarExtractorRegexp = namedPathVarExtractor
			if c.Routes[i].StructRoute.Parameters != nil {
				for k, v := range c.Routes[i].StructRoute.Parameters {
					if len(v.Matches) > 0 && !v.Always {
						matchesRegexp, err := regexp.Compile(v.Matches)
						if err != nil {
							return err
						}
						v.MatchesRegExp = matchesRegexp
						c.Routes[i].StructRoute.Parameters[k] = v
					}
				}
				// TODO: Extract this to a function
			}
			if c.Routes[i].StructRoute.Security != nil {
				for k, v := range c.Routes[i].StructRoute.Security {
					if len(v.Matches) > 0 && !v.Always {
						matchesRegexp, err := regexp.Compile(v.Matches)
						if err != nil {
							return err
						}
						v.MatchesRegExp = matchesRegexp
						c.Routes[i].StructRoute.Security[k] = v
					}
				}
			}
		} else if c.Routes[i].FuncRoute != nil {
			namedPathVarExtractor := defaultURLNamedPathVarExtractor
			if len(c.Routes[i].FuncRoute.NamedPathVarExtractor) > 0 {
				namedPathVarExtractor, err = regexp.Compile(c.Routes[i].FuncRoute.NamedPathVarExtractor)
				if err != nil {
					return err
				}
			}
			c.Routes[i].FuncRoute.NamedPathVarExtractorRegexp = namedPathVarExtractor
		}
	}
	for i := range c.Request {
		if c.Request[i].Validations != nil {
			for k, v := range c.Request[i].Validations {
				if len(v.Validation) > 0 {
					tagRegexp, err := regexp.Compile(v.Validation)
					if err != nil {
						decoder.Logger.Printf("error processing validation %s, could not compile regexp '%s': %v", k, v.Validation, err)
					}
					v.TagRegexp = []*regexp.Regexp{tagRegexp}
					c.Request[i].Validations[k] = v
				}
			}
		}
	}
	return nil
}

// NewCriteriaDecoder creates a CriteriaDecoder
func NewCriteriaDecoder(logger *log.Logger) Decoder {
	return Decoder{
		Logger: logger,
	}
}

// MatchesHTTPMethod returns true if text contains a known HTTP method
func MatchesHTTPMethod(text string) bool {
	return len(MatchHTTPMethod(text)) > 0
}

// MatchHTTPMethod matches an HTTP given a text
func MatchHTTPMethod(text string) string {
	httpMethodName := strings.ToUpper(text)
	for i := range httpMethods {
		m := httpMethods[i]
		if strings.Contains(httpMethodName, m) {
			return m
		}
	}
	return ""
}
