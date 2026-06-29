package payments

import (
	"net/http"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
	"kvm_v2/internal/repository"
	authsvc "kvm_v2/services/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	payments *repository.PaymentRepository
}

func NewHandler(payments *repository.PaymentRepository) *Handler {
	return &Handler{payments: payments}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.With(authsvc.RequireAdmin()).Get("/enrollments/{id}/invoice", h.getInvoice)
	r.With(authsvc.RequireAdmin()).Get("/enrollments/{id}/payments", h.listPayments)
	r.With(authsvc.RequireAdmin()).Post("/enrollments/{id}/payments", h.recordPayment)
}

type recordPaymentRequest struct {
	Amount    float64 `json:"amount"`
	Method    *string `json:"method"`
	Reference *string `json:"reference"`
}

func (h *Handler) getInvoice(w http.ResponseWriter, r *http.Request) {
	invoice, err := h.payments.GetInvoiceByEnrollment(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, invoice)
}

func (h *Handler) listPayments(w http.ResponseWriter, r *http.Request) {
	p := httpx.ParsePagination(r)
	payments, total, err := h.payments.ListPayments(r.Context(), chi.URLParam(r, "id"), p.Offset, p.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ListResponse[models.FeePayment]{
		Data: payments,
		Pagination: models.Pagination{Page: p.Page, Limit: p.Limit, Total: total},
	})
}

func (h *Handler) recordPayment(w http.ResponseWriter, r *http.Request) {
	claims, _ := authsvc.ClaimsFromContext(r.Context())

	var req recordPaymentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Amount <= 0 {
		httpx.WriteError(w, httpx.InvalidInput("amount must be greater than zero"))
		return
	}

	payment, err := h.payments.RecordPayment(r.Context(), chi.URLParam(r, "id"), req.Amount, req.Method, req.Reference, claims.UserID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, payment)
}
