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

const userSelect = `
	SELECT u.id, u.email, u.name, u.role, u.status, u.phone, u.class_id, COALESCE(ac.name, ''),
	       u.created_at, u.updated_at
	FROM users u
	LEFT JOIN academic_classes ac ON ac.id = u.class_id
`

func (r *UserRepository) scanUser(row interface {
	Scan(dest ...any) error
}) (*models.User, error) {
	var u models.User
	var phone sql.NullString
	var classID sql.NullString
	var className string
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Status, &phone, &classID, &className,
		&u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("user not found")
	}
	if err != nil {
		return nil, err
	}
	if phone.Valid {
		u.Phone = &phone.String
	}
	if classID.Valid {
		u.ClassID = &classID.String
	}
	u.ClassName = className
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash, name, role string, phone *string, classID *string) (*models.User, error) {
	var phoneVal sql.NullString
	if phone != nil {
		phoneVal = sql.NullString{String: *phone, Valid: true}
	}
	var classVal sql.NullString
	if classID != nil {
		classVal = sql.NullString{String: *classID, Valid: true}
	}

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, name, role, phone, class_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, email, passwordHash, name, role, phoneVal, classVal)

	var id string
	if err := row.Scan(&id); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, httpx.Conflict("email already in use")
		}
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, userSelect+` WHERE u.id = $1`, id))
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, string, error) {
	var u models.User
	var phone sql.NullString
	var classID sql.NullString
	var className string
	var hash string
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.password_hash, u.name, u.role, u.status, u.phone, u.class_id,
		       COALESCE(ac.name, ''), u.created_at, u.updated_at
		FROM users u
		LEFT JOIN academic_classes ac ON ac.id = u.class_id
		WHERE u.email = $1
	`, email).Scan(&u.ID, &u.Email, &hash, &u.Name, &u.Role, &u.Status, &phone, &classID, &className,
		&u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, "", httpx.NotFound("user not found")
	}
	if err != nil {
		return nil, "", err
	}
	if phone.Valid {
		u.Phone = &phone.String
	}
	if classID.Valid {
		u.ClassID = &classID.String
	}
	u.ClassName = className
	return &u, hash, nil
}

func (r *UserRepository) Update(ctx context.Context, id string, name string, phone *string, status *string, classID *string) (*models.User, error) {
	var phoneVal sql.NullString
	if phone != nil {
		phoneVal = sql.NullString{String: *phone, Valid: true}
	}
	statusVal := sql.NullString{}
	if status != nil {
		statusVal = sql.NullString{String: *status, Valid: true}
	}
	var classVal sql.NullString
	if classID != nil {
		classVal = sql.NullString{String: *classID, Valid: true}
	}

	row := r.db.QueryRowContext(ctx, `
		UPDATE users SET
			name = COALESCE(NULLIF($2, ''), name),
			phone = COALESCE($3, phone),
			status = COALESCE($4, status),
			class_id = COALESCE($5, class_id),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, id, name, phoneVal, statusVal, classVal)

	var updatedID string
	if err := row.Scan(&updatedID); err == sql.ErrNoRows {
		return nil, httpx.NotFound("user not found")
	} else if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, updatedID)
}

func (r *UserRepository) List(ctx context.Context, role, status, classID, search string, offset, limit int) ([]models.User, int, error) {
	where := "WHERE u.role = $1"
	args := []any{role}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND u.status = $%d", len(args))
	}
	if classID != "" {
		args = append(args, classID)
		where += fmt.Sprintf(" AND u.class_id = $%d", len(args))
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (u.name ILIKE $%d OR u.email ILIKE $%d)", len(args), len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users u `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(userSelect+` %s ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := r.scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
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
		return httpx.NotFound("user not found")
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return httpx.NotFound("user not found")
	}
	return nil
}

func (r *UserRepository) GetStudentClassID(ctx context.Context, studentID string) (string, error) {
	var role string
	var classID sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT role, class_id FROM users WHERE id = $1`, studentID).Scan(&role, &classID)
	if err == sql.ErrNoRows {
		return "", httpx.NotFound("user not found")
	}
	if err != nil {
		return "", err
	}
	if role != models.RoleStudent {
		return "", httpx.InvalidInput("user is not a student")
	}
	if !classID.Valid {
		return "", httpx.InvalidInput("student has no class assigned")
	}
	return classID.String, nil
}
