package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
)

type AcademicRepository struct {
	db *sql.DB
}

func NewAcademicRepository(db *sql.DB) *AcademicRepository {
	return &AcademicRepository{db: db}
}

// Academic Years

func (r *AcademicRepository) CreateYear(ctx context.Context, name, startDate, endDate string, isActive bool) (*models.AcademicYear, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if isActive {
		if _, err := tx.ExecContext(ctx, `UPDATE academic_years SET is_active = FALSE`); err != nil {
			return nil, err
		}
	}

	var y models.AcademicYear
	err = tx.QueryRowContext(ctx, `
		INSERT INTO academic_years (name, start_date, end_date, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, start_date::text, end_date::text, is_active, created_at, updated_at
	`, name, startDate, endDate, isActive).Scan(
		&y.ID, &y.Name, &y.StartDate, &y.EndDate, &y.IsActive, &y.CreatedAt, &y.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, httpx.Conflict("academic year name already exists")
		}
		return nil, err
	}
	return &y, tx.Commit()
}

func (r *AcademicRepository) GetYear(ctx context.Context, id string) (*models.AcademicYear, error) {
	var y models.AcademicYear
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, start_date::text, end_date::text, is_active, created_at, updated_at
		FROM academic_years WHERE id = $1
	`, id).Scan(&y.ID, &y.Name, &y.StartDate, &y.EndDate, &y.IsActive, &y.CreatedAt, &y.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("academic year not found")
	}
	return &y, err
}

func (r *AcademicRepository) ListYears(ctx context.Context, activeOnly *bool, offset, limit int) ([]models.AcademicYear, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if activeOnly != nil {
		args = append(args, *activeOnly)
		where += fmt.Sprintf(" AND is_active = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM academic_years `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, name, start_date::text, end_date::text, is_active, created_at, updated_at
		FROM academic_years %s ORDER BY start_date DESC LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var years []models.AcademicYear
	for rows.Next() {
		var y models.AcademicYear
		if err := rows.Scan(&y.ID, &y.Name, &y.StartDate, &y.EndDate, &y.IsActive, &y.CreatedAt, &y.UpdatedAt); err != nil {
			return nil, 0, err
		}
		years = append(years, y)
	}
	return years, total, rows.Err()
}

func (r *AcademicRepository) UpdateYear(ctx context.Context, id, name, startDate, endDate string, isActive *bool) (*models.AcademicYear, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if isActive != nil && *isActive {
		if _, err := tx.ExecContext(ctx, `UPDATE academic_years SET is_active = FALSE`); err != nil {
			return nil, err
		}
	}

	var y models.AcademicYear
	var activeVal sql.NullBool
	if isActive != nil {
		activeVal = sql.NullBool{Bool: *isActive, Valid: true}
	}

	err = tx.QueryRowContext(ctx, `
		UPDATE academic_years SET
			name = COALESCE(NULLIF($2, ''), name),
			start_date = COALESCE(NULLIF($3::date, NULL), start_date),
			end_date = COALESCE(NULLIF($4::date, NULL), end_date),
			is_active = COALESCE($5, is_active),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, start_date::text, end_date::text, is_active, created_at, updated_at
	`, id, name, nullIfEmpty(startDate), nullIfEmpty(endDate), activeVal).Scan(
		&y.ID, &y.Name, &y.StartDate, &y.EndDate, &y.IsActive, &y.CreatedAt, &y.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("academic year not found")
	}
	if err != nil {
		return nil, err
	}
	return &y, tx.Commit()
}

func (r *AcademicRepository) DeleteYear(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM academic_years WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return httpx.NotFound("academic year not found")
	}
	return nil
}

func (r *AcademicRepository) GetActiveYear(ctx context.Context) (*models.AcademicYear, error) {
	var y models.AcademicYear
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, start_date::text, end_date::text, is_active, created_at, updated_at
		FROM academic_years WHERE is_active = TRUE LIMIT 1
	`).Scan(&y.ID, &y.Name, &y.StartDate, &y.EndDate, &y.IsActive, &y.CreatedAt, &y.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("active academic year not found")
	}
	return &y, err
}

// Classes

func (r *AcademicRepository) CreateClass(ctx context.Context, name string, grade int) (*models.AcademicClass, error) {
	var c models.AcademicClass
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO academic_classes (name, grade)
		VALUES ($1, $2)
		RETURNING id, name, grade, created_at
	`, name, grade).Scan(&c.ID, &c.Name, &c.Grade, &c.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, httpx.Conflict("class name or grade already exists")
		}
		return nil, err
	}
	return &c, nil
}

