package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/ibpt_uc"
)

// IBPTHandler exposes IBPT/SCI import and lookup (Lei da Transparência).
type IBPTHandler struct {
	uc *ibpt_uc.IBPTUseCase
}

func NewIBPTHandler(uc *ibpt_uc.IBPTUseCase) *IBPTHandler {
	return &IBPTHandler{uc: uc}
}

type ibptImportRequest struct {
	UF  string `json:"uf"`
	CSV string `json:"csv"`
}

// Import loads an IBPT TabelaIBPTax CSV for a UF.
func (h *IBPTHandler) Import(w http.ResponseWriter, r *http.Request) {
	var req ibptImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	n, err := h.uc.ImportFromCSV(r.Context(), req.UF, req.CSV)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"imported": n})
}

// Lookup returns the approximate tax burden for an NCM in a UF.
// Query: ncm, uf.
func (h *IBPTHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	ncm := r.URL.Query().Get("ncm")
	uf := r.URL.Query().Get("uf")
	if ncm == "" || uf == "" {
		jsonError(w, http.StatusBadRequest, "ncm and uf are required")
		return
	}
	rate, err := h.uc.Lookup(r.Context(), ncm, uf)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, rate)
}
