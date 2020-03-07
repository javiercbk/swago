package criteria

import (
	"io"
	"log"
	"net/http"
	"strings"

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
	// ErrInvalidCallCriteria is returned when a Criteria contains an invalid callCriteria
	ErrInvalidCallCriteria ParserErr = "invalid response criteria"
	requestCallCriteria    string    = "request"
	responseCallCriteria   string    = "response"
)

var (
	// httpMethods
	httpMethods = [...]string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
)

// Criteria contains all the information to match a Handler, a request Parser and a Response marshaler
type Criteria struct {
	Routes   []RouteCriteria `yaml:"routes"`
	Request  []CallCriteria  `yaml:"request"`
	Response []CallCriteria  `yaml:"response"`
}

// FuncRoute matches a route that is defined as a function call
type FuncRoute struct {
	FuncName     string     `yaml:"funcName"`
	Hierarchy    string     `yaml:"hierarchy"`
	HTTPMethod   string     `yaml:"httpMethod"`
	PathIndex    int        `yaml:"pathIndex"`
	HandlerIndex int        `yaml:"handlerIndex"`
	ChildRoute   *FuncRoute `yaml:"childRoute,omitempty"`
}

// StructRoute matches a route that is defined as a struct
type StructRoute struct {
	Name            string `yaml:"name"`
	Pkg             string `yaml:"pkg"`
	PathField       string `yaml:"pathField"`
	HandlerField    string `yaml:"handlerField"`
	HTTPMethodField string `yaml:"httpMethodField"`
}

// RouteCriteria contains all the information to find a Route declaration
type RouteCriteria struct {
	StructRoute *StructRoute `yaml:"structRoute"`
	FuncRoute   *FuncRoute   `yaml:"funcRoute"`
}

// CallCriteria contains all the information to match a function call with an argument
type CallCriteria struct {
	Hierarchy  string `yaml:"hierarchy"`
	FuncName   string `yaml:"funcName"`
	ParamIndex int    `yaml:"paramIndex"`
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
	if err = decoder.ValidateCriteria(c); err != nil {
		return err
	}
	return nil
}

// ValidateCriteria returns nil if the criteria is valid otherwise it returns the validation error
func (decoder Decoder) ValidateCriteria(c *Criteria) error {
	if len(c.Routes) == 0 {
		decoder.Logger.Printf("criteria validation error: %s\n", ErrMissingRoutes.Error())
		return ErrMissingRoutes
	}
	if len(c.Request) == 0 {
		decoder.Logger.Printf("criteria validation error: %s\n", ErrMissingRequest.Error())
		return ErrMissingRequest
	}
	if len(c.Response) == 0 {
		decoder.Logger.Printf("criteria validation error: %s\n", ErrMissingResponse.Error())
		return ErrMissingResponse
	}
	var err error
	for i := range c.Routes {
		if err = decoder.validateRoute(&c.Routes[i]); err != nil {
			return err
		}
	}
	for i := range c.Request {
		if err = decoder.validateCallCriteria(&c.Request[i], requestCallCriteria); err != nil {
			return err
		}
	}
	for i := range c.Response {
		if err = decoder.validateCallCriteria(&c.Response[i], responseCallCriteria); err != nil {
			return err
		}
	}
	return nil
}

func (decoder Decoder) validateRoute(c *RouteCriteria) error {
	if c.FuncRoute != nil {
		return decoder.validateFuncRoute(c.FuncRoute)
	} else if c.StructRoute != nil {
		return decoder.validateStructRoute(c.StructRoute)
	}
	decoder.Logger.Printf("route is neither a func route nor a struct route\n")
	return ErrInvalidRoute
}

func (decoder Decoder) validateFuncRoute(c *FuncRoute) error {
	if c.FuncName == "" {
		decoder.Logger.Printf("route validation error: funcName must be a non empty string\n")
		return ErrInvalidRoute
	}
	if c.HTTPMethod != "" {
		c.HTTPMethod = strings.ToUpper(c.HTTPMethod)
		httpMethodFound := false
		for i := range httpMethods {
			if httpMethods[i] == c.HTTPMethod {
				httpMethodFound = true
				break
			}
		}
		if !httpMethodFound {
			decoder.Logger.Printf("route validation error: %s is not a valid http method\n", c.HTTPMethod)
			return ErrInvalidRoute
		}
	}
	if c.PathIndex < 0 {
		decoder.Logger.Printf("route validation error: func %s declared a negative path param index\n", c.FuncName)
		return ErrInvalidRoute
	}
	if c.HandlerIndex < 0 {
		decoder.Logger.Printf("route validation error: func %s declared a negative handler param index\n", c.FuncName)
		return ErrInvalidRoute
	}
	if c.ChildRoute != nil {
		return decoder.validateFuncRoute(c.ChildRoute)
	} else if c.PathIndex == c.HandlerIndex {
		decoder.Logger.Printf("route validation error: func %s either does not specified both path and handler, or assigned the same index for both\n", c.FuncName)
		return ErrInvalidRoute
	}
	return nil
}

func (decoder Decoder) validateStructRoute(c *StructRoute) error {
	if len(c.Name) == 0 {
		decoder.Logger.Printf("missing struct name field\n")
		return ErrInvalidRoute
	}
	if len(c.HTTPMethodField) == 0 {
		decoder.Logger.Printf("missing http method field\n")
		return ErrInvalidRoute
	}
	if len(c.HandlerField) == 0 {
		decoder.Logger.Printf("missing handler field\n")
		return ErrInvalidRoute
	}
	if len(c.PathField) == 0 {
		decoder.Logger.Printf("missing path field\n")
		return ErrInvalidRoute
	}
	return nil
}

func (decoder Decoder) validateCallCriteria(c *CallCriteria, callCriteriaName string) error {
	if c.FuncName == "" {
		decoder.Logger.Printf("%s validation error: funcName must be a non empty string\n", callCriteriaName)
		return ErrInvalidCallCriteria
	}
	if c.ParamIndex < 0 {
		decoder.Logger.Printf("%s validation error: funcName %s was given a negative param index\n", callCriteriaName, c.FuncName)
		return ErrInvalidCallCriteria
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
