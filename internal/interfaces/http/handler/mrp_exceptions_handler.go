package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_uc"
)

type MRPExceptionsHandler struct {
	uc *mrp_uc.NotifyExceptionsUseCase
}

func NewMRPExceptionsHandler(uc *mrp_uc.NotifyExceptionsUseCase) *MRPExceptionsHandler {
	return &MRPExceptionsHandler{uc: uc}
}

func (h *MRPExceptionsHandler) Notify(w http.ResponseWriter, r *http.Request) {
	var dto mrp_uc.NotifyExceptionsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
