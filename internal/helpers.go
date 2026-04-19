// This file stores helper functions and variables to be used throughout the backend application

package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	errNoTokenInRequest = errors.New("no token present in request")
)

type malformedRequest struct {
	Status  int
	Message string
}

func (mr *malformedRequest) Error() string {
	return mr.Message
}

func DecodeBodyJson(writer http.ResponseWriter, request *http.Request, destination any) *malformedRequest {
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

func ExtractBearerToken(r *http.Request) (string, error) {
	tokenHeader := r.Header.Get("Authorization")
	if len(tokenHeader) < 7 || !strings.EqualFold(tokenHeader[:6], "bearer") {
		return "", errNoTokenInRequest
	}
	token := strings.TrimSpace(tokenHeader[7:])
	if token == "" {
		return "", errNoTokenInRequest
	}
	return token, nil
}
