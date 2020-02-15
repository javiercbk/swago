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

func readIdentifiers(r *http.Request) []string {
	m := r.Method
	put := http.MethodPut
	delete := "Delete"
	get := identifiersGet
	return []string{m, put, delete, get}
}
