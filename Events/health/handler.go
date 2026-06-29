package health

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.check)
}

func (h *Handler) check(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := "ok"

	if err := h.db.PingContext(r.Context()); err != nil {
		dbStatus = "down"
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	if dbStatus != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   status,
		"database": dbStatus,
	})
}
