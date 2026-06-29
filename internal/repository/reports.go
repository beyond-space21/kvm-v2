package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

type AttendanceReport struct {
	StudentID   string  `json:"student_id,omitempty"`
	StudentName string  `json:"student_name,omitempty"`
	BatchID     string  `json:"batch_id,omitempty"`
	BatchName   string  `json:"batch_name,omitempty"`
	SubjectName string  `json:"subject_name,omitempty"`
	TeacherName string  `json:"teacher_name,omitempty"`
	Present     int     `json:"present"`
	Absent      int     `json:"absent"`
	Late        int     `json:"late"`
	Total       int     `json:"total"`
	Percentage  float64 `json:"percentage"`
}

type EnrollmentReport struct {
	StudentID   string  `json:"student_id"`
	StudentName string  `json:"student_name"`
	ClassName   string  `json:"class_name"`
	SubjectName string  `json:"subject_name"`
	BatchName   string  `json:"batch_name"`
	Status      string  `json:"status"`
	FeeAmount   float64 `json:"fee_amount"`
}

type FeeSummary struct {
	StudentID    string  `json:"student_id"`
	StudentName  string  `json:"student_name"`
	TotalFee     float64 `json:"total_fee"`
	TotalPaid    float64 `json:"total_paid"`
	TotalDue     float64 `json:"total_due"`
	SubjectCount int     `json:"subject_count"`
}

func (r *ReportRepository) StudentAttendance(ctx context.Context, studentID string) ([]AttendanceReport, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT e.batch_id, b.name, s.name,
			COUNT(*) FILTER (WHERE ar.status = 'present') AS present,
			COUNT(*) FILTER (WHERE ar.status = 'absent') AS absent,
			COUNT(*) FILTER (WHERE ar.status = 'late') AS late,
			COUNT(ses.id) AS total
		FROM enrollments e
		JOIN batches b ON b.id = e.batch_id
		JOIN subject_offerings so ON so.id = e.offering_id
		JOIN subjects s ON s.id = so.subject_id
		LEFT JOIN sessions ses ON ses.batch_id = e.batch_id AND ses.status = 'scheduled'
		LEFT JOIN attendance_records ar ON ar.session_id = ses.id AND ar.student_id = e.student_id
		WHERE e.student_id = $1 AND e.status = 'active'
		GROUP BY e.batch_id, b.name, s.name
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanAttendanceReports(rows)
}

