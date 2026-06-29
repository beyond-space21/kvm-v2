package models

import (
	"time"
)

const (
	RoleSuperAdmin = "super_admin"
	RoleStaff      = "staff"
	RoleStudent    = "student"

	StatusActive   = "active"
	StatusInactive = "inactive"

	BatchStatusActive   = "active"
	BatchStatusDisabled = "disabled"

	EnrollmentActive     = "active"
	EnrollmentTransferred = "transferred"
	EnrollmentRemoved    = "removed"

	SessionScheduled = "scheduled"
	SessionCancelled = "cancelled"

	AttendancePresent = "present"
	AttendanceAbsent  = "absent"
	AttendanceLate    = "late"

	InvoicePending = "pending"
	InvoicePartial = "partial"
	InvoicePaid    = "paid"
	InvoiceWaived  = "waived"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	Phone     *string   `json:"phone,omitempty"`
	ClassID   *string   `json:"class_id,omitempty"`
	ClassName string    `json:"class_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AcademicYear struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AcademicClass struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Grade     int       `json:"grade"`
	CreatedAt time.Time `json:"created_at"`
}

type Subject struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
}

type SubjectOffering struct {
	ID          string    `json:"id"`
	ClassID     string    `json:"class_id"`
	SubjectID   string    `json:"subject_id"`
	FeeAmount   float64   `json:"fee_amount"`
	FeeCurrency string    `json:"fee_currency"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ClassName   string    `json:"class_name,omitempty"`
	SubjectName string    `json:"subject_name,omitempty"`
}

type FeeHistory struct {
	ID            string    `json:"id"`
	OfferingID    string    `json:"offering_id"`
	FeeAmount     float64   `json:"fee_amount"`
	EffectiveFrom string    `json:"effective_from"`
	CreatedAt     time.Time `json:"created_at"`
}

type Batch struct {
	ID          string    `json:"id"`
	OfferingID  string    `json:"offering_id"`
	Name        string    `json:"name"`
	TeacherID   *string   `json:"teacher_id,omitempty"`
	TeacherName string    `json:"teacher_name,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	SubjectName string    `json:"subject_name,omitempty"`
	ClassName   string    `json:"class_name,omitempty"`
}

type Enrollment struct {
	ID             string     `json:"id"`
	StudentID      string     `json:"student_id"`
	StudentName    string     `json:"student_name,omitempty"`
	AcademicYearID string     `json:"academic_year_id"`
	OfferingID     string     `json:"offering_id"`
	BatchID        string     `json:"batch_id"`
	BatchName      string     `json:"batch_name,omitempty"`
	SubjectName    string     `json:"subject_name,omitempty"`
	ClassName      string     `json:"class_name,omitempty"`
	Status         string     `json:"status"`
	EnrolledAt     time.Time  `json:"enrolled_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type SessionTemplate struct {
	ID          string    `json:"id"`
	BatchID     string    `json:"batch_id"`
	TeacherID   string    `json:"teacher_id"`
	TeacherName string    `json:"teacher_name,omitempty"`
	BatchName   string    `json:"batch_name,omitempty"`
	DayOfWeek   int       `json:"day_of_week"`
	StartTime   string    `json:"start_time"`
	EndTime     string    `json:"end_time"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Session struct {
	ID          string    `json:"id"`
	TemplateID  string    `json:"template_id"`
	BatchID     string    `json:"batch_id"`
	BatchName   string    `json:"batch_name,omitempty"`
	SubjectName string    `json:"subject_name,omitempty"`
	SessionDate string    `json:"session_date"`
	StartTime   string    `json:"start_time"`
	EndTime     string    `json:"end_time"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type AttendanceRecord struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	StudentID   string    `json:"student_id"`
	StudentName string    `json:"student_name,omitempty"`
	Status      string    `json:"status"`
	MarkedBy    string    `json:"marked_by"`
	MarkedAt    time.Time `json:"marked_at"`
	Locked      bool      `json:"locked"`
}

type AttendanceAudit struct {
	ID           string    `json:"id"`
	AttendanceID string    `json:"attendance_id"`
	OldStatus    string    `json:"old_status"`
	NewStatus    string    `json:"new_status"`
	ActorID      string    `json:"actor_id"`
	Reason       *string   `json:"reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type ListResponse[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type FeeInvoice struct {
	ID           string    `json:"id"`
	EnrollmentID string    `json:"enrollment_id"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	Status       string    `json:"status"`
	DueDate      *string   `json:"due_date,omitempty"`
	PaidAmount   float64   `json:"paid_amount,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type FeePayment struct {
	ID         string    `json:"id"`
	InvoiceID  string    `json:"invoice_id"`
	Amount     float64   `json:"amount"`
	PaidAt     time.Time `json:"paid_at"`
	Method     *string   `json:"method,omitempty"`
	Reference  *string   `json:"reference,omitempty"`
	RecordedBy string    `json:"recorded_by"`
	CreatedAt  time.Time `json:"created_at"`
}
