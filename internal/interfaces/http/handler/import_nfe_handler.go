package handler

import (
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
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
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.uc.Execute(r.Context(), dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}
