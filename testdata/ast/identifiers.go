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

type identifierStruct struct{}

func (i identifierStruct) idFunc() {}

func readIdentifiers(r *http.Request) []string {
	m := r.Method
	put := http.MethodPut
	delete := "Delete"
	get := identifiersGet
	return []string{m, put, delete, get}
}

func callExprIdentifier() {
	ids := identifierStruct{}
	ids.idFunc()
	http.HandleFunc("/cool", func(w http.ResponseWriter, r *http.Request) {

	})
}
