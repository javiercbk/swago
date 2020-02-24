package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/foxbroadcasting/go-helpers/handlererror"
	"github.com/foxbroadcasting/go-helpers/requesthelper"
)

// SendJSONResponse sends a json response
func SendJSONResponse(w http.ResponseWriter, r *http.Request, resp interface{}, code int) {
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		log := requesthelper.GetFromContext(r.Context()).GetLoggingEntry()
		log.Error("sendJSONResponse() error while encoding. err = ", err)
	}
}

// DecodeJsonRequest decodes a json request
func DecodeJsonRequest(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return GetError("RequestError", "bad json request", http.StatusBadRequest)
	}
	return nil
}
