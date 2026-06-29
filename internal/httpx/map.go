package httpx

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"
)

func NormalizeError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrForbidden) ||
		errors.Is(err, ErrUnauthorized) ||
		errors.Is(err, ErrConflict) ||
		errors.Is(err, ErrBadRequest) ||
		errors.Is(err, ErrInvalidInput) {
		return err
	}

	if errors.Is(err, sql.ErrNoRows) {
		return NotFound("resource not found")
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			return Conflict(uniqueViolationMessage(pqErr))
		case "23503":
			return InvalidInput("referenced record does not exist")
		case "23502":
			return InvalidInput("required field is missing")
		case "22P02":
			return InvalidInput("invalid id format")
		case "23514":
			return InvalidInput("value does not meet validation rules")
		}
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "duplicate key"), strings.Contains(msg, "unique constraint"):
		return Conflict("record already exists")
	case strings.Contains(msg, "foreign key"):
		return InvalidInput("referenced record does not exist")
	case strings.Contains(msg, "invalid input syntax"):
		return InvalidInput("invalid id format")
	}

	return err
}

func uniqueViolationMessage(pqErr *pq.Error) string {
	constraint := pqErr.Constraint
	switch {
	case strings.Contains(constraint, "users_email"):
		return "email already in use"
	case strings.Contains(constraint, "academic_years_name"):
		return "academic year name already exists"
	case strings.Contains(constraint, "academic_classes_name"), strings.Contains(constraint, "academic_classes_grade"):
		return "class name or grade already exists"
	case strings.Contains(constraint, "subjects_name"), strings.Contains(constraint, "subjects_code"):
		return "subject name or code already exists"
	case strings.Contains(constraint, "subject_offerings"):
		return "offering for this class and subject already exists"
	case strings.Contains(constraint, "batches"):
		return "batch name already exists for this offering"
	case strings.Contains(constraint, "enrollments"):
		return "student already has an active enrollment for this offering"
	default:
		return "record already exists"
	}
}
