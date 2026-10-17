package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/estoque_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// EstoqueHandler expõe as operações de estoque além do lançamento simples.
type EstoqueHandler struct {
	UC *estoque_uc.UseCase
}

func inteiroDaRota(w http.ResponseWriter, r *http.Request, nome, rotulo string) (int64, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, nome)), 10, 64)
	if err != nil || v <= 0 {
		security.RespondError(w, http.StatusBadRequest, rotulo+" inválido")
		return 0, false
	}
	return v, true
}

func inteiroQuery(r *http.Request, chave string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get(chave)))
	return v
}

func (h *EstoqueHandler) responder(w http.ResponseWriter, status int, v any, err error) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, status, v)
}

// POST /api/stock/transfers
func (h *EstoqueHandler) Transferir(w http.ResponseWriter, r *http.Request) {
	var dto request.TransferenciaAlmoxarifadoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.Transferir(r.Context(), dto)
	h.responder(w, http.StatusCreated, out, err)
}

// POST /api/stock/movements/{id}/reverse
func (h *EstoqueHandler) Estornar(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "movimento")
	if !ok {
		return
	}
	var dto request.EstornoMovimentoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.Estornar(r.Context(), id, dto)
	h.responder(w, http.StatusCreated, out, err)
}

// GET /api/stock/kardex/{itemCode}?warehouse_id=&from=&to=
func (h *EstoqueHandler) Kardex(w http.ResponseWriter, r *http.Request) {
	item, ok := inteiroDaRota(w, r, "itemCode", "código do item")
	if !ok {
		return
	}
	q := r.URL.Query()
	out, err := h.UC.Kardex(r.Context(), item, inteiroDaQuery(q, "warehouse_id"), q.Get("from"), q.Get("to"))
	h.responder(w, http.StatusOK, out, err)
}

// GET /api/stock/position?date=&warehouse_id=
func (h *EstoqueHandler) Posicao(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := h.UC.Posicao(r.Context(), q.Get("date"), inteiroDaQuery(q, "warehouse_id"))
	h.responder(w, http.StatusOK, out, err)
}

// GET /api/stock/balances/atp/{itemCode}
func (h *EstoqueHandler) ATP(w http.ResponseWriter, r *http.Request) {
	item, ok := inteiroDaRota(w, r, "itemCode", "código do item")
	if !ok {
		return
	}
	out, err := h.UC.ATP(r.Context(), item, r.URL.Query().Get("mask"))
	h.responder(w, http.StatusOK, out, err)
}

// GET /api/stock/replenishment
func (h *EstoqueHandler) Reposicao(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Reposicao(r.Context())
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/stock/replenishment/purchase-requisition
func (h *EstoqueHandler) ReposicaoParaRequisicao(w http.ResponseWriter, r *http.Request) {
	var dto request.ReposicaoParaRequisicaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.GerarRequisicaoDeCompra(r.Context(), dto)
	h.responder(w, http.StatusCreated, out, err)
}

// GET /api/stock/analysis?months=&idle_days=
func (h *EstoqueHandler) Analise(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Analise(r.Context(), inteiroQuery(r, "months"), inteiroQuery(r, "idle_days"))
	h.responder(w, http.StatusOK, out, err)
}

// GET /api/stock/lots/expiring?days=
func (h *EstoqueHandler) LotesAVencer(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.LotesAVencer(r.Context(), inteiroQuery(r, "days"))
	h.responder(w, http.StatusOK, out, err)
}

// /api/stock/material-requisitions

func (h *EstoqueHandler) ListarRequisicoes(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.ListarRequisicoes(r.Context(), r.URL.Query().Get("status"))
	h.responder(w, http.StatusOK, out, err)
}

func (h *EstoqueHandler) CriarRequisicao(w http.ResponseWriter, r *http.Request) {
	var dto request.RequisicaoMaterialDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.CriarRequisicao(r.Context(), dto)
	h.responder(w, http.StatusCreated, out, err)
}

func (h *EstoqueHandler) ObterRequisicao(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "requisição")
	if !ok {
		return
	}
	out, err := h.UC.ObterRequisicao(r.Context(), id)
	h.responder(w, http.StatusOK, out, err)
}

func (h *EstoqueHandler) AtenderRequisicao(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "requisição")
	if !ok {
		return
	}
	var dto request.AtendimentoRequisicaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.AtenderRequisicao(r.Context(), id, dto)
	h.responder(w, http.StatusOK, out, err)
}

func (h *EstoqueHandler) CancelarRequisicao(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "requisição")
	if !ok {
		return
	}
	if err := h.UC.CancelarRequisicao(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
