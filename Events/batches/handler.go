package batches

import (
	"net/http"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
	"kvm_v2/internal/repository"
	authsvc "kvm_v2/services/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	batches *repository.BatchRepository
}

func NewHandler(batches *repository.BatchRepository) *Handler {
	return &Handler{batches: batches}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.With(authsvc.RequireAdmin()).Post("/batches", h.create)
	r.With(authsvc.RequireAdmin()).Get("/batches/{id}", h.get)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/batches/mine", h.mine)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/batches/{id}/students", h.students)
	r.With(authsvc.RequireAdmin()).Get("/batches", h.list)
	r.With(authsvc.RequireAdmin()).Patch("/batches/{id}", h.update)
	r.With(authsvc.RequireAdmin()).Delete("/batches/{id}", h.delete)
}

type createRequest struct {
	OfferingID string  `json:"offering_id"`
	Name       string  `json:"name"`
	TeacherID  *string `json:"teacher_id"`
}

type updateRequest struct {
	Name      string  `json:"name"`
	TeacherID *string `json:"teacher_id"`
	Status    *string `json:"status"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.OfferingID == "" || req.Name == "" {
		httpx.WriteError(w, httpx.InvalidInput("offering_id and batch name are required"))
		return
	}
	b, err := h.batches.Create(r.Context(), req.OfferingID, req.Name, req.TeacherID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, b)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	b, err := h.batches.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, b)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePagination(r)
	batches, total, err := h.batches.List(r.Context(), r.URL.Query().Get("offering_id"), r.URL.Query().Get("teacher_id"), r.URL.Query().Get("status"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.Batch]{
		Data: batches,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}

func (h *Handler) mine(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	teacherID := claims.UserID
	if claims.Role == models.RoleSuperAdmin {
		teacherID = r.URL.Query().Get("teacher_id")
	}
	p := httpx.ParsePagination(r)
	batches, total, err := h.batches.List(r.Context(), "", teacherID, models.BatchStatusActive, p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.Batch]{
		Data: batches,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.batches.Update(r.Context(), chi.URLParam(r, "id"), req.Name, req.TeacherID, req.Status)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, b)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.batches.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) students(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	batchID := chi.URLParam(r, "id")

	if claims.Role == models.RoleStaff {
		ok, err := h.batches.IsTeacherOf(r.Context(), batchID, claims.UserID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		if !ok {
			httpx.WriteError(w, httpx.Forbidden("you can only view students in your own batches"))
			return
		}
	}

	p := httpx.ParsePagination(r)
	students, total, err := h.batches.ListStudents(r.Context(), batchID, p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.User]{
		Data: students,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}
