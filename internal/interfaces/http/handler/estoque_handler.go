package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/estoque_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// EstoqueHandler expõe as operações de estoque além do lançamento simples.
type EstoqueHandler struct {
	UC *estoque_uc.UseCase
}

func inteiroDaRota(w http.ResponseWriter, r *http.Request, nome, rotulo string) (int64, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, nome)), 10, 64)
	if err != nil || v <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código inválido: "+rotulo)
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
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, map[string]any{
		"saida": stock_uc.ToStockMovementResponse(out.Saida), "entrada": stock_uc.ToStockMovementResponse(out.Entrada),
	})
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
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, stock_uc.ToStockMovementResponses(out))
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

// GET /api/stock/closings
func (h *EstoqueHandler) Fechamentos(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Fechamentos(r.Context())
	h.responder(w, http.StatusOK, out, err)
}

type fechamentoDTO struct {
	Competence string `json:"competence"`
}

// POST /api/stock/closings/simulate
func (h *EstoqueHandler) SimularFechamento(w http.ResponseWriter, r *http.Request) {
	var dto fechamentoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.FecharMes(r.Context(), dto.Competence, true)
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/stock/closings
func (h *EstoqueHandler) FecharMes(w http.ResponseWriter, r *http.Request) {
	var dto fechamentoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.FecharMes(r.Context(), dto.Competence, false)
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/stock/closings/{id}/reopen
func (h *EstoqueHandler) ReabrirMes(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "fechamento")
	if !ok {
		return
	}
	var dto struct {
		Reason string `json:"reason"`
	}
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	err := h.UC.ReabrirMes(r.Context(), id, dto.Reason)
	h.responder(w, http.StatusOK, map[string]bool{"ok": err == nil}, err)
}

// GET /api/stock/lot-statuses
func (h *EstoqueHandler) SituacoesLote(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.SituacoesLote(r.Context())
	h.responder(w, http.StatusOK, out, err)
}

// PUT /api/stock/lot-statuses
func (h *EstoqueHandler) SalvarSituacaoLote(w http.ResponseWriter, r *http.Request) {
	var dto entity.SituacaoLote
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	err := h.UC.SalvarSituacaoLote(r.Context(), dto)
	h.responder(w, http.StatusOK, dto, err)
}

// GET /api/stock/parameters
func (h *EstoqueHandler) MotivosDeDivergencia(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.MotivosDeDivergencia(r.Context())
	h.responder(w, http.StatusOK, out, err)
}

func (h *EstoqueHandler) SalvarMotivoDeDivergencia(w http.ResponseWriter, r *http.Request) {
	var dto entity.MotivoDivergencia
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	err := h.UC.SalvarMotivoDeDivergencia(r.Context(), dto)
	h.responder(w, http.StatusOK, dto, err)
}

func (h *EstoqueHandler) Parametros(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Parametros(r.Context())
	h.responder(w, http.StatusOK, out, err)
}

// PUT /api/stock/parameters
func (h *EstoqueHandler) SalvarParametros(w http.ResponseWriter, r *http.Request) {
	var dto entity.ParametrosEstoque
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	err := h.UC.SalvarParametros(r.Context(), dto)
	h.responder(w, http.StatusOK, dto, err)
}

// GET /api/stock/item-control/{itemCode}
func (h *EstoqueHandler) ControleDoItem(w http.ResponseWriter, r *http.Request) {
	item, ok := inteiroDaRota(w, r, "itemCode", "item")
	if !ok {
		return
	}
	out, err := h.UC.ControleDoItem(r.Context(), item)
	h.responder(w, http.StatusOK, out, err)
}

// PUT /api/stock/item-control/{itemCode}
func (h *EstoqueHandler) SalvarControleDoItem(w http.ResponseWriter, r *http.Request) {
	item, ok := inteiroDaRota(w, r, "itemCode", "item")
	if !ok {
		return
	}
	var dto entity.ControleDoItem
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	dto.ItemCode = item
	err := h.UC.SalvarControleDoItem(r.Context(), dto)
	h.responder(w, http.StatusOK, dto, err)
}

// POST /api/stock/lots/status
func (h *EstoqueHandler) MudarSituacaoDoLote(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		ItemCode int64   `json:"item_code"`
		Mask     string  `json:"mask"`
		Lot      string  `json:"lot"`
		Status   *string `json:"status_code"`
		Grade    *string `json:"grade"`
		Reason   string  `json:"reason"`
	}
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	err := h.UC.MudarSituacaoDoLote(r.Context(), entity.MudancaSituacaoLote{ItemCode: dto.ItemCode, Mask: dto.Mask, Lot: dto.Lot, Status: dto.Status, Grade: dto.Grade, Reason: dto.Reason})
	h.responder(w, http.StatusOK, map[string]bool{"ok": err == nil}, err)
}

