package golden

import (
	"net/http"
)

const (
	identifiersGet = http.MethodGet
)

var (
	identifiersPost = http.MethodPost
)

func readIdentifiers(r *http.Request) {
	m := r.Method
	put := http.MethodPut
	delete := "Delete"
	switch m {
	case identifiersGet:
		// Serve the resource.
	case identifiersPost:
		// Create a new record.
	case put:
		// Update an existing record.
	case delete:
		// Remove the record.
	case http.MethodPatch:
		// patch
	default:
	}
}
