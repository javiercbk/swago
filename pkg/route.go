package pkg

// Route is web route found in the folder
type Route struct {
	Pkg           string
	File          string
	Path          string
	HTTPMethod    string
	HandlerType   string
	ChildRoutes   []*Route
	Middlewares   []string
	RequestModel  Struct
	ResponseModel Struct
	Struct        map[string]string
}
