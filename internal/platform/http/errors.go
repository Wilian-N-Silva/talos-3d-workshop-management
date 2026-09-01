package httpplatform

import (
	"encoding/json"
	"net/http"
)

// ErrorEnvelope is the stable API error response shape.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody describes a machine-readable API error.
type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

// WriteError writes a standardized JSON error response.
func WriteError(
	response http.ResponseWriter,
	status int,
	code string,
	message string,
	details map[string]any,
) {
	if details == nil {
		details = map[string]any{}
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(ErrorEnvelope{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
