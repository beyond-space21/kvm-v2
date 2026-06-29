package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

var (
	ErrNotFound     = errors.New("resource not found")
	ErrForbidden    = errors.New("access denied")
	ErrUnauthorized = errors.New("authentication required")
	ErrConflict     = errors.New("resource conflict")
	ErrBadRequest   = errors.New("bad request")
	ErrInvalidInput = errors.New("invalid request")
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, err error) {
	err = NormalizeError(err)
	switch {
	case errors.Is(err, ErrNotFound):
		WriteJSON(w, http.StatusNotFound, APIError{Error: errorMessage(err, ErrNotFound), Code: "NOT_FOUND"})
	case errors.Is(err, ErrForbidden):
		WriteJSON(w, http.StatusForbidden, APIError{Error: errorMessage(err, ErrForbidden), Code: "FORBIDDEN"})
	case errors.Is(err, ErrUnauthorized):
		WriteJSON(w, http.StatusUnauthorized, APIError{Error: errorMessage(err, ErrUnauthorized), Code: "UNAUTHORIZED"})
	case errors.Is(err, ErrConflict):
		WriteJSON(w, http.StatusConflict, APIError{Error: errorMessage(err, ErrConflict), Code: "CONFLICT"})
	case errors.Is(err, ErrInvalidInput):
		WriteJSON(w, http.StatusBadRequest, APIError{Error: errorMessage(err, ErrInvalidInput), Code: "BAD_REQUEST"})
	case errors.Is(err, ErrBadRequest):
		WriteJSON(w, http.StatusBadRequest, APIError{Error: errorMessage(err, ErrBadRequest), Code: "BAD_REQUEST"})
	default:
		WriteJSON(w, http.StatusInternalServerError, APIError{Error: "internal server error", Code: "INTERNAL"})
	}
}

func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return BadRequest("request body is required")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return InvalidInput("request body must be valid JSON")
	}
	if dec.More() {
		return InvalidInput("request body must contain a single JSON object")
	}
	return nil
}

func InvalidInput(message string) error {
	return wrap(ErrInvalidInput, message)
}

func BadRequest(message string) error {
	return wrap(ErrBadRequest, message)
}

func NotFound(message string) error {
	return wrap(ErrNotFound, message)
}

func Forbidden(message string) error {
	return wrap(ErrForbidden, message)
}

func Unauthorized(message string) error {
	return wrap(ErrUnauthorized, message)
}

func Conflict(message string) error {
	return wrap(ErrConflict, message)
}

func wrap(base error, message string) error {
	if strings.TrimSpace(message) == "" {
		return base
	}
	return fmt.Errorf("%w: %s", base, message)
}

func errorMessage(err error, base error) string {
	if err == nil {
		return base.Error()
	}
	msg := err.Error()
	prefix := base.Error() + ": "
	if strings.HasPrefix(msg, prefix) {
		return strings.TrimPrefix(msg, prefix)
	}
	return msg
}
