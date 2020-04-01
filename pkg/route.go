package pkg

// Route is web route found in the folder
type Route struct {
	Pkg              string
	File             string
	Path             string
	HTTPMethod       string
	HandlerType      string
	ChildRoutes      []*Route
	Middlewares      []string
	RequestModel     Struct
	ServiceResponses []ServiceResponse
	Struct           map[string]string
}

// ServiceResponse is the response from a service
type ServiceResponse struct {
	Code  string
	Model Struct
}
