package handler

import (
	"encoding/json"
	"net/http"

	fiscaluc "github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
)

type ImportNFePurchaseHandler struct {
	uc *fiscaluc.ImportNFePurchaseUseCase
}

func NewImportNFePurchaseHandler(uc *fiscaluc.ImportNFePurchaseUseCase) *ImportNFePurchaseHandler {
	return &ImportNFePurchaseHandler{uc: uc}
}

func (h *ImportNFePurchaseHandler) Import(w http.ResponseWriter, r *http.Request) {
	var dto fiscaluc.ImportNFePurchaseDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}
