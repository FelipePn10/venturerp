package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// FiscalEntryEnterpriseHandler: cancelamento da nota de entrada, pedidos do
// item, parâmetros da contabilização automática e a caixa de entrada DF-e.
type FiscalEntryEnterpriseHandler struct {
	cancelar *fiscal_uc.CancelFiscalEntryUseCase
	pedidos  *fiscal_uc.PedidosDoItemUseCase
	contab   *fiscal_uc.AccountingParamsUseCase
	dfe      *fiscal_uc.DFeUseCase
}

func NewFiscalEntryEnterpriseHandler(c *fiscal_uc.CancelFiscalEntryUseCase, p *fiscal_uc.PedidosDoItemUseCase, a *fiscal_uc.AccountingParamsUseCase, d *fiscal_uc.DFeUseCase) *FiscalEntryEnterpriseHandler {
	return &FiscalEntryEnterpriseHandler{cancelar: c, pedidos: p, contab: a, dfe: d}
}

func idDaRota(r *http.Request, nome string) (int64, bool) {
	v, err := strconv.ParseInt(chi.URLParam(r, nome), 10, 64)
	return v, err == nil && v > 0
}

func (h *FiscalEntryEnterpriseHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "code")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var dto request.CancelFiscalEntryDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.cancelar.Execute(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FiscalEntryEnterpriseHandler) PedidosDoItem(w http.ResponseWriter, r *http.Request) {
	id, ok1 := idDaRota(r, "code")
	item, ok2 := idDaRota(r, "itemId")
	if !ok1 || !ok2 {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var itemCode *int64
	if v := strings.TrimSpace(r.URL.Query().Get("item_code")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "item inválido")
			return
		}
		itemCode = &n
	}
	out, err := h.pedidos.Execute(r.Context(), id, item, itemCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FiscalEntryEnterpriseHandler) GetParametrosContabeis(w http.ResponseWriter, r *http.Request) {
	out, err := h.contab.Get(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FiscalEntryEnterpriseHandler) SaveParametrosContabeis(w http.ResponseWriter, r *http.Request) {
	var p repository.AccountingPostingParams
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.contab.Save(r.Context(), p)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FiscalEntryEnterpriseHandler) VincularPlanoContabil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var body struct {
		AccountingAccountID *int64 `json:"accounting_account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.contab.VincularPlano(r.Context(), id, body.AccountingAccountID); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]any{"id": id, "accounting_account_id": body.AccountingAccountID})
}

func (h *FiscalEntryEnterpriseHandler) ListarRecebidas(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sim := func(k string) bool { return q.Get(k) == "1" || q.Get(k) == "true" }
	out, err := h.dfe.Listar(r.Context(), fiscal_uc.FiltroRecebidas{Pendentes: sim("pendentes"), Prazo: sim("prazo"), Busca: q.Get("q")})
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FiscalEntryEnterpriseHandler) SincronizarRecebidas(w http.ResponseWriter, r *http.Request) {
	out, err := h.dfe.Sincronizar(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FiscalEntryEnterpriseHandler) ManifestarRecebida(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Tipo          string `json:"tipo"`
		Justificativa string `json:"justificativa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.dfe.Manifestar(r.Context(), chi.URLParam(r, "chave"), body.Tipo, body.Justificativa); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"chave": chi.URLParam(r, "chave"), "manifestacao": body.Tipo})
}

func (h *FiscalEntryEnterpriseHandler) ImportarRecebida(w http.ResponseWriter, r *http.Request) {
	out, err := h.dfe.Importar(r.Context(), chi.URLParam(r, "chave"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

func (h *FiscalEntryEnterpriseHandler) StatusRecebidas(w http.ResponseWriter, r *http.Request) {
	out, err := h.dfe.Status(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FiscalEntryEnterpriseHandler) AutomaticoRecebidas(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ativo *bool `json:"ativo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Ativo == nil {
		security.RespondError(w, http.StatusBadRequest, "informe {\"ativo\": true|false}")
		return
	}
	if err := h.dfe.SetAutomatico(r.Context(), *body.Ativo); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]bool{"automatico": *body.Ativo})
}

func (h *FiscalEntryEnterpriseHandler) VincularContaBancariaContabil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var body struct {
		AccountingAccountID *int64 `json:"accounting_account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.contab.VincularContaBancaria(r.Context(), id, body.AccountingAccountID); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]any{"id": id, "accounting_account_id": body.AccountingAccountID})
}

// FornecedorDaNotaHandler cadastra o emitente da nota como fornecedor.
type FornecedorDaNotaHandler struct {
	uc *fiscal_uc.FornecedorDaNotaUseCase
}

func NewFornecedorDaNotaHandler(uc *fiscal_uc.FornecedorDaNotaUseCase) *FornecedorDaNotaHandler {
	return &FornecedorDaNotaHandler{uc: uc}
}

// Cadastrar — POST /api/fiscal/entries/{code}/fornecedor.
func (h *FornecedorDaNotaHandler) Cadastrar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "code")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var dto fiscal_uc.FornecedorDaNotaDTO
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
			return
		}
	}
	out, err := h.uc.Execute(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// ItemDaNotaHandler cadastra o item de uma linha da nota que não existe no cadastro.
type ItemDaNotaHandler struct {
	uc *fiscal_uc.ItemDaNotaUseCase
}

func NewItemDaNotaHandler(uc *fiscal_uc.ItemDaNotaUseCase) *ItemDaNotaHandler {
	return &ItemDaNotaHandler{uc: uc}
}

// Cadastrar — POST /api/fiscal/entries/{code}/itens/{itemId}/cadastrar-item.
func (h *ItemDaNotaHandler) Cadastrar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "code")
	itemID, ok2 := idDaRota(r, "itemId")
	if !ok || !ok2 {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var dto fiscal_uc.ItemDaNotaDTO
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
			return
		}
	}
	out, err := h.uc.Execute(r.Context(), id, itemID, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}
