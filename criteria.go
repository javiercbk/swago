package swago

import (
	"io"
	"log"
	"strings"

	"gopkg.in/yaml.v2"
)

// CriteriaParserErr is returned when there is an error parsing a criteria
type CriteriaParserErr string

func (p CriteriaParserErr) Error() string {
	return string(p)
}

const (
	// ErrMissingRoutes is returned when a Criteria does not have any route criteria
	ErrMissingRoutes CriteriaParserErr = "missing routes matching criteria array"
	// ErrMissingRequest is returned when a Criteria does not have any request criteria
	ErrMissingRequest CriteriaParserErr = "missing request matching criteria array"
	// ErrMissingResponse is returned when a Criteria does not have any response criteria
	ErrMissingResponse CriteriaParserErr = "missing response matching criteria array"
	// ErrInvalidRoute is returned when a Criteria contains an invalid route criteria
	ErrInvalidRoute CriteriaParserErr = "invalid route criteria"
	// ErrInvalidCallCriteria is returned when a Criteria contains an invalid callCriteria
	ErrInvalidCallCriteria CriteriaParserErr = "invalid response criteria"
	httpGET                string            = "GET"
	httpPOST               string            = "POST"
	httpPUT                string            = "PUT"
	httpDELETE             string            = "DELETE"
	httpPATCH              string            = "PATCH"
	requestCallCriteria    string            = "request"
	responseCallCriteria   string            = "response"
)

var (
	httpMethods = [...]string{httpGET, httpPOST, httpPUT, httpDELETE, httpPATCH}
)

// Criteria contains all the information to match a Handler, a request Parser and a Response marshaler
type Criteria struct {
	Routes   []RouteCriteria `yaml:"routes"`
	Request  []CallCriteria  `yaml:"request"`
	Response []CallCriteria  `yaml:"response"`
}

// RouteCriteria contains all the information to find a Route declaration
type RouteCriteria struct {
	Pkg          string `yaml:"pkg"`
	FuncName     string `yaml:"funcName"`
	VarType      string `yaml:"varType"`
	HTTPMethod   string `yaml:"httpMethod"`
	PathIndex    int    `yaml:"pathIndex"`
	HandlerIndex int    `yaml:"handlerIndex"`
}

// CallCriteria contains all the information to match a function call with an argument
type CallCriteria struct {
	Pkg        string `yaml:"pkg"`
	FuncName   string `yaml:"funcName"`
	VarType    string `yaml:"varType"`
	ParamIndex int    `yaml:"paramIndex"`
}

// CriteriaDecoder is able to decode and validate a Criteria
type CriteriaDecoder struct {
	Logger *log.Logger
}

// ParseCriteriaFromYAML parses a Criteria from a YAML reader
func (decoder CriteriaDecoder) ParseCriteriaFromYAML(r io.Reader, c *Criteria) error {
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
func (decoder CriteriaDecoder) ValidateCriteria(c *Criteria) error {
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

func (decoder CriteriaDecoder) validateRoute(c *RouteCriteria) error {
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
	if c.PathIndex == c.HandlerIndex {
		decoder.Logger.Printf("route validation error: func %s either does not specified both path and handler, or assigned the same index for both\n", c.FuncName)
		return ErrInvalidRoute
	}
	return nil
}

func (decoder CriteriaDecoder) validateCallCriteria(c *CallCriteria, callCriteriaName string) error {
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
func NewCriteriaDecoder(logger *log.Logger) CriteriaDecoder {
	return CriteriaDecoder{
		Logger: logger,
	}
}
