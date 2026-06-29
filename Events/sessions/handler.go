package sessions

import (
	"net/http"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
	"kvm_v2/internal/repository"
	authsvc "kvm_v2/services/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	sessions *repository.SessionRepository
	batches  *repository.BatchRepository
}

func NewHandler(sessions *repository.SessionRepository, batches *repository.BatchRepository) *Handler {
	return &Handler{sessions: sessions, batches: batches}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.With(authsvc.RequireAdmin()).Post("/session-templates", h.createTemplate)
	r.With(authsvc.RequireAdmin()).Get("/session-templates", h.listTemplates)
	r.With(authsvc.RequireAdmin()).Patch("/session-templates/{id}", h.updateTemplate)
	r.With(authsvc.RequireAdmin()).Post("/sessions/generate", h.generate)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/sessions/today", h.today)
	r.With(authsvc.RequireAdmin()).Patch("/sessions/{id}/cancel", h.cancel)
}

type templateRequest struct {
	BatchID   string `json:"batch_id"`
	TeacherID string `json:"teacher_id"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type templateUpdateRequest struct {
	DayOfWeek *int    `json:"day_of_week"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	TeacherID *string `json:"teacher_id"`
}

type generateRequest struct {
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	BatchID   *string `json:"batch_id"`
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.BatchID == "" || req.TeacherID == "" {
		httpx.WriteError(w, httpx.ErrInvalidInput)
		return
	}
	t, err := h.sessions.CreateTemplate(r.Context(), req.BatchID, req.TeacherID, req.DayOfWeek, req.StartTime, req.EndTime)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
}

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.sessions.ListTemplates(r.Context(), r.URL.Query().Get("batch_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, templates)
}

func (h *Handler) updateTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateUpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.sessions.UpdateTemplate(r.Context(), chi.URLParam(r, "id"), req.DayOfWeek, req.StartTime, req.EndTime, req.TeacherID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.StartDate == "" || req.EndDate == "" {
		httpx.WriteError(w, httpx.ErrInvalidInput)
		return
	}
	count, err := h.sessions.GenerateSessions(r.Context(), req.StartDate, req.EndDate, req.BatchID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"created": count})
}

func (h *Handler) today(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	teacherID := ""
	if claims.Role == models.RoleStaff {
		teacherID = claims.UserID
	} else if claims.Role == models.RoleSuperAdmin {
		teacherID = r.URL.Query().Get("teacher_id")
	}

	sessions, err := h.sessions.ListToday(r.Context(), teacherID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sessions)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	s, err := h.sessions.CancelSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}
