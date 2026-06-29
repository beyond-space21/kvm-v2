package users

import (
	"net/http"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
	"kvm_v2/internal/repository"
	authsvc "kvm_v2/services/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	users *repository.UserRepository
	auth  *authsvc.Service
}

func NewHandler(users *repository.UserRepository, auth *authsvc.Service) *Handler {
	return &Handler{users: users, auth: auth}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/staff", func(r chi.Router) {
		r.With(authsvc.RequireAdmin()).Post("/", h.createStaff)
		r.With(authsvc.RequireAdmin()).Get("/", h.listStaff)
		r.With(authsvc.RequireAdmin()).Get("/{id}", h.getStaff)
		r.With(authsvc.RequireAdmin()).Patch("/{id}", h.updateStaff)
		r.With(authsvc.RequireAdmin()).Delete("/{id}", h.deleteStaff)
	})

	r.Route("/students", func(r chi.Router) {
		r.With(authsvc.RequireAdmin()).Post("/", h.createStudent)
		r.With(authsvc.RequireAdmin()).Get("/", h.listStudents)
		r.With(authsvc.RequireAdmin()).Get("/{id}", h.getStudent)
		r.With(authsvc.RequireAdmin()).Patch("/{id}", h.updateStudent)
		r.With(authsvc.RequireAdmin()).Delete("/{id}", h.deleteStudent)
	})
}

type createUserRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Name     string  `json:"name"`
	Phone    *string `json:"phone"`
	ClassID  *string `json:"class_id"`
}

type updateUserRequest struct {
	Name    string  `json:"name"`
	Phone   *string `json:"phone"`
	Status  *string `json:"status"`
	ClassID *string `json:"class_id"`
}

func (h *Handler) createStaff(w http.ResponseWriter, r *http.Request) {
	h.createUser(w, r, models.RoleStaff, false)
}

func (h *Handler) createStudent(w http.ResponseWriter, r *http.Request) {
	h.createUser(w, r, models.RoleStudent, true)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request, role string, requireClass bool) {
	var req createUserRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		httpx.WriteError(w, httpx.InvalidInput("email, password, and name are required"))
		return
	}
	if requireClass && (req.ClassID == nil || *req.ClassID == "") {
		httpx.WriteError(w, httpx.InvalidInput("class_id is required for students"))
		return
	}

	hash, err := authsvc.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	user, err := h.users.Create(r.Context(), req.Email, hash, req.Name, role, req.Phone, req.ClassID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) getStaff(w http.ResponseWriter, r *http.Request) {
	h.getUser(w, r, models.RoleStaff)
}

func (h *Handler) getStudent(w http.ResponseWriter, r *http.Request) {
	h.getUser(w, r, models.RoleStudent)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request, expectedRole string) {
	user, err := h.users.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if user.Role != expectedRole {
		httpx.WriteError(w, httpx.NotFound(expectedRole+" not found"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) updateStaff(w http.ResponseWriter, r *http.Request) {
	h.updateUser(w, r, models.RoleStaff)
}

func (h *Handler) updateStudent(w http.ResponseWriter, r *http.Request) {
	h.updateUser(w, r, models.RoleStudent)
}

func (h *Handler) deleteStaff(w http.ResponseWriter, r *http.Request) {
	h.deleteUser(w, r, models.RoleStaff)
}

func (h *Handler) deleteStudent(w http.ResponseWriter, r *http.Request) {
	h.deleteUser(w, r, models.RoleStudent)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request, expectedRole string) {
	id := chi.URLParam(r, "id")
	existing, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if existing.Role != expectedRole {
		httpx.WriteError(w, httpx.NotFound(expectedRole+" not found"))
		return
	}

	var req updateUserRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	user, err := h.users.Update(r.Context(), id, req.Name, req.Phone, req.Status, req.ClassID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request, expectedRole string) {
	id := chi.URLParam(r, "id")
	existing, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if existing.Role != expectedRole {
		httpx.WriteError(w, httpx.NotFound(expectedRole+" not found"))
		return
	}
	if err := h.users.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listStaff(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePagination(r)
	q := r.URL.Query()
	users, total, err := h.users.List(r.Context(), models.RoleStaff, q.Get("status"), "", q.Get("search"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.User]{
		Data: users,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}

func (h *Handler) listStudents(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePagination(r)
	q := r.URL.Query()
	users, total, err := h.users.List(r.Context(), models.RoleStudent, q.Get("status"), q.Get("class_id"), q.Get("search"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.User]{
		Data: users,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}
