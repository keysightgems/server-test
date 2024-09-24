package http

import (
	"fmt"
	"net/http"
)

type JSONWriter interface {
	ToJson() (string, error)
}

// WriteJSONResponse sets an HTTP response with the provided status-code and JSON body.
func WriteJSONResponse(w http.ResponseWriter, statuscode int, data JSONWriter) (int, error) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statuscode)

	dataStr, err := data.ToJson()

	if err != nil {
		if _, err := WriteDefaultResponse(w, http.StatusInternalServerError); err != nil {
			log.Print(err.Error())
		}
	}
	return w.Write([]byte(dataStr))
}

// WriteDefaultResponse sets the body of a response with the text associated with the HTTP status.
func WriteDefaultResponse(w http.ResponseWriter, status int) (int, error) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	fmt.Fprintf(w, "%d %s", status, http.StatusText(status))
	return status, nil
}

// WriteJSONResponse sets an HTTP response with the provided status-code and JSON data.
func WriteCustomJSONResponse(w http.ResponseWriter, statuscode int, data []byte) (int, error) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statuscode)
	return w.Write([]byte(data))
}
