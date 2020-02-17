package golden

import (
	"net/http"
	//
	. "strings"
)

const (
	identifiersGet = http.MethodGet
)

var (
	identifiersPost = http.MethodPost
)

type identifierStruct struct{}

func (i identifierStruct) idFunc()         {}
func (i *identifierStruct) idFuncWithPtr() {}

func readIdentifiers(r *http.Request) []string {
	m := r.Method
	put := http.MethodPut
	delete := "Delete"
	get := identifiersGet
	return []string{m, put, delete, get}
}

func callExprIdentifier(someStr string) {
	ids := identifierStruct{}
	ids.idFunc()
	coolPath := ToLower("/cool")
	http.HandleFunc(coolPath, func(w http.ResponseWriter, r *http.Request) {

	})
}

func findValue() []string {
	put := http.MethodPut
	del := "Delete"
	get := identifiersGet
	callExprIdentifier(put)
	callExprIdentifier(del)
	callExprIdentifier(get)
	callExprIdentifier("cool")
	callExprIdentifier(http.MethodPatch)
	return []string{put, del, get}
}
