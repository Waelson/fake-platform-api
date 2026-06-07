package response

import (
	"encoding/json"
	"net/http"
)

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// JSON writes status and data as JSON. Caller controls the HTTP status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Error writes a structured error response.
func Error(w http.ResponseWriter, status int, code, message string, retryable bool) {
	JSON(w, status, ErrorBody{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			Retryable: retryable,
		},
	})
}
