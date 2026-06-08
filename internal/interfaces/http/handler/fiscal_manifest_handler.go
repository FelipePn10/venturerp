package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
)

// FiscalManifestHandler exposes recipient manifestation and number inutilization.
type FiscalManifestHandler struct {
	manifestUC *fiscal_uc.ManifestarDestinatarioUseCase
	inutilUC   *fiscal_uc.InutilizarNumeracaoUseCase
}

func NewFiscalManifestHandler(
	manifestUC *fiscal_uc.ManifestarDestinatarioUseCase,
	inutilUC *fiscal_uc.InutilizarNumeracaoUseCase,
) *FiscalManifestHandler {
	return &FiscalManifestHandler{manifestUC: manifestUC, inutilUC: inutilUC}
}

func (h *FiscalManifestHandler) Manifestar(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.ManifestarDestinatarioDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.manifestUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalManifestHandler) Inutilizar(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.InutilizarNumeracaoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.inutilUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