func (r *AcademicRepository) ListClasses(ctx context.Context, grade *int, offset, limit int) ([]models.AcademicClass, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if grade != nil {
		args = append(args, *grade)
		where += fmt.Sprintf(" AND grade = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM academic_classes `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, name, grade, created_at FROM academic_classes %s ORDER BY grade LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var classes []models.AcademicClass
	for rows.Next() {
		var c models.AcademicClass
		if err := rows.Scan(&c.ID, &c.Name, &c.Grade, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		classes = append(classes, c)
	}
	return classes, total, rows.Err()
}

func (r *AcademicRepository) GetClass(ctx context.Context, id string) (*models.AcademicClass, error) {
	var c models.AcademicClass
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, grade, created_at FROM academic_classes WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Grade, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("class not found")
	}
	return &c, err
}

func (r *AcademicRepository) UpdateClass(ctx context.Context, id, name string, grade *int) (*models.AcademicClass, error) {
	var gradeVal sql.NullInt64
	if grade != nil {
		gradeVal = sql.NullInt64{Int64: int64(*grade), Valid: true}
	}

	var c models.AcademicClass
	err := r.db.QueryRowContext(ctx, `
		UPDATE academic_classes SET
			name = COALESCE(NULLIF($2, ''), name),
			grade = COALESCE($3, grade)
		WHERE id = $1
		RETURNING id, name, grade, created_at
	`, id, name, gradeVal).Scan(&c.ID, &c.Name, &c.Grade, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("class not found")
	}
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, httpx.Conflict("class name or grade already exists")
		}
		return nil, err
	}
	return &c, nil
}

func (r *AcademicRepository) DeleteClass(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM academic_classes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return httpx.NotFound("class not found")
	}
	return nil
}

// Subjects

func (r *AcademicRepository) CreateSubject(ctx context.Context, name, code string) (*models.Subject, error) {
	var s models.Subject
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO subjects (name, code) VALUES ($1, $2)
		RETURNING id, name, code, created_at
	`, name, code).Scan(&s.ID, &s.Name, &s.Code, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *AcademicRepository) ListSubjects(ctx context.Context, search string, offset, limit int) ([]models.Subject, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (name ILIKE $%d OR code ILIKE $%d)", len(args), len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subjects `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, name, code, created_at FROM subjects %s ORDER BY name LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var subjects []models.Subject
	for rows.Next() {
		var s models.Subject
		if err := rows.Scan(&s.ID, &s.Name, &s.Code, &s.CreatedAt); err != nil {
			return nil, 0, err
		}
		subjects = append(subjects, s)
	}
	return subjects, total, rows.Err()
}

func (r *AcademicRepository) GetSubject(ctx context.Context, id string) (*models.Subject, error) {
	var s models.Subject
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, code, created_at FROM subjects WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.Code, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("subject not found")
	}
	return &s, err
}

func (r *AcademicRepository) UpdateSubject(ctx context.Context, id, name, code string) (*models.Subject, error) {
	var s models.Subject
	err := r.db.QueryRowContext(ctx, `
		UPDATE subjects SET
			name = COALESCE(NULLIF($2, ''), name),
			code = COALESCE(NULLIF($3, ''), code)
		WHERE id = $1
		RETURNING id, name, code, created_at
	`, id, name, code).Scan(&s.ID, &s.Name, &s.Code, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("subject not found")
	}
	return &s, err
}

func (r *AcademicRepository) DeleteSubject(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM subjects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return httpx.NotFound("subject not found")
	}
	return nil
}

// Subject Offerings

