package attendance

import (
	"net/http"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
	"kvm_v2/internal/repository"
	authsvc "kvm_v2/services/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	attendance *repository.AttendanceRepository
	sessions   *repository.SessionRepository
	batches    *repository.BatchRepository
}

func NewHandler(attendance *repository.AttendanceRepository, sessions *repository.SessionRepository, batches *repository.BatchRepository) *Handler {
	return &Handler{attendance: attendance, sessions: sessions, batches: batches}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.With(authsvc.RequireStaffOrAdmin()).Post("/attendance/sessions/{sessionId}", h.bulkMark)
	r.Get("/attendance/sessions/{sessionId}", h.bySession)
	r.Get("/attendance/students/{id}", h.byStudent)
	r.With(authsvc.RequireAdmin()).Patch("/attendance/{id}", h.adminEdit)
}

type bulkMarkRequest struct {
	Records []repository.AttendanceMark `json:"records"`
	Lock    bool                        `json:"lock"`
}

type editRequest struct {
	Status string  `json:"status"`
	Reason *string `json:"reason"`
}

func (h *Handler) bulkMark(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	sessionID := chi.URLParam(r, "sessionId")

	session, err := h.sessions.GetSession(r.Context(), sessionID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if claims.Role == models.RoleStaff {
		ok, err := h.batches.IsTeacherOf(r.Context(), session.BatchID, claims.UserID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		if !ok {
			httpx.WriteError(w, httpx.Forbidden("you can only mark attendance for your own batches"))
			return
		}
	}

	var req bulkMarkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if len(req.Records) == 0 {
		httpx.WriteError(w, httpx.InvalidInput("at least one attendance record is required"))
		return
	}

	records, err := h.attendance.BulkMark(r.Context(), sessionID, claims.UserID, req.Records, req.Lock)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, records)
}

func (h *Handler) bySession(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	sessionID := chi.URLParam(r, "sessionId")

	if claims.Role == models.RoleStudent {
		httpx.WriteError(w, httpx.Forbidden("students cannot view session attendance"))
		return
	}

	if claims.Role == models.RoleStaff {
		session, err := h.sessions.GetSession(r.Context(), sessionID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		ok, err := h.batches.IsTeacherOf(r.Context(), session.BatchID, claims.UserID)
		if err != nil || !ok {
			httpx.WriteError(w, httpx.Forbidden("you can only view attendance for your own batches"))
			return
		}
	}

	p := httpx.ParsePagination(r)
	records, total, err := h.attendance.GetBySession(r.Context(), sessionID, r.URL.Query().Get("status"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.AttendanceRecord]{
		Data: records,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}

func (h *Handler) byStudent(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	studentID := chi.URLParam(r, "id")

	if claims.Role == models.RoleStudent && claims.UserID != studentID {
		httpx.WriteError(w, httpx.Forbidden("you can only view your own attendance"))
		return
	}

	p := httpx.ParsePagination(r)
	records, total, err := h.attendance.GetByStudent(r.Context(), studentID, r.URL.Query().Get("batch_id"), r.URL.Query().Get("status"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	pct, present, totalSessions, err := h.attendance.StudentPercentage(r.Context(), studentID, r.URL.Query().Get("batch_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"records": models.ListResponse[models.AttendanceRecord]{
			Data: records,
			Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
		},
		"percentage": pct,
		"present":    present,
		"total":      totalSessions,
	})
}

func (h *Handler) adminEdit(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())

	var req editRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Status == "" {
		httpx.WriteError(w, httpx.InvalidInput("status is required"))
		return
	}

	rec, err := h.attendance.AdminEdit(r.Context(), chi.URLParam(r, "id"), req.Status, claims.UserID, req.Reason)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rec)
}
