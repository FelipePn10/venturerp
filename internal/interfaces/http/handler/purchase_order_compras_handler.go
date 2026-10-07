package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// PurchaseOrderComprasHandler atende o documento do pedido (PDF e envio), o
// acompanhamento de entregas, o histórico de preço, as notas que atenderam o
// pedido e a previsão de pagamentos.
type PurchaseOrderComprasHandler struct {
	Documento        *purchase_order_uc.DocumentoPedidoUseCase
	AcompanhamentoUC *purchase_order_uc.AcompanhamentoUseCase
	Historico        *purchase_order_uc.HistoricoPrecoUseCase
	NotasUC          *purchase_order_uc.NotasDoPedidoUseCase
	Previsao         *purchase_order_uc.PrevisaoPagamentosUseCase
}

func codigoPedido(w http.ResponseWriter, r *http.Request) (int64, bool) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código do pedido inválido")
		return 0, false
	}
	return code, true
}

// PDF: GET /api/purchase-order/{code}/pdf
func (h *PurchaseOrderComprasHandler) PDF(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoPedido(w, r)
	if !ok {
		return
	}
	b, nome, err := h.Documento.PDF(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, nome))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

// Destinatarios: GET /api/purchase-order/{code}/destinatarios
func (h *PurchaseOrderComprasHandler) Destinatarios(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoPedido(w, r)
	if !ok {
		return
	}
	out, err := h.Documento.Destinatarios(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// Envios: GET /api/purchase-order/{code}/envios
func (h *PurchaseOrderComprasHandler) Envios(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoPedido(w, r)
	if !ok {
		return
	}
	out, err := h.Documento.Envios(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// Enviar: POST /api/purchase-order/{code}/enviar
func (h *PurchaseOrderComprasHandler) Enviar(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoPedido(w, r)
	if !ok {
		return
	}
	var dto purchase_order_uc.EnviarDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "corpo inválido: "+err.Error())
		return
	}
	out, err := h.Documento.Enviar(r.Context(), code, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// Acompanhamento: GET /api/purchase-order/acompanhamento?situacao=&supplier_code=
func (h *PurchaseOrderComprasHandler) Acompanhamento(w http.ResponseWriter, r *http.Request) {
	var fornecedor *int64
	if v := r.URL.Query().Get("supplier_code"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "fornecedor inválido")
			return
		}
		fornecedor = &n
	}
	out, err := h.AcompanhamentoUC.Linhas(r.Context(), r.URL.Query().Get("situacao"), fornecedor, time.Now())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func codigoPedidoELinha(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	code, ok := codigoPedido(w, r)
	if !ok {
		return 0, 0, false
	}
	linha, err := strconv.ParseInt(chi.URLParam(r, "lineCode"), 10, 64)
	if err != nil || linha <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código da linha inválido")
		return 0, 0, false
	}
	return code, linha, true
}

// RegistrarFollowup: POST /api/purchase-order/{code}/items/{lineCode}/followups
func (h *PurchaseOrderComprasHandler) RegistrarFollowup(w http.ResponseWriter, r *http.Request) {
	code, linha, ok := codigoPedidoELinha(w, r)
	if !ok {
		return
	}
	var dto purchase_order_uc.FollowupDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "corpo inválido: "+err.Error())
		return
	}
	out, err := h.AcompanhamentoUC.RegistrarFollowup(r.Context(), code, linha, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

// Followups: GET /api/purchase-order/{code}/items/{lineCode}/followups
func (h *PurchaseOrderComprasHandler) Followups(w http.ResponseWriter, r *http.Request) {
	code, linha, ok := codigoPedidoELinha(w, r)
	if !ok {
		return
	}
	out, err := h.AcompanhamentoUC.Followups(r.Context(), code, linha)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// HistoricoPreco: GET /api/purchase-order/historico-preco?item_code=
// O middleware de compatibilidade troca o código comercial pela chave interna.
func (h *PurchaseOrderComprasHandler) HistoricoPreco(w http.ResponseWriter, r *http.Request) {
	item, err := strconv.ParseInt(r.URL.Query().Get("item_code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "informe o item")
		return
	}
	out, err := h.Historico.Execute(r.Context(), item, time.Now())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// Notas: GET /api/purchase-order/{code}/notas
func (h *PurchaseOrderComprasHandler) Notas(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoPedido(w, r)
	if !ok {
		return
	}
	out, err := h.NotasUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// PrevisaoDoPedido: GET /api/purchase-order/{code}/previsao-pagamentos
func (h *PurchaseOrderComprasHandler) PrevisaoDoPedido(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoPedido(w, r)
	if !ok {
		return
	}
	out, err := h.Previsao.DoPedido(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// PrevisaoGeral: GET /api/purchase-order/previsao-pagamentos?de=AAAA-MM-DD&ate=AAAA-MM-DD
func (h *PurchaseOrderComprasHandler) PrevisaoGeral(w http.ResponseWriter, r *http.Request) {
	de, err := time.Parse("2006-01-02", r.URL.Query().Get("de"))
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "data inicial inválida; use AAAA-MM-DD")
		return
	}
	ate, err := time.Parse("2006-01-02", r.URL.Query().Get("ate"))
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "data final inválida; use AAAA-MM-DD")
		return
	}
	out, err := h.Previsao.Geral(r.Context(), de, ate)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
