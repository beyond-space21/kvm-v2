package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

var (
	ErrNotFound      = errors.New("not found")
	ErrForbidden     = errors.New("forbidden")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrConflict      = errors.New("conflict")
	ErrBadRequest    = errors.New("bad request")
	ErrInvalidInput  = errors.New("invalid input")
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		WriteJSON(w, http.StatusNotFound, APIError{Error: err.Error(), Code: "NOT_FOUND"})
	case errors.Is(err, ErrForbidden):
		WriteJSON(w, http.StatusForbidden, APIError{Error: err.Error(), Code: "FORBIDDEN"})
	case errors.Is(err, ErrUnauthorized):
		WriteJSON(w, http.StatusUnauthorized, APIError{Error: err.Error(), Code: "UNAUTHORIZED"})
	case errors.Is(err, ErrConflict):
		WriteJSON(w, http.StatusConflict, APIError{Error: err.Error(), Code: "CONFLICT"})
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrBadRequest):
		WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error(), Code: "BAD_REQUEST"})
	default:
		WriteJSON(w, http.StatusInternalServerError, APIError{Error: "internal server error", Code: "INTERNAL"})
	}
}

func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return ErrBadRequest
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return ErrInvalidInput
	}
	return nil
}