// POST /api/stock/lots/implant
func (h *EstoqueHandler) ImplantarLote(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		ItemCode    int64           `json:"item_code"`
		Mask        string          `json:"mask"`
		WarehouseID int64           `json:"warehouse_id"`
		Address     string          `json:"address"`
		Lot         string          `json:"lot"`
		Quantity    decimal.Decimal `json:"quantity"`
	}
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	err := h.UC.ImplantarLote(r.Context(), entity.ImplantacaoLote{ItemCode: dto.ItemCode, Mask: dto.Mask, WarehouseID: dto.WarehouseID, Address: dto.Address, Lot: dto.Lot, Quantity: dto.Quantity})
	h.responder(w, http.StatusOK, map[string]bool{"ok": err == nil}, err)
}

// GET /api/stock/inconsistencies?warehouse_id=&item_code=
func (h *EstoqueHandler) Inconsistencias(w http.ResponseWriter, r *http.Request) {
	var almox, item *int64
	if v, err := strconv.ParseInt(r.URL.Query().Get("warehouse_id"), 10, 64); err == nil && v > 0 {
		almox = &v
	}
	if v, err := strconv.ParseInt(r.URL.Query().Get("item_code"), 10, 64); err == nil && v > 0 {
		item = &v
	}
	out, err := h.UC.Inconsistencias(r.Context(), almox, item)
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/stock/inconsistencies/fix
func (h *EstoqueHandler) AcertarInconsistencia(w http.ResponseWriter, r *http.Request) {
	var dto request.AcertoInconsistenciaDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	err := h.UC.AcertarInconsistencia(r.Context(), dto)
	h.responder(w, http.StatusOK, map[string]bool{"ok": err == nil}, err)
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

// GET /api/stock/scan?code=&warehouse_id=
func (h *EstoqueHandler) ResolverLeitura(w http.ResponseWriter, r *http.Request) {
	var almox *int64
	if v := inteiroQuery(r, "warehouse_id"); v > 0 {
		id := int64(v)
		almox = &id
	}
	out, err := h.UC.ResolverLeitura(r.Context(), r.URL.Query().Get("code"), almox)
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/stock/transit
func (h *EstoqueHandler) EnviarParaTransito(w http.ResponseWriter, r *http.Request) {
	var dto request.EnvioTransitoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.EnviarParaTransito(r.Context(), dto)
	h.responder(w, http.StatusCreated, out, err)
}

// GET /api/stock/transit?open=true
func (h *EstoqueHandler) Transitos(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Transitos(r.Context(), r.URL.Query().Get("open") == "true")
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/stock/transit/{id}/settle
func (h *EstoqueHandler) BaixarTransito(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "envio")
	if !ok {
		return
	}
	var dto request.BaixaTransitoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.BaixarTransito(r.Context(), id, dto)
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/stock/replenishment/generate
func (h *EstoqueHandler) GerarReposicao(w http.ResponseWriter, r *http.Request) {
	var dto request.GerarReposicaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.GerarReposicao(r.Context(), dto)
	h.responder(w, http.StatusCreated, out, err)
}

// GET /api/stock/replenishment/policies
func (h *EstoqueHandler) PoliticasDeReposicao(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.PoliticasDeReposicao(r.Context())
	h.responder(w, http.StatusOK, out, err)
}

// PUT /api/stock/replenishment/policies
func (h *EstoqueHandler) SalvarPoliticaDeReposicao(w http.ResponseWriter, r *http.Request) {
	var dto request.PoliticaReposicaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.SalvarPoliticaDeReposicao(r.Context(), dto)
	h.responder(w, http.StatusOK, out, err)
}

// DELETE /api/stock/replenishment/policies/{id}
func (h *EstoqueHandler) ExcluirPoliticaDeReposicao(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "política")
	if !ok {
		return
	}
	err := h.UC.ExcluirPoliticaDeReposicao(r.Context(), id)
	h.responder(w, http.StatusOK, map[string]bool{"ok": err == nil}, err)
}

// GET /api/stock/analysis?months=&idle_days=
func (h *EstoqueHandler) Analise(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Analise(r.Context(), inteiroQuery(r, "months"), inteiroQuery(r, "idle_days"))
	h.responder(w, http.StatusOK, out, err)
}

// GET /api/stock/accuracy?months=&warehouse_id=&tol_a=&tol_b=&tol_c=
func (h *EstoqueHandler) Acuracidade(w http.ResponseWriter, r *http.Request) {
	var almox *int64
	if v := inteiroQuery(r, "warehouse_id"); v > 0 {
		id := int64(v)
		almox = &id
	}
	q := r.URL.Query()
	out, err := h.UC.Acuracidade(r.Context(), inteiroQuery(r, "months"), almox, q.Get("tol_a"), q.Get("tol_b"), q.Get("tol_c"))
	h.responder(w, http.StatusOK, out, err)
}

// GET /api/stock/aging
func (h *EstoqueHandler) IdadeDoEstoque(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.IdadeDoEstoque(r.Context())
	h.responder(w, http.StatusOK, out, err)
}

// /api/stock/kanban
func (h *EstoqueHandler) CartoesKanban(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.CartoesKanban(r.Context())
	h.responder(w, http.StatusOK, out, err)
}

func (h *EstoqueHandler) SalvarCartaoKanban(w http.ResponseWriter, r *http.Request) {
	var dto entity.CartaoKanban
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.SalvarCartaoKanban(r.Context(), dto)
	h.responder(w, http.StatusOK, out, err)
}

func (h *EstoqueHandler) AcionarCartaoKanban(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "cartão")
	if !ok {
		return
	}
	var dto struct {
		Cards int `json:"cards"`
	}
	if !security.DecodeBodyOptional(w, r, &dto) {
		return
	}
	out, err := h.UC.AcionarCartaoKanban(r.Context(), id, dto.Cards)
	h.responder(w, http.StatusCreated, out, err)
}

// GET /api/stock/waves/load-lines/{loadCode}?warehouse_id=
func (h *EstoqueHandler) LinhasDeCargaParaOnda(w http.ResponseWriter, r *http.Request) {
	carga, ok := inteiroDaRota(w, r, "loadCode", "carga")
	if !ok {
		return
	}
	almox := int64(inteiroQuery(r, "warehouse_id"))
	out, err := h.UC.LinhasDeCargaParaOnda(r.Context(), carga, almox)
	h.responder(w, http.StatusOK, out, err)
}

// GET /api/stock/aging/lots?warehouse_id=
func (h *EstoqueHandler) IdadeDosLotes(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.IdadeDosLotes(r.Context(), inteiroDaQuery(r.URL.Query(), "warehouse_id"))
	h.responder(w, http.StatusOK, out, err)
}

// GET /api/stock/replenishment/suggest-levels?item_code=&warehouse_id=&months=&lead_time_days=&safety_days=
func (h *EstoqueHandler) NiveisPeloConsumo(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	item := inteiroDaQuery(q, "item_code")
	if item == nil {
		security.RespondUseCaseError(w, errorsuc.NewValidationError("escolha o item"))
		return
	}
	out, err := h.UC.NiveisPeloConsumo(r.Context(), *item, q.Get("mask"), inteiroDaQuery(q, "warehouse_id"), inteiroQuery(r, "months"), inteiroQuery(r, "lead_time_days"), inteiroQuery(r, "safety_days"))
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

// GET /api/stock/inventory-tolerances
func (h *EstoqueHandler) ToleranciasPorPerfil(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.ToleranciasPorPerfil(r.Context())
	h.responder(w, http.StatusOK, out, err)
}

// PUT /api/stock/inventory-tolerances
func (h *EstoqueHandler) SalvarToleranciaDoPerfil(w http.ResponseWriter, r *http.Request) {
	var dto entity.ToleranciaDoPerfil
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	err := h.UC.SalvarToleranciaDoPerfil(r.Context(), dto)
	h.responder(w, http.StatusOK, dto, err)
}

// DELETE /api/stock/inventory-tolerances/{id}
func (h *EstoqueHandler) ExcluirToleranciaDoPerfil(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "perfil")
	if !ok {
		return
	}
	err := h.UC.ExcluirToleranciaDoPerfil(r.Context(), id)
	h.responder(w, http.StatusOK, map[string]bool{"ok": err == nil}, err)
}

// GET /api/inventory/{id}/notebooks
func (h *EstoqueHandler) Cadernos(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "inventário")
	if !ok {
		return
	}
	out, err := h.UC.Cadernos(r.Context(), id)
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/inventory/{id}/notebooks
func (h *EstoqueHandler) GerarCadernos(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "inventário")
	if !ok {
		return
	}
	var dto struct {
		LinesPerNotebook int `json:"lines_per_notebook"`
	}
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.GerarCadernos(r.Context(), id, dto.LinesPerNotebook)
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/inventory/{id}/notebooks/{number}/printed
func (h *EstoqueHandler) MarcarCadernoImpresso(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "inventário")
	if !ok {
		return
	}
	numero, ok := inteiroDaRota(w, r, "number", "caderno")
	if !ok {
		return
	}
	err := h.UC.MarcarCadernoImpresso(r.Context(), id, int(numero))
	h.responder(w, http.StatusOK, map[string]bool{"ok": err == nil}, err)
}

// GET /api/stock/consignments?direction=&open=
func (h *EstoqueHandler) Consignacoes(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Consignacoes(r.Context(), r.URL.Query().Get("direction"), r.URL.Query().Get("open") != "false")
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/stock/consignments
func (h *EstoqueHandler) ReceberConsignacao(w http.ResponseWriter, r *http.Request) {
	var dto request.ReceberConsignacaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.ReceberConsignacao(r.Context(), dto)
	h.responder(w, http.StatusCreated, out, err)
}

// POST /api/stock/consignments/{id}/return
func (h *EstoqueHandler) DevolverConsignacao(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "consignação")
	if !ok {
		return
	}
	var dto request.BaixaConsignacaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.DevolverConsignacao(r.Context(), id, dto)
	h.responder(w, http.StatusOK, out, err)
}

// POST /api/stock/consignments/{id}/consume
func (h *EstoqueHandler) ConsumirConsignacao(w http.ResponseWriter, r *http.Request) {
	id, ok := inteiroDaRota(w, r, "id", "consignação")
	if !ok {
		return
	}
	var dto request.BaixaConsignacaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.ConsumirConsignacao(r.Context(), id, dto)
	h.responder(w, http.StatusOK, out, err)
}

// GET /api/stock/cnpj-transfers
func (h *EstoqueHandler) TransferenciasEntreCNPJs(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.TransferenciasEntreCNPJs(r.Context())
	h.responder(w, http.StatusOK, out, err)
}
