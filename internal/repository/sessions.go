package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) CreateTemplate(ctx context.Context, batchID, teacherID string, dayOfWeek int, startTime, endTime string) (*models.SessionTemplate, error) {
	var t models.SessionTemplate
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO session_templates (batch_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4::time, $5::time)
		RETURNING id, batch_id, teacher_id, day_of_week, start_time::text, end_time::text, created_at, updated_at
	`, batchID, teacherID, dayOfWeek, startTime, endTime).Scan(
		&t.ID, &t.BatchID, &t.TeacherID, &t.DayOfWeek, &t.StartTime, &t.EndTime, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *SessionRepository) ListTemplates(ctx context.Context, batchID string) ([]models.SessionTemplate, error) {
	query := `
		SELECT st.id, st.batch_id, st.teacher_id, u.name, b.name, st.day_of_week,
		       st.start_time::text, st.end_time::text, st.created_at, st.updated_at
		FROM session_templates st
		JOIN users u ON u.id = st.teacher_id
		JOIN batches b ON b.id = st.batch_id
		WHERE 1=1`
	args := []any{}
	if batchID != "" {
		args = append(args, batchID)
		query += ` AND st.batch_id = $1`
	}
	query += ` ORDER BY st.day_of_week, st.start_time`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.SessionTemplate
	for rows.Next() {
		var t models.SessionTemplate
		if err := rows.Scan(&t.ID, &t.BatchID, &t.TeacherID, &t.TeacherName, &t.BatchName,
			&t.DayOfWeek, &t.StartTime, &t.EndTime, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *SessionRepository) UpdateTemplate(ctx context.Context, id string, dayOfWeek *int, startTime, endTime string, teacherID *string) (*models.SessionTemplate, error) {
	var day sql.NullInt64
	if dayOfWeek != nil {
		day = sql.NullInt64{Int64: int64(*dayOfWeek), Valid: true}
	}
	var teacher sql.NullString
	if teacherID != nil {
		teacher = sql.NullString{String: *teacherID, Valid: true}
	}

	var t models.SessionTemplate
	err := r.db.QueryRowContext(ctx, `
		UPDATE session_templates SET
			day_of_week = COALESCE($2, day_of_week),
			start_time = COALESCE(NULLIF($3::time, NULL), start_time),
			end_time = COALESCE(NULLIF($4::time, NULL), end_time),
			teacher_id = COALESCE($5, teacher_id),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, batch_id, teacher_id, day_of_week, start_time::text, end_time::text, created_at, updated_at
	`, id, day, nullIfEmpty(startTime), nullIfEmpty(endTime), teacher).Scan(
		&t.ID, &t.BatchID, &t.TeacherID, &t.DayOfWeek, &t.StartTime, &t.EndTime, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	return &t, err
}

func (r *SessionRepository) GenerateSessions(ctx context.Context, startDate, endDate string, batchID *string) (int, error) {
	query := `
		SELECT st.id, st.batch_id, st.start_time::text, st.end_time::text, st.day_of_week
		FROM session_templates st
		JOIN batches b ON b.id = st.batch_id
		WHERE b.status = 'active'`
	args := []any{}
	if batchID != nil {
		args = append(args, *batchID)
		query += fmt.Sprintf(` AND st.batch_id = $%d`, len(args))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type templateInfo struct {
		id, batchID, start, end string
		dayOfWeek               int
	}
	var templates []templateInfo
	for rows.Next() {
		var t templateInfo
		if err := rows.Scan(&t.id, &t.batchID, &t.start, &t.end, &t.dayOfWeek); err != nil {
			return 0, err
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, httpx.ErrInvalidInput
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return 0, httpx.ErrInvalidInput
	}

	created := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		for _, t := range templates {
			if int(d.Weekday()) != t.dayOfWeek {
				continue
			}
			res, err := r.db.ExecContext(ctx, `
				INSERT INTO sessions (template_id, batch_id, session_date, start_time, end_time)
				VALUES ($1, $2, $3, $4::time, $5::time)
				ON CONFLICT (template_id, session_date) DO NOTHING
			`, t.id, t.batchID, d.Format("2006-01-02"), t.start, t.end)
			if err != nil {
				return created, err
			}
			n, _ := res.RowsAffected()
			created += int(n)
		}
	}
	return created, nil
}

func (r *SessionRepository) ListToday(ctx context.Context, teacherID string) ([]models.Session, error) {
	query := sessionSelect + ` WHERE s.session_date = CURRENT_DATE AND s.status = 'scheduled'`
	args := []any{}
	if teacherID != "" {
		args = append(args, teacherID)
		query += ` AND b.teacher_id = $1`
	}
	query += ` ORDER BY s.start_time`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSessions(rows)
}

func (r *SessionRepository) GetSession(ctx context.Context, id string) (*models.Session, error) {
	row := r.db.QueryRowContext(ctx, sessionSelect+` WHERE s.id = $1`, id)
	var s models.Session
	err := row.Scan(&s.ID, &s.TemplateID, &s.BatchID, &s.BatchName, &s.SubjectName,
		&s.SessionDate, &s.StartTime, &s.EndTime, &s.Status, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	return &s, err
}

func (r *SessionRepository) CancelSession(ctx context.Context, id string) (*models.Session, error) {
	var s models.Session
	err := r.db.QueryRowContext(ctx, `
		UPDATE sessions SET status = 'cancelled' WHERE id = $1
		RETURNING id, template_id, batch_id, session_date::text, start_time::text, end_time::text, status, created_at
	`, id).Scan(&s.ID, &s.TemplateID, &s.BatchID, &s.SessionDate, &s.StartTime, &s.EndTime, &s.Status, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.ErrNotFound
	}
	return &s, err
}

const sessionSelect = `
	SELECT s.id, s.template_id, s.batch_id, b.name, sub.name,
	       s.session_date::text, s.start_time::text, s.end_time::text, s.status, s.created_at
	FROM sessions s
	JOIN batches b ON b.id = s.batch_id
	JOIN subject_offerings so ON so.id = b.offering_id
	JOIN subjects sub ON sub.id = so.subject_id
`

func (r *SessionRepository) scanSessions(rows *sql.Rows) ([]models.Session, error) {
	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		if err := rows.Scan(&s.ID, &s.TemplateID, &s.BatchID, &s.BatchName, &s.SubjectName,
			&s.SessionDate, &s.StartTime, &s.EndTime, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}
