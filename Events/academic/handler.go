package academic

import (
	"net/http"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/repository"
	authsvc "kvm_v2/services/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	academic *repository.AcademicRepository
}

func NewHandler(academic *repository.AcademicRepository) *Handler {
	return &Handler{academic: academic}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	admin := authsvc.RequireAdmin()

	r.With(admin).Route("/academic-years", func(r chi.Router) {
		r.Post("/", h.createYear)
		r.Get("/", h.listYears)
		r.Get("/{id}", h.getYear)
		r.Patch("/{id}", h.updateYear)
		r.Delete("/{id}", h.deleteYear)
	})

	r.Route("/classes", func(r chi.Router) {
		r.Get("/", h.listClasses)
		r.Get("/{id}", h.getClass)
		r.Get("/{id}/offerings", h.listOfferingsByClass)
	})

	r.With(admin).Route("/subjects", func(r chi.Router) {
		r.Post("/", h.createSubject)
		r.Get("/", h.listSubjects)
		r.Get("/{id}", h.getSubject)
		r.Patch("/{id}", h.updateSubject)
		r.Delete("/{id}", h.deleteSubject)
	})

	r.With(admin).Route("/offerings", func(r chi.Router) {
		r.Post("/", h.createOffering)
		r.Get("/", h.listOfferings)
		r.Get("/{id}", h.getOffering)
		r.Patch("/{id}/fee", h.updateOfferingFee)
		r.Delete("/{id}", h.deleteOffering)
		r.Get("/{id}/fee-history", h.feeHistory)
	})
}

type yearRequest struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	IsActive  bool   `json:"is_active"`
}

func (h *Handler) createYear(w http.ResponseWriter, r *http.Request) {
	var req yearRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.Name == "" {
		httpx.WriteError(w, httpx.ErrInvalidInput)
		return
	}
	y, err := h.academic.CreateYear(r.Context(), req.Name, req.StartDate, req.EndDate, req.IsActive)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, y)
}

func (h *Handler) listYears(w http.ResponseWriter, r *http.Request) {
	years, err := h.academic.ListYears(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, years)
}

func (h *Handler) getYear(w http.ResponseWriter, r *http.Request) {
	y, err := h.academic.GetYear(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, y)
}

func (h *Handler) updateYear(w http.ResponseWriter, r *http.Request) {
	var req yearRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	var active *bool
	if r.URL.Query().Has("active") || req.IsActive {
		active = &req.IsActive
	}
	y, err := h.academic.UpdateYear(r.Context(), chi.URLParam(r, "id"), req.Name, req.StartDate, req.EndDate, active)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, y)
}

func (h *Handler) deleteYear(w http.ResponseWriter, r *http.Request) {
	if err := h.academic.DeleteYear(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listClasses(w http.ResponseWriter, r *http.Request) {
	classes, err := h.academic.ListClasses(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, classes)
}

func (h *Handler) getClass(w http.ResponseWriter, r *http.Request) {
	c, err := h.academic.GetClass(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) listOfferingsByClass(w http.ResponseWriter, r *http.Request) {
	offerings, err := h.academic.ListOfferings(r.Context(), r.URL.Query().Get("year_id"), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, offerings)
}

type subjectRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func (h *Handler) createSubject(w http.ResponseWriter, r *http.Request) {
	var req subjectRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.Name == "" || req.Code == "" {
		httpx.WriteError(w, httpx.ErrInvalidInput)
		return
	}
	s, err := h.academic.CreateSubject(r.Context(), req.Name, req.Code)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s)
}

func (h *Handler) listSubjects(w http.ResponseWriter, r *http.Request) {
	subjects, err := h.academic.ListSubjects(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, subjects)
}

func (h *Handler) getSubject(w http.ResponseWriter, r *http.Request) {
	s, err := h.academic.GetSubject(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) updateSubject(w http.ResponseWriter, r *http.Request) {
	var req subjectRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	s, err := h.academic.UpdateSubject(r.Context(), chi.URLParam(r, "id"), req.Name, req.Code)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) deleteSubject(w http.ResponseWriter, r *http.Request) {
	if err := h.academic.DeleteSubject(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type offeringRequest struct {
	AcademicYearID string  `json:"academic_year_id"`
	ClassID        string  `json:"class_id"`
	SubjectID      string  `json:"subject_id"`
	FeeAmount      float64 `json:"fee_amount"`
	FeeCurrency    string  `json:"fee_currency"`
}

func (h *Handler) createOffering(w http.ResponseWriter, r *http.Request) {
	var req offeringRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.AcademicYearID == "" || req.ClassID == "" || req.SubjectID == "" {
		httpx.WriteError(w, httpx.ErrInvalidInput)
		return
	}
	if req.FeeCurrency == "" {
		req.FeeCurrency = "INR"
	}
	o, err := h.academic.CreateOffering(r.Context(), req.AcademicYearID, req.ClassID, req.SubjectID, req.FeeAmount, req.FeeCurrency)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, o)
}

func (h *Handler) listOfferings(w http.ResponseWriter, r *http.Request) {
	offerings, err := h.academic.ListOfferings(r.Context(), r.URL.Query().Get("year_id"), r.URL.Query().Get("class_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, offerings)
}

func (h *Handler) getOffering(w http.ResponseWriter, r *http.Request) {
	o, err := h.academic.GetOffering(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

type feeUpdateRequest struct {
	FeeAmount     float64 `json:"fee_amount"`
	EffectiveFrom string  `json:"effective_from"`
}

func (h *Handler) updateOfferingFee(w http.ResponseWriter, r *http.Request) {
	var req feeUpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	o, err := h.academic.UpdateOfferingFee(r.Context(), chi.URLParam(r, "id"), req.FeeAmount, req.EffectiveFrom)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

func (h *Handler) deleteOffering(w http.ResponseWriter, r *http.Request) {
	if err := h.academic.DeleteOffering(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) feeHistory(w http.ResponseWriter, r *http.Request) {
	history, err := h.academic.ListFeeHistory(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, history)
}
