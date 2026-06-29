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
		httpx.WriteError(w, httpx.Forbidden("you can only view your own attendance report"))
		return
	}
	if claims.Role != models.RoleSuperAdmin && claims.Role != models.RoleStaff && claims.Role != models.RoleStudent {
		httpx.WriteError(w, httpx.Forbidden("insufficient permissions for this action"))
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
			httpx.WriteError(w, httpx.Forbidden("you can only view reports for your own batches"))
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
		httpx.WriteError(w, httpx.Forbidden("you can only view your own teacher report"))
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
		httpx.WriteError(w, httpx.InvalidInput("date query parameter is required (YYYY-MM-DD)"))
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
		httpx.WriteError(w, httpx.InvalidInput("month query parameter is required (YYYY-MM)"))
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
	p := httpx.ParsePagination(r)
	q := r.URL.Query()
	reports, total, err := h.reports.EnrollmentReport(r.Context(), q.Get("year_id"), q.Get("class_id"), q.Get("batch_id"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[repository.EnrollmentReport]{
		Data: reports,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}

func (h *Handler) fees(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePagination(r)
	q := r.URL.Query()
	summaries, total, err := h.reports.FeeSummary(r.Context(), q.Get("year_id"), q.Get("class_id"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[repository.FeeSummary]{
		Data: summaries,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}
