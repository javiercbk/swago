package routes

import (
	"net/http"
)

// Middleware defines a middleware
type Middleware func(http.Handler, Route) http.Handler

// Route defines a route
type Route struct {
	Name            string           // A short name to describe the handler.  This name will be logged and filterable in graphs
	Method          string           // GET, PUT, POST, etc
	Pattern         string           // The route you want (eg.  /widget/{id} )
	HandlerFunc     http.HandlerFunc // The function that should be called
	ValidateJwt     bool             // Should we validate the JWT before processing request
	AddlMiddlewares []Middleware     // Any additional middleware functions
	HandlerTimeout  int              // The time in seconds for a timeout on the handler
}
