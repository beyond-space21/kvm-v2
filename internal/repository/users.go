package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash, name, role string, phone *string) (*models.User, error) {
	var u models.User
	var phoneVal sql.NullString
	if phone != nil {
		phoneVal = sql.NullString{String: *phone, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, name, role, phone)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, name, role, status, phone, created_at, updated_at
	`, email, passwordHash, name, role, phoneVal).Scan(
		&u.ID, &u.Email, &u.Name, &u.Role, &u.Status, &phoneVal, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, httpx.ErrConflict
		}
		return nil, err
	}
	if phoneVal.Valid {
		u.Phone = &phoneVal.String
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	var phone sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, name, role, status, phone, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Status, &phone, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if phone.Valid {
		u.Phone = &phone.String
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, string, error) {
	var u models.User
	var phone sql.NullString
	var hash string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, name, role, status, phone, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &hash, &u.Name, &u.Role, &u.Status, &phone, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, "", httpx.ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	if phone.Valid {
		u.Phone = &phone.String
	}
	return &u, hash, nil
}

func (r *UserRepository) Update(ctx context.Context, id string, name string, phone *string, status *string) (*models.User, error) {
	var u models.User
	var phoneVal sql.NullString
	if phone != nil {
		phoneVal = sql.NullString{String: *phone, Valid: true}
	}
	statusVal := sql.NullString{}
	if status != nil {
		statusVal = sql.NullString{String: *status, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, `
		UPDATE users SET
			name = COALESCE(NULLIF($2, ''), name),
			phone = COALESCE($3, phone),
			status = COALESCE($4, status),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, name, role, status, phone, created_at, updated_at
	`, id, name, phoneVal, statusVal).Scan(
		&u.ID, &u.Email, &u.Name, &u.Role, &u.Status, &phoneVal, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if phoneVal.Valid {
		u.Phone = &phoneVal.String
	}
	return &u, nil
}

func (r *UserRepository) List(ctx context.Context, role string, offset, limit int) ([]models.User, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if role != "" {
		args = append(args, role)
		where += fmt.Sprintf(" AND role = $%d", len(args))
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM users " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT id, email, name, role, status, phone, created_at, updated_at
		FROM users %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var phone sql.NullString
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Status, &phone, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if phone.Valid {
			u.Phone = &phone.String
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *UserRepository) CountByRole(ctx context.Context, role string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = $1`, role).Scan(&count)
	return count, err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id, hash string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, id, hash)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return httpx.ErrNotFound
	}
	return nil
}
