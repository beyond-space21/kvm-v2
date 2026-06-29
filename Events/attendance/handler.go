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
			httpx.WriteError(w, httpx.ErrForbidden)
			return
		}
	}

	var req bulkMarkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || len(req.Records) == 0 {
		httpx.WriteError(w, httpx.ErrInvalidInput)
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
		httpx.WriteError(w, httpx.ErrForbidden)
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
			httpx.WriteError(w, httpx.ErrForbidden)
			return
		}
	}

	records, err := h.attendance.GetBySession(r.Context(), sessionID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, records)
}

func (h *Handler) byStudent(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	studentID := chi.URLParam(r, "id")

	if claims.Role == models.RoleStudent && claims.UserID != studentID {
		httpx.WriteError(w, httpx.ErrForbidden)
		return
	}

	records, err := h.attendance.GetByStudent(r.Context(), studentID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	pct, present, total, err := h.attendance.StudentPercentage(r.Context(), studentID, r.URL.Query().Get("batch_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"records":    records,
		"percentage": pct,
		"present":    present,
		"total":      total,
	})
}

func (h *Handler) adminEdit(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())

	var req editRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.Status == "" {
		httpx.WriteError(w, httpx.ErrInvalidInput)
		return
	}

	rec, err := h.attendance.AdminEdit(r.Context(), chi.URLParam(r, "id"), req.Status, claims.UserID, req.Reason)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rec)
}
