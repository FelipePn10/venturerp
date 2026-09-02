package handler

import (
	"encoding/json"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/commercial_commission_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/commercial_commission/entity"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"strings"
)

type CommercialCommissionHandler struct {
	uc *commercial_commission_uc.UseCase
}

func NewCommercialCommissionHandler(uc *commercial_commission_uc.UseCase) *CommercialCommissionHandler {
	return &CommercialCommissionHandler{uc: uc}
}
func (h *CommercialCommissionHandler) UpsertSettings(w http.ResponseWriter, r *http.Request) {
	var d commercial_commission_uc.SettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		security.RespondError(w, 400, "corpo da requisição inválido")
		return
	}
	v, err := h.uc.UpsertSettings(r.Context(), d)
	respondCommission(w, v, err, 200)
}
func (h *CommercialCommissionHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	v, err := h.uc.GetSettings(r.Context())
	respondCommission(w, v, err, 200)
}
func (h *CommercialCommissionHandler) List(w http.ResponseWriter, r *http.Request) {
	f := entity.Filter{}
	if raw := r.URL.Query().Get("representative_code"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			security.RespondError(w, 422, "representative_code inválido")
			return
		}
		f.RepresentativeCode = &v
	}
	if raw := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status"))); raw != "" {
		f.Status = &raw
	}
	var err error
	f.From, err = commercial_commission_uc.ParseDate(r.URL.Query().Get("from"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	f.To, err = commercial_commission_uc.ParseDate(r.URL.Query().Get("to"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	f.Limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	f.Offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	v, err := h.uc.List(r.Context(), f)
	respondCommission(w, v, err, 200)
}
func (h *CommercialCommissionHandler) Transition(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, 400, "código da comissão inválido")
		return
	}
	var d commercial_commission_uc.TransitionDTO
	if err = json.NewDecoder(r.Body).Decode(&d); err != nil {
		security.RespondError(w, 400, "corpo da requisição inválido")
		return
	}
	v, err := h.uc.Transition(r.Context(), code, chi.URLParam(r, "action"), r.Header.Get("Idempotency-Key"), d)
	respondCommission(w, v, err, 200)
}
func respondCommission(w http.ResponseWriter, v any, err error, status int) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, status, v)
}
