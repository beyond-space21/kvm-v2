package repository

import (
	"context"
	"database/sql"
	"fmt"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
)

type AttendanceRepository struct {
	db *sql.DB
}

func NewAttendanceRepository(db *sql.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

type AttendanceMark struct {
	StudentID string `json:"student_id"`
	Status    string `json:"status"`
}

func (r *AttendanceRepository) BulkMark(ctx context.Context, sessionID, markedBy string, marks []AttendanceMark, lock bool) ([]models.AttendanceRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var records []models.AttendanceRecord
	for _, m := range marks {
		var rec models.AttendanceRecord
		err := tx.QueryRowContext(ctx, `
			INSERT INTO attendance_records (session_id, student_id, status, marked_by, locked)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (session_id, student_id) DO UPDATE SET
				status = EXCLUDED.status,
				marked_by = EXCLUDED.marked_by,
				marked_at = NOW(),
				locked = EXCLUDED.locked
			RETURNING id, session_id, student_id, status, marked_by, marked_at, locked
		`, sessionID, m.StudentID, m.Status, markedBy, lock).Scan(
			&rec.ID, &rec.SessionID, &rec.StudentID, &rec.Status, &rec.MarkedBy, &rec.MarkedAt, &rec.Locked,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	return records, tx.Commit()
}

func (r *AttendanceRepository) GetBySession(ctx context.Context, sessionID string) ([]models.AttendanceRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ar.id, ar.session_id, ar.student_id, u.name, ar.status, ar.marked_by, ar.marked_at, ar.locked
		FROM attendance_records ar
		JOIN users u ON u.id = ar.student_id
		WHERE ar.session_id = $1 ORDER BY u.name
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.AttendanceRecord
	for rows.Next() {
		var rec models.AttendanceRecord
		if err := rows.Scan(&rec.ID, &rec.SessionID, &rec.StudentID, &rec.StudentName,
			&rec.Status, &rec.MarkedBy, &rec.MarkedAt, &rec.Locked); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *AttendanceRepository) GetByStudent(ctx context.Context, studentID string) ([]models.AttendanceRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ar.id, ar.session_id, ar.student_id, u.name, ar.status, ar.marked_by, ar.marked_at, ar.locked
		FROM attendance_records ar
		JOIN users u ON u.id = ar.student_id
		WHERE ar.student_id = $1 ORDER BY ar.marked_at DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.AttendanceRecord
	for rows.Next() {
		var rec models.AttendanceRecord
		if err := rows.Scan(&rec.ID, &rec.SessionID, &rec.StudentID, &rec.StudentName,
			&rec.Status, &rec.MarkedBy, &rec.MarkedAt, &rec.Locked); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *AttendanceRepository) Get(ctx context.Context, id string) (*models.AttendanceRecord, error) {
	var rec models.AttendanceRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, session_id, student_id, status, marked_by, marked_at, locked
		FROM attendance_records WHERE id = $1
	`, id).Scan(&rec.ID, &rec.SessionID, &rec.StudentID, &rec.Status, &rec.MarkedBy, &rec.MarkedAt, &rec.Locked)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	return &rec, err
}

func (r *AttendanceRepository) AdminEdit(ctx context.Context, id, newStatus, actorID string, reason *string) (*models.AttendanceRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var oldStatus string
	err = tx.QueryRowContext(ctx, `SELECT status FROM attendance_records WHERE id = $1 FOR UPDATE`, id).Scan(&oldStatus)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO attendance_audit_log (attendance_id, old_status, new_status, actor_id, reason)
		VALUES ($1, $2, $3, $4, $5)
	`, id, oldStatus, newStatus, actorID, reason); err != nil {
		return nil, err
	}

	var rec models.AttendanceRecord
	err = tx.QueryRowContext(ctx, `
		UPDATE attendance_records SET status = $2, marked_by = $3, marked_at = NOW()
		WHERE id = $1
		RETURNING id, session_id, student_id, status, marked_by, marked_at, locked
	`, id, newStatus, actorID).Scan(
		&rec.ID, &rec.SessionID, &rec.StudentID, &rec.Status, &rec.MarkedBy, &rec.MarkedAt, &rec.Locked,
	)
	if err != nil {
		return nil, err
	}
	return &rec, tx.Commit()
}

func (r *AttendanceRepository) StudentPercentage(ctx context.Context, studentID, batchID string) (float64, int, int, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE ar.status IN ('present', 'late')) AS present,
			COUNT(*) AS total
		FROM sessions s
		JOIN enrollments e ON e.batch_id = s.batch_id AND e.student_id = $1 AND e.status = 'active'
		LEFT JOIN attendance_records ar ON ar.session_id = s.id AND ar.student_id = $1
		WHERE s.status = 'scheduled'`
	args := []any{studentID}
	if batchID != "" {
		args = append(args, batchID)
		query += fmt.Sprintf(` AND s.batch_id = $%d`, len(args))
	}

	var present, total int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&present, &total)
	if err != nil {
		return 0, 0, 0, err
	}
	if total == 0 {
		return 0, 0, 0, nil
	}
	return float64(present) / float64(total) * 100, present, total, nil
}
