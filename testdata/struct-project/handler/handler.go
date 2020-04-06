package handler

import (
	"net/http"

	"structproj/api"
)

// V1Handler1 is a sample handler
func V1Handler1(w http.ResponseWriter, r *http.Request) {
	request := V1Request1{}

	if err := api.DecodeJsonRequest(r, &request); err != nil {
		sendError(w, r, GetError("bad request", err, http.StatusBadRequest))
		return
	}
	responseStruct := V1Response1{}
	api.SendJSONResponse(w, r, responseStruct, http.StatusOK)
}

// V1Handler2 is a sample handler
func V1Handler2(w http.ResponseWriter, r *http.Request) {
	request := V1Request2{}

	if err := api.DecodeJsonRequest(r, &request); err != nil {
		sendError(w, r, GetError("bad request", err, http.StatusBadRequest))
		return
	}
	responseStruct := V1Response2{}
	api.SendJSONResponse(w, r, responseStruct, http.StatusOK)
}

// V1Handler3 is a sample handler
func V1Handler3(w http.ResponseWriter, r *http.Request) {
	request := V1Request3{}

	if err := api.DecodeJsonRequest(r, &request); err != nil {
		sendError(w, r, GetError("bad request", err, http.StatusBadRequest))
		return
	}
	responseStruct := V1Response3{}
	api.SendJSONResponse(w, r, responseStruct, http.StatusOK)
}
