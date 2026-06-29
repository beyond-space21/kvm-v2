package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
)

type EnrollmentRepository struct {
	db *sql.DB
}

func NewEnrollmentRepository(db *sql.DB) *EnrollmentRepository {
	return &EnrollmentRepository{db: db}
}

func (r *EnrollmentRepository) Create(ctx context.Context, studentID, yearID, offeringID, batchID string) (*models.Enrollment, error) {
	var e models.Enrollment
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO enrollments (student_id, academic_year_id, offering_id, batch_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, student_id, academic_year_id, offering_id, batch_id, status, enrolled_at, ended_at, created_at, updated_at
	`, studentID, yearID, offeringID, batchID).Scan(
		&e.ID, &e.StudentID, &e.AcademicYearID, &e.OfferingID, &e.BatchID, &e.Status,
		&e.EnrolledAt, &e.EndedAt, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, httpx.ErrConflict
		}
		return nil, err
	}
	return &e, nil
}

func (r *EnrollmentRepository) Get(ctx context.Context, id string) (*models.Enrollment, error) {
	return r.scanOne(r.db.QueryRowContext(ctx, enrollmentSelect+` WHERE e.id = $1`, id))
}

func (r *EnrollmentRepository) List(ctx context.Context, studentID, yearID, batchID, status string, offset, limit int) ([]models.Enrollment, int, error) {
	where, args := r.filters(studentID, yearID, batchID, status)

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM enrollments e `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := enrollmentSelect + where + fmt.Sprintf(` ORDER BY e.created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var enrollments []models.Enrollment
	for rows.Next() {
		e, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		enrollments = append(enrollments, *e)
	}
	return enrollments, total, rows.Err()
}

func (r *EnrollmentRepository) Transfer(ctx context.Context, id, newBatchID string) (*models.Enrollment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var old models.Enrollment
	err = tx.QueryRowContext(ctx, `
		SELECT id, student_id, academic_year_id, offering_id, batch_id, status
		FROM enrollments WHERE id = $1 AND status = 'active' FOR UPDATE
	`, id).Scan(&old.ID, &old.StudentID, &old.AcademicYearID, &old.OfferingID, &old.BatchID, &old.Status)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `
		UPDATE enrollments SET status = 'transferred', ended_at = $2, updated_at = NOW() WHERE id = $1
	`, id, now); err != nil {
		return nil, err
	}

	var e models.Enrollment
	err = tx.QueryRowContext(ctx, `
		INSERT INTO enrollments (student_id, academic_year_id, offering_id, batch_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, student_id, academic_year_id, offering_id, batch_id, status, enrolled_at, ended_at, created_at, updated_at
	`, old.StudentID, old.AcademicYearID, old.OfferingID, newBatchID).Scan(
		&e.ID, &e.StudentID, &e.AcademicYearID, &e.OfferingID, &e.BatchID, &e.Status,
		&e.EnrolledAt, &e.EndedAt, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &e, tx.Commit()
}

func (r *EnrollmentRepository) Remove(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE enrollments SET status = 'removed', ended_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'active'
	`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

func (r *EnrollmentRepository) History(ctx context.Context, studentID string) ([]models.Enrollment, error) {
	rows, err := r.db.QueryContext(ctx, enrollmentSelect+` WHERE e.student_id = $1 ORDER BY e.created_at DESC`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enrollments []models.Enrollment
	for rows.Next() {
		e, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		enrollments = append(enrollments, *e)
	}
	return enrollments, rows.Err()
}

const enrollmentSelect = `
	SELECT e.id, e.student_id, u.name, e.academic_year_id, e.offering_id, e.batch_id, b.name, s.name, ac.name,
	       e.status, e.enrolled_at, e.ended_at, e.created_at, e.updated_at
	FROM enrollments e
	JOIN users u ON u.id = e.student_id
	JOIN batches b ON b.id = e.batch_id
	JOIN subject_offerings so ON so.id = e.offering_id
	JOIN subjects s ON s.id = so.subject_id
	JOIN academic_classes ac ON ac.id = so.class_id
`

func (r *EnrollmentRepository) filters(studentID, yearID, batchID, status string) (string, []any) {
	where := "WHERE 1=1"
	args := []any{}
	if studentID != "" {
		args = append(args, studentID)
		where += fmt.Sprintf(" AND e.student_id = $%d", len(args))
	}
	if yearID != "" {
		args = append(args, yearID)
		where += fmt.Sprintf(" AND e.academic_year_id = $%d", len(args))
	}
	if batchID != "" {
		args = append(args, batchID)
		where += fmt.Sprintf(" AND e.batch_id = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND e.status = $%d", len(args))
	}
	return where, args
}

func (r *EnrollmentRepository) scanOne(row *sql.Row) (*models.Enrollment, error) {
	var e models.Enrollment
	var ended sql.NullTime
	err := row.Scan(&e.ID, &e.StudentID, &e.StudentName, &e.AcademicYearID, &e.OfferingID, &e.BatchID,
		&e.BatchName, &e.SubjectName, &e.ClassName, &e.Status, &e.EnrolledAt, &ended, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if ended.Valid {
		e.EndedAt = &ended.Time
	}
	return &e, nil
}

func (r *EnrollmentRepository) scanRow(rows *sql.Rows) (*models.Enrollment, error) {
	var e models.Enrollment
	var ended sql.NullTime
	err := rows.Scan(&e.ID, &e.StudentID, &e.StudentName, &e.AcademicYearID, &e.OfferingID, &e.BatchID,
		&e.BatchName, &e.SubjectName, &e.ClassName, &e.Status, &e.EnrolledAt, &ended, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if ended.Valid {
		e.EndedAt = &ended.Time
	}
	return &e, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate") || contains(err.Error(), "unique"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
