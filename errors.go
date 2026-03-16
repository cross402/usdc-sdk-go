package pay

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyBaseURL     = errors.New("baseURL is required")
	ErrEmptyIntentID    = errors.New("intent_id is required")
	ErrEmptySettleProof = errors.New("settle_proof is required")
	ErrMissingAuth      = errors.New("auth option required")
	ErrNilParams        = errors.New("params must not be nil")
)

// UnexpectedError wraps unexpected errors (marshal, request creation).
type UnexpectedError struct {
	Err error
}

func (e *UnexpectedError) Error() string {
	return fmt.Sprintf("unexpected error: %v", e.Err)
}

func (e *UnexpectedError) Unwrap() error { return e.Err }

// RequestError represents an HTTP 4xx/5xx error response from the API.
type RequestError struct {
	StatusCode int
	Body       string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("request failed with status %d: %s", e.StatusCode, e.Body)
}

// ValidationError is returned when the SDK rejects a request before
// it reaches the API (e.g. empty intent ID). Wraps a sentinel error
// so callers can use both errors.As and errors.Is.
type ValidationError struct {
	Message string
	Err     error
}

func (e *ValidationError) Error() string { return "validation: " + e.Message }

func (e *ValidationError) Unwrap() error { return e.Err }
