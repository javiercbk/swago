package golden

import (
	"net/http"
)

const (
	get = http.MethodGet
)

var (
	post = http.MethodPost
)

func idealSwitch(r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Serve the resource.
	case http.MethodPost:
		// Create a new record.
	case http.MethodPut:
		// Update an existing record.
	case http.MethodDelete:
		// Remove the record.
	case http.MethodPatch:
		// patch
	default:
	}
}

func notCompleteSwitch(r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Serve the resource.
	case http.MethodPost:
		// Create a new record.
	default:
	}
}

func switchWithVars(r *http.Request) {
	m := r.Method
	put := http.MethodPut
	switch m {
	case get:
		// Serve the resource.
	case post:
		// Create a new record.
	case put:
		// Update an existing record.
	case http.MethodDelete:
		// Remove the record.
	default:
	}
}
