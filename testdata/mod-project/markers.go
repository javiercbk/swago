package golden

import (
	"modproj/user"
	"net/http"
)

// FirstHandler is the first handler swago:handler
func FirstHandler(r *http.Request, w http.ResponseWriter) {

}

// SecondHandler is the second handler
func SecondHandler(r *http.Request, w http.ResponseWriter) {

}

// ThirdHandler is the second handler
func ThirdHandler(r *http.Request, w http.ResponseWriter) {

}

func RouteInit() {
	mux := http.NewServeMux()
	mux.HandleFunc("/some/path/to/handle", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Serve the resource.
		case http.MethodPost:
			// Create a new record.
		case http.MethodPut:
			// Update an existing record.
		case http.MethodDelete:
			// Remove the record.
		default:
		}
	})
}

func functionFinder() {
	u := user.User{}
	u.UserFunc()
	aSimpleFunc()
	mux := http.NewServeMux()
	mux.HandleFunc("/some/path/to/handle", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Serve the resource.
		case http.MethodPost:
			// Create a new record.
		case http.MethodPut:
			// Update an existing record.
		case http.MethodDelete:
			// Remove the record.
		default:
		}
	})
}
