package handler

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse легаси залупа
// @Description ErrorResponse provides a consistent structure for API errors
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}

// writeJSON writes any struct as JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError sends JSON error response in unified format
func writeProductError(w http.ResponseWriter, status int, message string, details string) {
	resp := ErrorProductResponse{
		Code:    status,
		Message: message,
		Details: details,
	}
	writeJSON(w, status, resp)
}

// writeError sends JSON error response in unified format
func writeUserError(w http.ResponseWriter, status int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorUserResponse{
		Code:    status,
		Message: message,
		Details: details,
	})
}
