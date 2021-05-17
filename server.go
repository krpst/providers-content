package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// App represents the server's internal state.
// It holds configuration about providers and content.
type App struct {
	ContentProvider ContentProvider
}

func (a App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("%s %s", req.Method, req.URL.String())

	switch req.Method {
	case http.MethodGet:
		switch req.URL.Path {
		case "/":
			a.getContent(w, req)
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func (a App) parseUserIP(req *http.Request) string {
	var ipAddress = req.Header.Get("X-Forwarded-For")

	return ipAddress
}

func encodeJSONResponse(w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func encodeErrorResp(err error, w http.ResponseWriter) {
	log.Printf("error: %s", err)

	errResp := struct {
		Err Error `json:"error"`
	}{Error{
		Code:    ErrorCode(err),
		Message: ErrorMessage(err),
	}}

	var customErr Error
	// we don't want to share details about internal errors.
	if !errors.As(err, &customErr) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)

		return
	}

	switch ErrorCode(err) {
	case ErrValidation:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(errResp)
}
