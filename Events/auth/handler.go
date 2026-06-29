package auth

import (
	"context"
	"net/http"
	"time"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
	"kvm_v2/internal/repository"
	authsvc "kvm_v2/services/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	users *repository.UserRepository
	auth  *authsvc.Service
	ttl   time.Duration
}

func NewHandler(users *repository.UserRepository, auth *authsvc.Service) *Handler {
	return &Handler{users: users, auth: auth, ttl: 24 * time.Hour}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/login", h.login)
	r.Post("/auth/bootstrap", h.bootstrap)

	r.Group(func(r chi.Router) {
		r.Use(h.auth.Middleware)
		r.Get("/auth/me", h.me)
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  models.User  `json:"user"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Email == "" || req.Password == "" {
		httpx.WriteError(w, httpx.InvalidInput("email and password are required"))
		return
	}

	user, hash, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		httpx.WriteError(w, httpx.Unauthorized("invalid email or password"))
		return
	}
	if user.Status != models.StatusActive {
		httpx.WriteError(w, httpx.Forbidden("account is inactive"))
		return
	}
	if !authsvc.CheckPassword(hash, req.Password) {
		httpx.WriteError(w, httpx.Unauthorized("invalid email or password"))
		return
	}

	token, err := h.auth.SignToken(user.ID, user.Role, h.ttl)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, loginResponse{Token: token, User: *user})
}

type bootstrapRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (h *Handler) bootstrap(w http.ResponseWriter, r *http.Request) {
	count, err := h.users.CountByRole(r.Context(), models.RoleSuperAdmin)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if count > 0 {
		httpx.WriteError(w, httpx.Forbidden("a super admin already exists"))
		return
	}

	var req bootstrapRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		httpx.WriteError(w, httpx.InvalidInput("email, password, and name are required"))
		return
	}

	hash, err := authsvc.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	user, err := h.users.Create(r.Context(), req.Email, hash, req.Name, models.RoleSuperAdmin, nil, nil)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	token, err := h.auth.SignToken(user.ID, user.Role, h.ttl)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, loginResponse{Token: token, User: *user})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := authsvc.ClaimsFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.Unauthorized("authentication required"))
		return
	}

	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func BootstrapAdmin(ctx context.Context, users *repository.UserRepository, email, password string) error {
	count, err := users.CountByRole(ctx, models.RoleSuperAdmin)
	if err != nil || count > 0 {
		return err
	}
	hash, err := authsvc.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = users.Create(ctx, email, hash, "Super Admin", models.RoleSuperAdmin, nil, nil)
	return err
}
