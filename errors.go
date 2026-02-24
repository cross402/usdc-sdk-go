package pay

import "fmt"

// APIError represents an error response from the v2 payment API (4xx/5xx).
// StatusCode is the HTTP status; Message is the human-readable body message.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d: %s", e.StatusCode, e.Message)
}
