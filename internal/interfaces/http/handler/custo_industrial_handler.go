package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/custo_industrial_uc"
)

// CustoIndustrialHandler: variação padrão × real, margem do pedido e roll-up em lote.
type CustoIndustrialHandler struct{ UC *custo_industrial_uc.UseCase }

// Variacoes: GET /api/standard-cost/variacoes?de=AAAA-MM-DD&ate=AAAA-MM-DD (padrão: mês corrente).
func (h *CustoIndustrialHandler) Variacoes(w http.ResponseWriter, r *http.Request) {
	hoje := time.Now()
	de := time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, hoje.Location())
	ate := hoje
	for _, p := range []struct {
		nome string
		dst  *time.Time
	}{{"de", &de}, {"ate", &ate}} {
		if v := r.URL.Query().Get(p.nome); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				jsonError(w, http.StatusBadRequest, "data inválida em "+p.nome+" (use AAAA-MM-DD)")
				return
			}
			*p.dst = t
		}
	}
	out, err := h.UC.VariacoesDoPeriodo(r.Context(), de, ate)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// MargemDoPedido: GET /api/standard-cost/margem/pedido/{code}.
func (h *CustoIndustrialHandler) MargemDoPedido(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "pedido inválido")
		return
	}
	out, err := h.UC.Margem(r.Context(), code)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// ApurarTodos: POST /api/standard-cost/rollup/todos — custo-padrão de todos os itens.
func (h *CustoIndustrialHandler) ApurarTodos(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.ApurarTodos(r.Context())
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}
