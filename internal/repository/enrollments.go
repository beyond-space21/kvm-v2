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
	if err := r.validateEnrollment(ctx, studentID, offeringID, batchID); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var feeAmount float64
	var currency string
	err = tx.QueryRowContext(ctx, `
		SELECT fee_amount, fee_currency FROM subject_offerings WHERE id = $1
	`, offeringID).Scan(&feeAmount, &currency)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("offering not found")
	}
	if err != nil {
		return nil, err
	}

	var e models.Enrollment
	err = tx.QueryRowContext(ctx, `
		INSERT INTO enrollments (student_id, academic_year_id, offering_id, batch_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, student_id, academic_year_id, offering_id, batch_id, status, enrolled_at, ended_at, created_at, updated_at
	`, studentID, yearID, offeringID, batchID).Scan(
		&e.ID, &e.StudentID, &e.AcademicYearID, &e.OfferingID, &e.BatchID, &e.Status,
		&e.EnrolledAt, &e.EndedAt, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, httpx.Conflict("student already has an active enrollment for this offering")
		}
		if contains(err.Error(), "batch does not belong to offering") {
			return nil, httpx.InvalidInput("batch does not belong to offering")
		}
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO fee_invoices (enrollment_id, amount, currency, status)
		VALUES ($1, $2, $3, 'pending')
	`, e.ID, feeAmount, currency)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, e.ID)
}

func (r *EnrollmentRepository) validateEnrollment(ctx context.Context, studentID, offeringID, batchID string) error {
	var studentRole string
	var studentClassID sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT role, class_id FROM users WHERE id = $1
	`, studentID).Scan(&studentRole, &studentClassID)
	if err == sql.ErrNoRows {
		return httpx.NotFound("student not found")
	}
	if err != nil {
		return err
	}
	if studentRole != models.RoleStudent {
		return httpx.InvalidInput("user is not a student")
	}
	if !studentClassID.Valid {
		return httpx.InvalidInput("student has no class assigned")
	}

	var offeringClassID string
	err = r.db.QueryRowContext(ctx, `
		SELECT class_id FROM subject_offerings WHERE id = $1
	`, offeringID).Scan(&offeringClassID)
	if err == sql.ErrNoRows {
		return httpx.NotFound("offering not found")
	}
	if err != nil {
		return err
	}
	if offeringClassID != studentClassID.String {
		return httpx.InvalidInput("offering class does not match student class")
	}

	var batchOfferingID string
	err = r.db.QueryRowContext(ctx, `
		SELECT offering_id FROM batches WHERE id = $1
	`, batchID).Scan(&batchOfferingID)
	if err == sql.ErrNoRows {
		return httpx.NotFound("batch not found")
	}
	if err != nil {
		return err
	}
	if batchOfferingID != offeringID {
		return httpx.InvalidInput("batch does not belong to offering")
	}
	return nil
}

func (r *EnrollmentRepository) Get(ctx context.Context, id string) (*models.Enrollment, error) {
	return r.scanOne(r.db.QueryRowContext(ctx, enrollmentSelect+` WHERE e.id = $1`, id))
}

func (r *EnrollmentRepository) List(ctx context.Context, studentID, yearID, offeringID, batchID, status string, offset, limit int) ([]models.Enrollment, int, error) {
	where, args := r.filters(studentID, yearID, offeringID, batchID, status)

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
		return nil, httpx.NotFound("active enrollment not found")
	}
	if err != nil {
		return nil, err
	}

	var newBatchOfferingID string
	err = tx.QueryRowContext(ctx, `SELECT offering_id FROM batches WHERE id = $1`, newBatchID).Scan(&newBatchOfferingID)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("batch not found")
	}
	if err != nil {
		return nil, err
	}
	if newBatchOfferingID != old.OfferingID {
		return nil, httpx.InvalidInput("transfer batch must belong to the same offering")
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
		if contains(err.Error(), "batch does not belong to offering") {
			return nil, httpx.InvalidInput("batch does not belong to offering")
		}
		return nil, err
	}

	var feeAmount float64
	var currency string
	if err := tx.QueryRowContext(ctx, `
		SELECT amount, currency FROM fee_invoices WHERE enrollment_id = $1
	`, id).Scan(&feeAmount, &currency); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO fee_invoices (enrollment_id, amount, currency, status)
		VALUES ($1, $2, $3, 'pending')
	`, e.ID, feeAmount, currency); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, e.ID)
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
		return httpx.NotFound("active enrollment not found")
	}
	return nil
}

func (r *EnrollmentRepository) History(ctx context.Context, studentID, yearID, offeringID, status string, offset, limit int) ([]models.Enrollment, int, error) {
	where := "WHERE e.student_id = $1"
	args := []any{studentID}
	if yearID != "" {
		args = append(args, yearID)
		where += fmt.Sprintf(" AND e.academic_year_id = $%d", len(args))
	}
	if offeringID != "" {
		args = append(args, offeringID)
		where += fmt.Sprintf(" AND e.offering_id = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND e.status = $%d", len(args))
	}

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

func (r *EnrollmentRepository) filters(studentID, yearID, offeringID, batchID, status string) (string, []any) {
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
	if offeringID != "" {
		args = append(args, offeringID)
		where += fmt.Sprintf(" AND e.offering_id = $%d", len(args))
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
		return nil, httpx.NotFound("enrollment not found")
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