func (r *ReportRepository) BatchReport(ctx context.Context, batchID string) ([]AttendanceReport, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.name,
			COUNT(*) FILTER (WHERE ar.status = 'present') AS present,
			COUNT(*) FILTER (WHERE ar.status = 'absent') AS absent,
			COUNT(*) FILTER (WHERE ar.status = 'late') AS late,
			COUNT(ses.id) AS total
		FROM enrollments e
		JOIN users u ON u.id = e.student_id
		LEFT JOIN sessions ses ON ses.batch_id = e.batch_id AND ses.status = 'scheduled'
		LEFT JOIN attendance_records ar ON ar.session_id = ses.id AND ar.student_id = e.student_id
		WHERE e.batch_id = $1 AND e.status = 'active'
		GROUP BY u.id, u.name ORDER BY u.name
	`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []AttendanceReport
	for rows.Next() {
		var rep AttendanceReport
		if err := rows.Scan(&rep.StudentID, &rep.StudentName, &rep.Present, &rep.Absent, &rep.Late, &rep.Total); err != nil {
			return nil, err
		}
		rep.Percentage = pct(rep.Present+rep.Late, rep.Total)
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

func (r *ReportRepository) SubjectReport(ctx context.Context, subjectID string) ([]AttendanceReport, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.id, b.name, u.name,
			COUNT(*) FILTER (WHERE ar.status = 'present') AS present,
			COUNT(*) FILTER (WHERE ar.status = 'absent') AS absent,
			COUNT(*) FILTER (WHERE ar.status = 'late') AS late,
			COUNT(ses.id) AS total
		FROM batches b
		JOIN subject_offerings so ON so.id = b.offering_id
		JOIN enrollments e ON e.batch_id = b.id AND e.status = 'active'
		JOIN users u ON u.id = b.teacher_id
		LEFT JOIN sessions ses ON ses.batch_id = b.id AND ses.status = 'scheduled'
		LEFT JOIN attendance_records ar ON ar.session_id = ses.id
		WHERE so.subject_id = $1
		GROUP BY b.id, b.name, u.name
	`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []AttendanceReport
	for rows.Next() {
		var rep AttendanceReport
		if err := rows.Scan(&rep.BatchID, &rep.BatchName, &rep.TeacherName, &rep.Present, &rep.Absent, &rep.Late, &rep.Total); err != nil {
			return nil, err
		}
		rep.Percentage = pct(rep.Present+rep.Late, rep.Total)
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

func (r *ReportRepository) TeacherReport(ctx context.Context, teacherID string) ([]AttendanceReport, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.id, b.name, s.name,
			COUNT(*) FILTER (WHERE ar.status = 'present') AS present,
			COUNT(*) FILTER (WHERE ar.status = 'absent') AS absent,
			COUNT(*) FILTER (WHERE ar.status = 'late') AS late,
			COUNT(ses.id) AS total
		FROM batches b
		JOIN subject_offerings so ON so.id = b.offering_id
		JOIN subjects s ON s.id = so.subject_id
		LEFT JOIN sessions ses ON ses.batch_id = b.id AND ses.status = 'scheduled'
		LEFT JOIN attendance_records ar ON ar.session_id = ses.id
		WHERE b.teacher_id = $1
		GROUP BY b.id, b.name, s.name
	`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanBatchSubjectReports(rows)
}

func (r *ReportRepository) DailyReport(ctx context.Context, date, teacherID string) ([]AttendanceReport, error) {
	query := `
		SELECT b.id, b.name, s.name,
			COUNT(*) FILTER (WHERE ar.status = 'present') AS present,
			COUNT(*) FILTER (WHERE ar.status = 'absent') AS absent,
			COUNT(*) FILTER (WHERE ar.status = 'late') AS late,
			COUNT(ses.id) AS total
		FROM sessions ses
		JOIN batches b ON b.id = ses.batch_id
		JOIN subject_offerings so ON so.id = b.offering_id
		JOIN subjects s ON s.id = so.subject_id
		LEFT JOIN attendance_records ar ON ar.session_id = ses.id
		WHERE ses.session_date = $1::date AND ses.status = 'scheduled'`
	args := []any{date}
	if teacherID != "" {
		args = append(args, teacherID)
		query += fmt.Sprintf(` AND b.teacher_id = $%d`, len(args))
	}
	query += ` GROUP BY b.id, b.name, s.name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanBatchSubjectReports(rows)
}

func (r *ReportRepository) MonthlyReport(ctx context.Context, month, teacherID string) ([]AttendanceReport, error) {
	query := `
		SELECT b.id, b.name, s.name,
			COUNT(*) FILTER (WHERE ar.status = 'present') AS present,
			COUNT(*) FILTER (WHERE ar.status = 'absent') AS absent,
			COUNT(*) FILTER (WHERE ar.status = 'late') AS late,
			COUNT(ses.id) AS total
		FROM sessions ses
		JOIN batches b ON b.id = ses.batch_id
		JOIN subject_offerings so ON so.id = b.offering_id
		JOIN subjects s ON s.id = so.subject_id
		LEFT JOIN attendance_records ar ON ar.session_id = ses.id
		WHERE to_char(ses.session_date, 'YYYY-MM') = $1 AND ses.status = 'scheduled'`
	args := []any{month}
	if teacherID != "" {
		args = append(args, teacherID)
		query += fmt.Sprintf(` AND b.teacher_id = $%d`, len(args))
	}
	query += ` GROUP BY b.id, b.name, s.name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanBatchSubjectReports(rows)
}

func (r *ReportRepository) EnrollmentReport(ctx context.Context, yearID, classID, batchID string, offset, limit int) ([]EnrollmentReport, int, error) {
	where := "WHERE e.status = 'active'"
	args := []any{}
	if yearID != "" {
		args = append(args, yearID)
		where += ` AND e.academic_year_id = $` + fmt.Sprint(len(args))
	}
	if classID != "" {
		args = append(args, classID)
		where += ` AND so.class_id = $` + fmt.Sprint(len(args))
	}
	if batchID != "" {
		args = append(args, batchID)
		where += ` AND e.batch_id = $` + fmt.Sprint(len(args))
	}

	from := `
		FROM enrollments e
		JOIN users u ON u.id = e.student_id
		JOIN batches b ON b.id = e.batch_id
		JOIN subject_offerings so ON so.id = e.offering_id
		JOIN subjects s ON s.id = so.subject_id
		JOIN academic_classes ac ON ac.id = so.class_id`

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) `+from+` `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT u.id, u.name, ac.name, s.name, b.name, e.status, so.fee_amount
		%s %s ORDER BY u.name, s.name LIMIT $%d OFFSET $%d
	`, from, where, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []EnrollmentReport
	for rows.Next() {
		var rep EnrollmentReport
		if err := rows.Scan(&rep.StudentID, &rep.StudentName, &rep.ClassName, &rep.SubjectName,
			&rep.BatchName, &rep.Status, &rep.FeeAmount); err != nil {
			return nil, 0, err
		}
		reports = append(reports, rep)
	}
	return reports, total, rows.Err()
}

func (r *ReportRepository) FeeSummary(ctx context.Context, yearID, classID string, offset, limit int) ([]FeeSummary, int, error) {
	where := "WHERE e.status = 'active'"
	args := []any{}
	if yearID != "" {
		args = append(args, yearID)
		where += ` AND e.academic_year_id = $` + fmt.Sprint(len(args))
	}
	if classID != "" {
		args = append(args, classID)
		where += ` AND u.class_id = $` + fmt.Sprint(len(args))
	}

	from := `
		FROM enrollments e
		JOIN users u ON u.id = e.student_id
		LEFT JOIN fee_invoices fi ON fi.enrollment_id = e.id
		LEFT JOIN (
			SELECT invoice_id, SUM(amount) AS total_paid
			FROM fee_payments
			GROUP BY invoice_id
		) paid ON paid.invoice_id = fi.id`

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT u.id) `+from+` `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT u.id, u.name,
		       COALESCE(SUM(fi.amount), 0),
		       COALESCE(SUM(paid.total_paid), 0),
		       COUNT(DISTINCT e.offering_id)
		%s %s GROUP BY u.id, u.name ORDER BY u.name LIMIT $%d OFFSET $%d
	`, from, where, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var summaries []FeeSummary
	for rows.Next() {
		var s FeeSummary
		if err := rows.Scan(&s.StudentID, &s.StudentName, &s.TotalFee, &s.TotalPaid, &s.SubjectCount); err != nil {
			return nil, 0, err
		}
		s.TotalDue = s.TotalFee - s.TotalPaid
		if s.TotalDue < 0 {
			s.TotalDue = 0
		}
		summaries = append(summaries, s)
	}
	return summaries, total, rows.Err()
}

func (r *ReportRepository) scanAttendanceReports(rows *sql.Rows) ([]AttendanceReport, error) {
	var reports []AttendanceReport
	for rows.Next() {
		var rep AttendanceReport
		if err := rows.Scan(&rep.BatchID, &rep.BatchName, &rep.SubjectName,
			&rep.Present, &rep.Absent, &rep.Late, &rep.Total); err != nil {
			return nil, err
		}
		rep.Percentage = pct(rep.Present+rep.Late, rep.Total)
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

func (r *ReportRepository) scanBatchSubjectReports(rows *sql.Rows) ([]AttendanceReport, error) {
	var reports []AttendanceReport
	for rows.Next() {
		var rep AttendanceReport
		if err := rows.Scan(&rep.BatchID, &rep.BatchName, &rep.SubjectName,
			&rep.Present, &rep.Absent, &rep.Late, &rep.Total); err != nil {
			return nil, err
		}
		rep.Percentage = pct(rep.Present+rep.Late, rep.Total)
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

func pct(present, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(present) / float64(total) * 100
}
