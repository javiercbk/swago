package routes

import (
	"net/http"
)

// Middleware defines a middleware
type Middleware func(http.Handler, Route) http.Handler

// Route defines a route
type Route struct {
	Name            string `tag:"Name"`
	Method          string `another:"tag"`
	Pattern         string
	HandlerFunc     http.HandlerFunc
	ValidateJwt     bool
	AddlMiddlewares []Middleware
	HandlerTimeout  int
}
