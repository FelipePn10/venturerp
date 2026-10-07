package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// DevolucaoCompraHandler expõe a devolução de compra ao fornecedor.
type DevolucaoCompraHandler struct {
	uc *fiscal_uc.DevolucaoCompraUseCase
}

func NewDevolucaoCompraHandler(uc *fiscal_uc.DevolucaoCompraUseCase) *DevolucaoCompraHandler {
	return &DevolucaoCompraHandler{uc: uc}
}

// Previa: GET /api/fiscal/entries/{code}/devolucao
func (h *DevolucaoCompraHandler) Previa(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "code")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	out, err := h.uc.Previa(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// Criar: POST /api/fiscal/devolucoes
func (h *DevolucaoCompraHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.DevolucaoCompraDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Criar(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

// Reprocessar: POST /api/fiscal/devolucoes/{id}/efetivar
func (h *DevolucaoCompraHandler) Reprocessar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	out, err := h.uc.Reprocessar(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
