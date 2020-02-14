package golden

import (
	cool "net/http"
)

const (
	getCool = cool.MethodGet
)

var (
	postCool = cool.MethodPost
)

func coolIdealSwitch(r *cool.Request) {
	switch r.Method {
	case cool.MethodGet:
		// Serve the resource.
	case cool.MethodPost:
		// Create a new record.
	case cool.MethodPut:
		// Update an existing record.
	case cool.MethodDelete:
		// Remove the record.
	case cool.MethodPatch:
		// patch
	default:
	}
}

func coolNotCompleteSwitch(r *cool.Request) {
	switch r.Method {
	case cool.MethodGet:
		// Serve the resource.
	case cool.MethodPost:
		// Create a new record.
	default:
	}
}

func coolSwitchWithVars(r *cool.Request) {
	m := r.Method
	putCool := cool.MethodPut
	switch m {
	case getCool:
		// Serve the resource.
	case postCool:
		// Create a new record.
	case putCool:
		// Update an existing record.
	case cool.MethodDelete:
		// Remove the record.
	default:
	}
}
