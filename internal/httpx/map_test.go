package httpx

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/lib/pq"
)

func TestNormalizeError_KeepsTypedErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		base error
	}{
		{"not found", NotFound("user not found"), ErrNotFound},
		{"forbidden", Forbidden("access denied for batch"), ErrForbidden},
		{"unauthorized", Unauthorized("invalid token"), ErrUnauthorized},
		{"conflict", Conflict("email already in use"), ErrConflict},
		{"invalid input", InvalidInput("name is required"), ErrInvalidInput},
		{"bad request", BadRequest("body required"), ErrBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeError(tc.err)
			if !errors.Is(got, tc.base) {
				t.Fatalf("expected %v, got %v", tc.base, got)
			}
			if got.Error() != tc.err.Error() {
				t.Fatalf("expected message %q, got %q", tc.err.Error(), got.Error())
			}
		})
	}
}

func TestNormalizeError_SQLNoRows(t *testing.T) {
	got := NormalizeError(sql.ErrNoRows)
	if !errors.Is(got, ErrNotFound) {
		t.Fatalf("expected NOT_FOUND, got %v", got)
	}
	if errorMessage(got, ErrNotFound) != "resource not found" {
		t.Fatalf("unexpected message: %s", errorMessage(got, ErrNotFound))
	}
}

func TestNormalizeError_PostgresCodes(t *testing.T) {
	cases := []struct {
		code    pq.ErrorCode
		wantMsg string
		base    error
	}{
		{"23505", "record already exists", ErrConflict},
		{"23503", "referenced record does not exist", ErrInvalidInput},
		{"23502", "required field is missing", ErrInvalidInput},
		{"22P02", "invalid id format", ErrInvalidInput},
		{"23514", "value does not meet validation rules", ErrInvalidInput},
	}

	for _, tc := range cases {
		t.Run(string(tc.code), func(t *testing.T) {
			err := &pq.Error{Code: tc.code, Message: "db error"}
			got := NormalizeError(err)
			if !errors.Is(got, tc.base) {
				t.Fatalf("expected %v, got %v", tc.base, got)
			}
			if errorMessage(got, tc.base) != tc.wantMsg {
				t.Fatalf("expected %q, got %q", tc.wantMsg, errorMessage(got, tc.base))
			}
		})
	}
}

func TestNormalizeError_UniqueViolationConstraint(t *testing.T) {
	err := &pq.Error{Code: "23505", Constraint: "users_email_key"}
	got := NormalizeError(err)
	if errorMessage(got, ErrConflict) != "email already in use" {
		t.Fatalf("unexpected message: %s", errorMessage(got, ErrConflict))
	}
}

func TestNormalizeError_UnknownErrorPassthrough(t *testing.T) {
	raw := errors.New("something broke")
	got := NormalizeError(raw)
	if got != raw {
		t.Fatalf("expected passthrough, got %v", got)
	}
}
