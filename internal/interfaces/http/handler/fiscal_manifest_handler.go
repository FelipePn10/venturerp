package handler

import (
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
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
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.manifestUC.Execute(r.Context(), dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalManifestHandler) Inutilizar(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.InutilizarNumeracaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.inutilUC.Execute(r.Context(), dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
