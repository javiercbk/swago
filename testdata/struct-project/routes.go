package main

import (
	"net/http"

	"structproj/handler"
	r "structproj/routes"
)

var routes = r.Routes{
	r.Route{
		Name:           "V1Handler1",
		Method:         http.MethodPost,
		Pattern:        "/handler/1",
		ApiVer:         "v1",
		HandlerTimeout: 15,
		HandlerFunc:    handler.V1Handler1,
		ValidateJwt:    false,
	},
	r.Route{
		Name:           "V1Handler2",
		Method:         http.MethodGet,
		Pattern:        "/handler2/nice",
		ApiVer:         "v1",
		HandlerTimeout: 15,
		HandlerFunc:    handler.V1Handler2,
		ValidateJwt:    true,
	},
	r.Route{
		Name:           "V1Handler3",
		Method:         http.MethodPut,
		Pattern:        "/handler/{name}/3",
		ApiVer:         "v1",
		HandlerTimeout: 15,
		HandlerFunc:    handler.V1Handler3,
		ValidateJwt:    true,
	},
}

var serviceroutes = dcgroute.GenericServiceRoutes
