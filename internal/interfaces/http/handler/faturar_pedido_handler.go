package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// FaturarPedidoHandler emite a NF-e de saída a partir do pedido de venda.
type FaturarPedidoHandler struct {
	uc *fiscal_uc.FaturarPedidoUseCase
}

func NewFaturarPedidoHandler(uc *fiscal_uc.FaturarPedidoUseCase) *FaturarPedidoHandler {
	return &FaturarPedidoHandler{uc: uc}
}

func (h *FaturarPedidoHandler) Previa(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "pedido inválido")
		return
	}
	out, err := h.uc.Previa(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FaturarPedidoHandler) Faturar(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.FaturarPedidoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	out, err := h.uc.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}
