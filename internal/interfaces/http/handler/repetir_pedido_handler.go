package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
)

// RepetirPedidoHandler: POST /api/sales-order/{code}/repetir — copia o pedido
// como rascunho com o preço de hoje.
type RepetirPedidoHandler struct {
	UC *sales_order_uc.RepetirPedidoUseCase
}

func (h *RepetirPedidoHandler) Repetir(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "pedido inválido")
		return
	}
	out, err := h.UC.Repetir(r.Context(), code)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, out)
}