func (r *AcademicRepository) CreateOffering(ctx context.Context, classID, subjectID string, fee float64, currency string) (*models.SubjectOffering, error) {
	var o models.SubjectOffering
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO subject_offerings (class_id, subject_id, fee_amount, fee_currency)
		VALUES ($1, $2, $3, $4)
		RETURNING id, class_id, subject_id, fee_amount, fee_currency, created_at, updated_at
	`, classID, subjectID, fee, currency).Scan(
		&o.ID, &o.ClassID, &o.SubjectID, &o.FeeAmount, &o.FeeCurrency, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *AcademicRepository) GetOffering(ctx context.Context, id string) (*models.SubjectOffering, error) {
	var o models.SubjectOffering
	err := r.db.QueryRowContext(ctx, `
		SELECT so.id, so.class_id, so.subject_id, so.fee_amount, so.fee_currency,
		       so.created_at, so.updated_at, ac.name, s.name
		FROM subject_offerings so
		JOIN academic_classes ac ON ac.id = so.class_id
		JOIN subjects s ON s.id = so.subject_id
		WHERE so.id = $1
	`, id).Scan(
		&o.ID, &o.ClassID, &o.SubjectID, &o.FeeAmount, &o.FeeCurrency,
		&o.CreatedAt, &o.UpdatedAt, &o.ClassName, &o.SubjectName,
	)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("offering not found")
	}
	return &o, err
}

func (r *AcademicRepository) ListOfferings(ctx context.Context, classID, subjectID string, offset, limit int) ([]models.SubjectOffering, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if classID != "" {
		args = append(args, classID)
		where += fmt.Sprintf(" AND so.class_id = $%d", len(args))
	}
	if subjectID != "" {
		args = append(args, subjectID)
		where += fmt.Sprintf(" AND so.subject_id = $%d", len(args))
	}

	countQuery := `
		SELECT COUNT(*) FROM subject_offerings so
		JOIN academic_classes ac ON ac.id = so.class_id
		JOIN subjects s ON s.id = so.subject_id ` + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT so.id, so.class_id, so.subject_id, so.fee_amount, so.fee_currency,
		       so.created_at, so.updated_at, ac.name, s.name
		FROM subject_offerings so
		JOIN academic_classes ac ON ac.id = so.class_id
		JOIN subjects s ON s.id = so.subject_id
		%s ORDER BY ac.grade, s.name LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var offerings []models.SubjectOffering
	for rows.Next() {
		var o models.SubjectOffering
		if err := rows.Scan(
			&o.ID, &o.ClassID, &o.SubjectID, &o.FeeAmount, &o.FeeCurrency,
			&o.CreatedAt, &o.UpdatedAt, &o.ClassName, &o.SubjectName,
		); err != nil {
			return nil, 0, err
		}
		offerings = append(offerings, o)
	}
	return offerings, total, rows.Err()
}

func (r *AcademicRepository) UpdateOffering(ctx context.Context, id, classID, subjectID string) (*models.SubjectOffering, error) {
	var classVal, subjectVal sql.NullString
	if classID != "" {
		classVal = sql.NullString{String: classID, Valid: true}
	}
	if subjectID != "" {
		subjectVal = sql.NullString{String: subjectID, Valid: true}
	}

	var o models.SubjectOffering
	err := r.db.QueryRowContext(ctx, `
		UPDATE subject_offerings SET
			class_id = COALESCE($2, class_id),
			subject_id = COALESCE($3, subject_id),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, class_id, subject_id, fee_amount, fee_currency, created_at, updated_at
	`, id, classVal, subjectVal).Scan(
		&o.ID, &o.ClassID, &o.SubjectID, &o.FeeAmount, &o.FeeCurrency, &o.CreatedAt, &o.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("offering not found")
	}
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, httpx.Conflict("offering for this class and subject already exists")
		}
		return nil, err
	}
	return &o, nil
}

func (r *AcademicRepository) UpdateOfferingFee(ctx context.Context, id string, fee float64, effectiveFrom string) (*models.SubjectOffering, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO subject_offering_fee_history (offering_id, fee_amount, effective_from)
		SELECT id, fee_amount, COALESCE($2::date, CURRENT_DATE) FROM subject_offerings WHERE id = $1
	`, id, effectiveFrom); err != nil {
		return nil, err
	}

	var o models.SubjectOffering
	err = tx.QueryRowContext(ctx, `
		UPDATE subject_offerings SET fee_amount = $2, updated_at = NOW() WHERE id = $1
		RETURNING id, class_id, subject_id, fee_amount, fee_currency, created_at, updated_at
	`, id, fee).Scan(
		&o.ID, &o.ClassID, &o.SubjectID, &o.FeeAmount, &o.FeeCurrency, &o.CreatedAt, &o.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("offering not found")
	}
	if err != nil {
		return nil, err
	}
	return &o, tx.Commit()
}

func (r *AcademicRepository) DeleteOffering(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM subject_offerings WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return httpx.NotFound("offering not found")
	}
	return nil
}

func (r *AcademicRepository) ListFeeHistory(ctx context.Context, offeringID string) ([]models.FeeHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, offering_id, fee_amount, effective_from::text, created_at
		FROM subject_offering_fee_history WHERE offering_id = $1 ORDER BY effective_from DESC
	`, offeringID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.FeeHistory
	for rows.Next() {
		var h models.FeeHistory
		if err := rows.Scan(&h.ID, &h.OfferingID, &h.FeeAmount, &h.EffectiveFrom, &h.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
