package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

func (h *PlannedOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreatePlannedOrderDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *PlannedOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *PlannedOrderHandler) Firm(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDaRota(w, r, "code")
	if !ok {
		return
	}
	result, err := h.firmUC.Execute(r.Context(), request.FirmOrderDTO{OrderCode: code})
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PlannedOrderHandler) Transition(w http.ResponseWriter, r *http.Request) {
	var dto request.TransitionPlannedOrderDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.firmUC.ExecuteTransition(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
