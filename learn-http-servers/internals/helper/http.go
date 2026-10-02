package helper

import (
	"fmt"
	"log/slog"
	"net/http"
)

func WriteErrorResponse(w http.ResponseWriter, error string, code int) {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)

	if _, err := fmt.Fprintf(w, `{"error":%q}`, error); err != nil {
		slog.Error("Failed to write response body", "err", err)
	}
}

func WriteInternalServerErrorResponse(w http.ResponseWriter, internalMessage string, err error) {
	slog.Error(internalMessage, "err", err)
	WriteErrorResponse(w, "Something went wrong", http.StatusInternalServerError)
}

func WriteResponseBody(w http.ResponseWriter, body []byte) {
	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write response body", "err", err)
	}
}
