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
	r.With(authsvc.RequireAdmin()).Get("/session-templates/{id}", h.getTemplate)
	r.With(authsvc.RequireAdmin()).Patch("/session-templates/{id}", h.updateTemplate)
	r.With(authsvc.RequireAdmin()).Delete("/session-templates/{id}", h.deleteTemplate)
	r.With(authsvc.RequireAdmin()).Post("/sessions/generate", h.generate)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/sessions/today", h.today)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/sessions/{id}", h.getSession)
	r.With(authsvc.RequireAdmin()).Patch("/sessions/{id}/cancel", h.cancel)
}

type templateRequest struct {
	BatchID   string `json:"batch_id"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type templateUpdateRequest struct {
	DayOfWeek *int   `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type generateRequest struct {
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	BatchID   *string `json:"batch_id"`
}

func (h *Handler) batchTeacherID(w http.ResponseWriter, r *http.Request, batchID string) (string, bool) {
	batch, err := h.batches.Get(r.Context(), batchID)
	if err != nil {
		httpx.WriteError(w, err)
		return "", false
	}
	if batch.TeacherID == nil || *batch.TeacherID == "" {
		httpx.WriteError(w, httpx.InvalidInput("batch has no teacher assigned"))
		return "", false
	}
	return *batch.TeacherID, true
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.BatchID == "" {
		httpx.WriteError(w, httpx.InvalidInput("batch_id is required"))
		return
	}
	teacherID, ok := h.batchTeacherID(w, r, req.BatchID)
	if !ok {
		return
	}
	t, err := h.sessions.CreateTemplate(r.Context(), req.BatchID, teacherID, req.DayOfWeek, req.StartTime, req.EndTime)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
}

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePagination(r)
	q := r.URL.Query()
	templates, total, err := h.sessions.ListTemplates(r.Context(), q.Get("batch_id"), q.Get("teacher_id"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.SessionTemplate]{
		Data: templates,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}

func (h *Handler) getTemplate(w http.ResponseWriter, r *http.Request) {
	t, err := h.sessions.GetTemplate(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) updateTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateUpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	existing, err := h.sessions.GetTemplate(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	teacherID, ok := h.batchTeacherID(w, r, existing.BatchID)
	if !ok {
		return
	}
	t, err := h.sessions.UpdateTemplate(r.Context(), existing.ID, req.DayOfWeek, req.StartTime, req.EndTime, teacherID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.DeleteTemplate(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.StartDate == "" || req.EndDate == "" {
		httpx.WriteError(w, httpx.InvalidInput("start_date and end_date are required"))
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

	p := httpx.ParsePagination(r)
	sessions, total, err := h.sessions.ListToday(r.Context(), teacherID, p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.Session]{
		Data: sessions,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	s, err := h.sessions.GetSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	s, err := h.sessions.CancelSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}
