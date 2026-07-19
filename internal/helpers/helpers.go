// Package helpers provides shared utility functions used across the backend.
// It includes HTTP request parsing, JSON decoding, and input validation helpers
// designed to reduce boilerplate in handler code.
package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// malformedRequest represents an HTTP request that failed validation or decoding.
// It carries both a human-readable message and the appropriate HTTP status code
// so handlers can respond without additional error translation logic.
type malformedRequest struct {
	Status  int
	Message string
}

// Error implements the error interface for malformedRequest.
func (mr *malformedRequest) Error() string {
	return mr.Message
}

// DecodeBodyJSON decodes a JSON request body into the provided destination struct.
//
// It enforces the following constraints:
//   - Content-Type must be application/json if provided
//   - Body must not exceed 1MB
//   - Body must contain valid, well-formed JSON
//   - Body must not contain unknown fields
//   - Body must contain exactly one JSON object
//
// Returns a *malformedRequest error with the appropriate HTTP status code
// and a descriptive message if any constraint is violated, or nil on success.
func DecodeBodyJSON(writer http.ResponseWriter, request *http.Request, destination any) *malformedRequest {
	contentType := request.Header.Get("Content-Type")
	if contentType != "" {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
		if mediaType != "application/json" {
			msg := "Content-Type header is not application/json"
			return &malformedRequest{Status: http.StatusUnsupportedMediaType, Message: msg}
		}
	}

	request.Body = http.MaxBytesReader(writer, request.Body, 1048576)

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&destination)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &malformedRequest{Status: http.StatusBadRequest, Message: msg}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := "Request body contains badly-formed JSON"
			return &malformedRequest{Status: http.StatusBadRequest, Message: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &malformedRequest{Status: http.StatusBadRequest, Message: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &malformedRequest{Status: http.StatusBadRequest, Message: msg}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &malformedRequest{Status: http.StatusBadRequest, Message: msg}

		case errors.As(err, &maxBytesError):
			msg := fmt.Sprintf("Request body must not be larger than %d bytes", maxBytesError.Limit)
			return &malformedRequest{Status: http.StatusRequestEntityTooLarge, Message: msg}

		default:
			return &malformedRequest{Message: err.Error(), Status: http.StatusBadRequest}
		}
	}

	err = decoder.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		msg := "Request body must only contain a single JSON object"
		return &malformedRequest{Status: http.StatusBadRequest, Message: msg}
	}
	return nil
}
