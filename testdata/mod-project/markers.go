package golden

import (
	"net/http"
)

// FirstHandler is the first handler swago:handler
func FirstHandler(r *http.Request, w http.ResponseWriter) {

}

// SecondHandler is the second handler
// swago:handler
func SecondHandler(r *http.Request, w http.ResponseWriter) {

}

// ThirdHandler is the second handler
/*  swago:handler */
func ThirdHandler(r *http.Request, w http.ResponseWriter) {

}
