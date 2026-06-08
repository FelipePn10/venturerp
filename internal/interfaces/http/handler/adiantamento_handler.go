package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// AdiantamentoHandler exposes the advance-payment (adiantamentos) endpoints.
type AdiantamentoHandler struct {
	createUC  *financial_uc.CreateAdiantamentoUseCase
	listUC    *financial_uc.ListAdiantamentosUseCase
	getUC     *financial_uc.GetAdiantamentoUseCase
	aplicarUC *financial_uc.AplicarAdiantamentoUseCase
}

func NewAdiantamentoHandler(
	createUC *financial_uc.CreateAdiantamentoUseCase,
	listUC *financial_uc.ListAdiantamentosUseCase,
	getUC *financial_uc.GetAdiantamentoUseCase,
	aplicarUC *financial_uc.AplicarAdiantamentoUseCase,
) *AdiantamentoHandler {
	return &AdiantamentoHandler{createUC: createUC, listUC: listUC, getUC: getUC, aplicarUC: aplicarUC}
}

func (h *AdiantamentoHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateAdiantamentoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *AdiantamentoHandler) List(w http.ResponseWriter, r *http.Request) {
	var tipo *string
	if v := r.URL.Query().Get("tipo"); v != "" {
		tipo = &v
	}
	var parceiroID *int64
	if v := r.URL.Query().Get("parceiro_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			parceiroID = &id
		}
	}
	result, err := h.listUC.Execute(r.Context(), tipo, parceiroID)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *AdiantamentoHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.getUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *AdiantamentoHandler) Aplicar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.AplicarAdiantamentoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.aplicarUC.Execute(r.Context(), id, dto)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
