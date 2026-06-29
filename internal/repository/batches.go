package repository

import (
	"context"
	"database/sql"
	"fmt"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
)

type BatchRepository struct {
	db *sql.DB
}

func NewBatchRepository(db *sql.DB) *BatchRepository {
	return &BatchRepository{db: db}
}

func (r *BatchRepository) Create(ctx context.Context, offeringID, name string, teacherID *string, capacity *int) (*models.Batch, error) {
	var b models.Batch
	var teacher sql.NullString
	var cap sql.NullInt64
	if teacherID != nil {
		teacher = sql.NullString{String: *teacherID, Valid: true}
	}
	if capacity != nil {
		cap = sql.NullInt64{Int64: int64(*capacity), Valid: true}
	}

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO batches (offering_id, name, teacher_id, capacity)
		VALUES ($1, $2, $3, $4)
		RETURNING id, offering_id, name, teacher_id, capacity, status, created_at, updated_at
	`, offeringID, name, teacher, cap).Scan(
		&b.ID, &b.OfferingID, &b.Name, &teacher, &cap, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if teacher.Valid {
		b.TeacherID = &teacher.String
	}
	if cap.Valid {
		c := int(cap.Int64)
		b.Capacity = &c
	}
	return &b, nil
}

func (r *BatchRepository) Get(ctx context.Context, id string) (*models.Batch, error) {
	return r.scanBatch(r.db.QueryRowContext(ctx, batchSelect+` WHERE b.id = $1`, id))
}

func (r *BatchRepository) List(ctx context.Context, offeringID, teacherID, status string, offset, limit int) ([]models.Batch, int, error) {
	where, args := r.batchFilters(offeringID, teacherID, status)

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM batches b `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := batchSelect + where + fmt.Sprintf(` ORDER BY b.created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var batches []models.Batch
	for rows.Next() {
		b, err := r.scanBatchRow(rows)
		if err != nil {
			return nil, 0, err
		}
		batches = append(batches, *b)
	}
	return batches, total, rows.Err()
}

func (r *BatchRepository) Update(ctx context.Context, id string, name string, teacherID *string, capacity *int, status *string) (*models.Batch, error) {
	var teacher sql.NullString
	if teacherID != nil {
		teacher = sql.NullString{String: *teacherID, Valid: true}
	}
	var cap sql.NullInt64
	if capacity != nil {
		cap = sql.NullInt64{Int64: int64(*capacity), Valid: true}
	}
	var statusVal sql.NullString
	if status != nil {
		statusVal = sql.NullString{String: *status, Valid: true}
	}

	row := r.db.QueryRowContext(ctx, `
		UPDATE batches SET
			name = COALESCE(NULLIF($2, ''), name),
			teacher_id = COALESCE($3, teacher_id),
			capacity = COALESCE($4, capacity),
			status = COALESCE($5, status),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, offering_id, name, teacher_id, capacity, status, created_at, updated_at
	`, id, name, teacher, cap, statusVal)

	var b models.Batch
	var t sql.NullString
	var c sql.NullInt64
	err := row.Scan(&b.ID, &b.OfferingID, &b.Name, &t, &c, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if t.Valid {
		b.TeacherID = &t.String
	}
	if c.Valid {
		v := int(c.Int64)
		b.Capacity = &v
	}
	return &b, nil
}

func (r *BatchRepository) ListStudents(ctx context.Context, batchID string) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.email, u.name, u.role, u.status, u.phone, u.created_at, u.updated_at
		FROM enrollments e
		JOIN users u ON u.id = e.student_id
		WHERE e.batch_id = $1 AND e.status = 'active'
		ORDER BY u.name
	`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var phone sql.NullString
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Status, &phone, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		if phone.Valid {
			u.Phone = &phone.String
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *BatchRepository) IsTeacherOf(ctx context.Context, batchID, teacherID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM batches WHERE id = $1 AND teacher_id = $2)
	`, batchID, teacherID).Scan(&exists)
	return exists, err
}

const batchSelect = `
	SELECT b.id, b.offering_id, b.name, b.teacher_id, b.capacity, b.status, b.created_at, b.updated_at,
	       COALESCE(u.name, ''), s.name, ac.name
	FROM batches b
	JOIN subject_offerings so ON so.id = b.offering_id
	JOIN subjects s ON s.id = so.subject_id
	JOIN academic_classes ac ON ac.id = so.class_id
	LEFT JOIN users u ON u.id = b.teacher_id
`

func (r *BatchRepository) batchFilters(offeringID, teacherID, status string) (string, []any) {
	where := "WHERE 1=1"
	args := []any{}
	if offeringID != "" {
		args = append(args, offeringID)
		where += fmt.Sprintf(" AND b.offering_id = $%d", len(args))
	}
	if teacherID != "" {
		args = append(args, teacherID)
		where += fmt.Sprintf(" AND b.teacher_id = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND b.status = $%d", len(args))
	}
	return where, args
}

func (r *BatchRepository) scanBatch(row *sql.Row) (*models.Batch, error) {
	var b models.Batch
	var teacher sql.NullString
	var cap sql.NullInt64
	var teacherName, subjectName, className string
	err := row.Scan(&b.ID, &b.OfferingID, &b.Name, &teacher, &cap, &b.Status, &b.CreatedAt, &b.UpdatedAt,
		&teacherName, &subjectName, &className)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if teacher.Valid {
		b.TeacherID = &teacher.String
	}
	if cap.Valid {
		v := int(cap.Int64)
		b.Capacity = &v
	}
	b.TeacherName = teacherName
	b.SubjectName = subjectName
	b.ClassName = className
	return &b, nil
}

func (r *BatchRepository) scanBatchRow(rows *sql.Rows) (*models.Batch, error) {
	var b models.Batch
	var teacher sql.NullString
	var cap sql.NullInt64
	var teacherName, subjectName, className string
	err := rows.Scan(&b.ID, &b.OfferingID, &b.Name, &teacher, &cap, &b.Status, &b.CreatedAt, &b.UpdatedAt,
		&teacherName, &subjectName, &className)
	if err != nil {
		return nil, err
	}
	if teacher.Valid {
		b.TeacherID = &teacher.String
	}
	if cap.Valid {
		v := int(cap.Int64)
		b.Capacity = &v
	}
	b.TeacherName = teacherName
	b.SubjectName = subjectName
	b.ClassName = className
	return &b, nil
}
