package enrollments

import (
	"net/http"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
	"kvm_v2/internal/repository"
	authsvc "kvm_v2/services/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	enrollments *repository.EnrollmentRepository
}

func NewHandler(enrollments *repository.EnrollmentRepository) *Handler {
	return &Handler{enrollments: enrollments}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.With(authsvc.RequireAdmin()).Post("/enrollments", h.create)
	r.With(authsvc.RequireAdmin()).Get("/enrollments/{id}", h.get)
	r.With(authsvc.RequireAdmin()).Patch("/enrollments/{id}/transfer", h.transfer)
	r.With(authsvc.RequireAdmin()).Delete("/enrollments/{id}", h.remove)
	r.Get("/enrollments", h.list)
	r.With(authsvc.RequireAdmin()).Get("/students/{id}/enrollments/history", h.history)
}

type createRequest struct {
	StudentID      string `json:"student_id"`
	AcademicYearID string `json:"academic_year_id"`
	OfferingID     string `json:"offering_id"`
	BatchID        string `json:"batch_id"`
}

type transferRequest struct {
	BatchID string `json:"batch_id"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.StudentID == "" || req.AcademicYearID == "" || req.OfferingID == "" || req.BatchID == "" {
		httpx.WriteError(w, httpx.InvalidInput("student_id, academic_year_id, offering_id, and batch_id are required"))
		return
	}
	e, err := h.enrollments.Create(r.Context(), req.StudentID, req.AcademicYearID, req.OfferingID, req.BatchID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, e)
}

func (h *Handler) transfer(w http.ResponseWriter, r *http.Request) {
	var req transferRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.BatchID == "" {
		httpx.WriteError(w, httpx.InvalidInput("batch_id is required"))
		return
	}
	e, err := h.enrollments.Transfer(r.Context(), chi.URLParam(r, "id"), req.BatchID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, e)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	e, err := h.enrollments.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, e)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.enrollments.Remove(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())
	studentID := r.URL.Query().Get("student_id")
	status := r.URL.Query().Get("status")

	if claims.Role == models.RoleStudent {
		studentID = claims.UserID
		status = models.EnrollmentActive
	}

	p := httpx.ParsePagination(r)
	q := r.URL.Query()
	enrollments, total, err := h.enrollments.List(r.Context(), studentID, q.Get("year_id"), q.Get("offering_id"), q.Get("batch_id"), status, p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.Enrollment]{
		Data: enrollments,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePagination(r)
	q := r.URL.Query()
	history, total, err := h.enrollments.History(r.Context(), chi.URLParam(r, "id"), q.Get("year_id"), q.Get("offering_id"), q.Get("status"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.Enrollment]{
		Data: history,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}
