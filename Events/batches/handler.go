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
	r.With(authsvc.RequireStaffOrAdmin()).Get("/batches/mine", h.mine)
	r.With(authsvc.RequireStaffOrAdmin()).Get("/batches/{id}/students", h.students)
	r.With(authsvc.RequireAdmin()).Get("/batches", h.list)
	r.With(authsvc.RequireAdmin()).Patch("/batches/{id}", h.update)
}

type createRequest struct {
	OfferingID string  `json:"offering_id"`
	Name       string  `json:"name"`
	TeacherID  *string `json:"teacher_id"`
	Capacity   *int    `json:"capacity"`
}

type updateRequest struct {
	Name      string  `json:"name"`
	TeacherID *string `json:"teacher_id"`
	Capacity  *int    `json:"capacity"`
	Status    *string `json:"status"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.OfferingID == "" || req.Name == "" {
		httpx.WriteError(w, httpx.ErrInvalidInput)
		return
	}
	b, err := h.batches.Create(r.Context(), req.OfferingID, req.Name, req.TeacherID, req.Capacity)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, b)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePagination(r)
	batches, total, err := h.batches.List(r.Context(), r.URL.Query().Get("offering_id"), "", r.URL.Query().Get("status"), p.Offset, p.Limit)
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
	b, err := h.batches.Update(r.Context(), chi.URLParam(r, "id"), req.Name, req.TeacherID, req.Capacity, req.Status)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, b)
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
			httpx.WriteError(w, httpx.ErrForbidden)
			return
		}
	}

	students, err := h.batches.ListStudents(r.Context(), batchID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, students)
}
