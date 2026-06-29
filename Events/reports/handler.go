package reports

import (
	"net/http"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
	"kvm_v2/internal/repository"
	authsvc "kvm_v2/services/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	reports *repository.ReportRepository
	batches *repository.BatchRepository
}

func NewHandler(reports *repository.ReportRepository, batches *repository.BatchRepository) *Handler {
	return &Handler{reports: reports, batches: batches}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/reports/students/{id}/attendance", h.studentAttendance)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/reports/batches/{id}", h.batch)
	r.With(authsvc.RequireAdmin()).Get("/reports/subjects/{id}", h.subject)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/reports/teachers/{id}", h.teacher)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/reports/daily", h.daily)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/reports/monthly", h.monthly)
	r.With(authsvc.RequireAdmin()).Get("/reports/enrollments", h.enrollments)
	r.With(authsvc.RequireAdmin()).Get("/reports/fees", h.fees)
}

func (h *Handler) studentAttendance(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	studentID := chi.URLParam(r, "id")

	if claims.Role == models.RoleStudent && claims.UserID != studentID {
		httpx.WriteError(w, httpx.ErrForbidden)
		return
	}
	if claims.Role != models.RoleSuperAdmin && claims.Role != models.RoleStaff && claims.Role != models.RoleStudent {
		httpx.WriteError(w, httpx.ErrForbidden)
		return
	}

	reports, err := h.reports.StudentAttendance(r.Context(), studentID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reports)
}

func (h *Handler) batch(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	batchID := chi.URLParam(r, "id")

	if claims.Role == models.RoleStaff {
		ok, err := h.batches.IsTeacherOf(r.Context(), batchID, claims.UserID)
		if err != nil || !ok {
			httpx.WriteError(w, httpx.ErrForbidden)
			return
		}
	}

	reports, err := h.reports.BatchReport(r.Context(), batchID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reports)
}

func (h *Handler) subject(w http.ResponseWriter, r *http.Request) {
	reports, err := h.reports.SubjectReport(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reports)
}

func (h *Handler) teacher(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	teacherID := chi.URLParam(r, "id")

	if claims.Role == models.RoleStaff && claims.UserID != teacherID {
		httpx.WriteError(w, httpx.ErrForbidden)
		return
	}

	reports, err := h.reports.TeacherReport(r.Context(), teacherID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reports)
}

func (h *Handler) daily(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	date := r.URL.Query().Get("date")
	if date == "" {
		httpx.WriteError(w, httpx.ErrInvalidInput)
		return
	}

	teacherID := ""
	if claims.Role == models.RoleStaff {
		teacherID = claims.UserID
	}

	reports, err := h.reports.DailyReport(r.Context(), date, teacherID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reports)
}

func (h *Handler) monthly(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	month := r.URL.Query().Get("month")
	if month == "" {
		httpx.WriteError(w, httpx.ErrInvalidInput)
		return
	}

	teacherID := ""
	if claims.Role == models.RoleStaff {
		teacherID = claims.UserID
	}

	reports, err := h.reports.MonthlyReport(r.Context(), month, teacherID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reports)
}

func (h *Handler) enrollments(w http.ResponseWriter, r *http.Request) {
	reports, err := h.reports.EnrollmentReport(r.Context(), r.URL.Query().Get("year_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reports)
}

func (h *Handler) fees(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.reports.FeeSummary(r.Context(), r.URL.Query().Get("year_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summaries)
}
