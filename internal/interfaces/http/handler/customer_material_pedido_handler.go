package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	cmentity "github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	cmrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// ── Pedido de beneficiamento e relatórios ───────────────────────────────────

type vincularPedidoRequest struct {
	SalesOrderCode int64 `json:"sales_order_code"`
}

type linhaDoPedidoResponse struct {
	Code             int64  `json:"code"`
	Sequencia        int    `json:"sequence"`
	ItemCode         int64  `json:"item_code"`
	ItemBusinessCode string `json:"item_business_code"`
	Descricao        string `json:"description"`
	Unidade          string `json:"uom"`
	Quantidade       string `json:"requested_qty"`
	Atendida         string `json:"attended_qty"`
	Cancelada        string `json:"cancelled_qty"`
	EmNota           string `json:"in_draft_qty"`
	Saldo            string `json:"balance_qty"`
	PrecoUnitario    string `json:"unit_price"`
	Status           string `json:"status"`
}

type pedidoResponse struct {
	Code         int64                   `json:"code"`
	Numero       int64                   `json:"order_number"`
	CustomerCode int64                   `json:"customer_code"`
	Status       string                  `json:"status"`
	Linhas       []linhaDoPedidoResponse `json:"lines"`
}

type propostaResponse struct {
	Remessa remessaResponse `json:"remittance"`
	Pedido  *pedidoResponse `json:"sales_order"`
	Servico struct {
		CodigoItem         string `json:"codigo_item"`
		Descricao          string `json:"descricao"`
		Unidade            string `json:"unidade"`
		Quantidade         string `json:"quantidade"`
		ValorUnit          string `json:"valor_unitario"`
		SalesOrderItemCode *int64 `json:"sales_order_item_code"`
	} `json:"servico"`
	Devolucoes []devolucaoRequest `json:"devolucoes"`
	Aviso      string             `json:"aviso,omitempty"`
}

// PropostaDeFaturamento abre o faturamento já preenchido pelo pedido.
func (h *CustomerMaterialHandler) PropostaDeFaturamento(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	p, err := h.uc.PropostaDeFaturamento(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	out := propostaResponse{Remessa: montarRemessa(p.Remessa), Aviso: p.Aviso, Devolucoes: []devolucaoRequest{}}
	if p.Pedido != nil {
		ped := montarPedido(p.Pedido)
		out.Pedido = &ped
	}
	out.Servico.CodigoItem = p.Servico.CodigoItem
	out.Servico.Descricao = p.Servico.Descricao
	out.Servico.Unidade = p.Servico.Unidade
	out.Servico.Quantidade = p.Servico.Quantidade
	out.Servico.ValorUnit = p.Servico.ValorUnit
	out.Servico.SalesOrderItemCode = p.Servico.SalesOrderItemCode
	for _, d := range p.Devolucoes {
		out.Devolucoes = append(out.Devolucoes, devolucaoRequest{
			RemittanceItemID: d.RemittanceItemID, Quantity: d.Quantidade, MovementType: string(d.Tipo),
		})
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// VincularPedido liga a remessa ao pedido de beneficiamento depois do
// recebimento e libera o material bloqueado por falta de pedido.
func (h *CustomerMaterialHandler) VincularPedido(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	var body vincularPedidoRequest
	if !decodeCM(w, r, &body) {
		return
	}
	usuario, ok := usuarioDaSessaoCM(w, r)
	if !ok {
		return
	}
	remessa, err := h.uc.VincularPedido(r.Context(), id, body.SalesOrderCode, usuario)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, montarRemessa(remessa))
}

type movimentoDoRelatorioResponse struct {
	movimentoResponse
	RemittanceID          int64   `json:"remittance_id"`
	CustomerCode          int64   `json:"customer_code"`
	CustomerName          string  `json:"customer_name"`
	NFeNumber             int64   `json:"nfe_number"`
	NFeSeries             string  `json:"nfe_series"`
	CustomerItemCode      string  `json:"customer_item_code"`
	Description           string  `json:"description"`
	NCM                   string  `json:"ncm"`
	UOM                   string  `json:"uom"`
	NotaSaidaNumero       *int64  `json:"exit_nf_number"`
	NotaSaidaSerie        *string `json:"exit_nf_series"`
	NotaSaidaStatus       *string `json:"exit_nf_status"`
	ProductionOrderNumber *int64  `json:"production_order_number"`
}

// RelatorioDeMovimentos: GET /movements?customer_code&remittance_id&from&to&type&include_reversed&q&limit
func (h *CustomerMaterialHandler) RelatorioDeMovimentos(w http.ResponseWriter, r *http.Request) {
	if !queryValida(w, r, filtrosDaQuery{Inteiros: []string{"customer_code", "remittance_id", "limit"}, Datas: []string{"from", "to"}}) {
		return
	}
	q := r.URL.Query()
	f := cmrepo.FiltroMovimentos{
		CustomerCode:      inteiroDaQueryCM(q.Get("customer_code")),
		RemittanceID:      inteiroDaQueryCM(q.Get("remittance_id")),
		De:                dataDaQueryCM(q.Get("from")),
		Ate:               dataDaQueryCM(q.Get("to")),
		IncluirEstornados: q.Get("include_reversed") == "true",
		Q:                 q.Get("q"),
	}
	if v := inteiroDaQueryCM(q.Get("limit")); v != nil {
		f.Limite = int(*v)
	}
	for _, bruto := range q["type"] {
		for _, t := range strings.Split(bruto, ",") {
			if t = strings.ToUpper(strings.TrimSpace(t)); t != "" {
				f.Tipos = append(f.Tipos, t)
			}
		}
	}
	linhas, err := h.uc.RelatorioDeMovimentos(r.Context(), f)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	out := make([]movimentoDoRelatorioResponse, 0, len(linhas))
	for _, l := range linhas {
		out = append(out, movimentoDoRelatorioResponse{
			movimentoResponse: montarMovimento(&l.Movimento),
			RemittanceID:      l.RemittanceID, CustomerCode: l.CustomerCode, CustomerName: l.CustomerName,
			NFeNumber: l.NFeNumber, NFeSeries: l.NFeSeries,
			CustomerItemCode: l.CustomerItemCode, Description: l.Description, NCM: l.NCM, UOM: l.UOM,
			NotaSaidaNumero: l.NotaSaidaNumero, NotaSaidaSerie: l.NotaSaidaSerie, NotaSaidaStatus: l.NotaSaidaStatus, ProductionOrderNumber: l.ProductionOrderNumber,
		})
	}
	security.RespondJSON(w, http.StatusOK, out)
}

type horasResponse struct {
	ProductionOrderID int64   `json:"production_order_id"`
	OrderNumber       int64   `json:"order_number"`
	Status            string  `json:"status"`
	ItemCode          int64   `json:"item_code"`
	ItemBusinessCode  string  `json:"item_business_code"`
	ItemDescription   string  `json:"item_description"`
	PlannedQty        string  `json:"planned_qty"`
	ProducedQty       string  `json:"produced_qty"`
	CustomerCode      *int64  `json:"customer_code"`
	CustomerName      string  `json:"customer_name"`
	SalesOrderCode    *int64  `json:"sales_order_code"`
	SalesOrderNumber  *int64  `json:"sales_order_number"`
	HorasApontadas    string  `json:"appointed_hours"`
	Apontamentos      int32   `json:"appointments"`
	HorasDeOperacao   string  `json:"operation_hours"`
	PrimeiroRegistro  *string `json:"first_record"`
	UltimoRegistro    *string `json:"last_record"`
}

// HorasDeBeneficiamento: GET /hours?customer_code&sales_order_code&from&to
func (h *CustomerMaterialHandler) HorasDeBeneficiamento(w http.ResponseWriter, r *http.Request) {
	if !queryValida(w, r, filtrosDaQuery{Inteiros: []string{"customer_code", "sales_order_code"}, Datas: []string{"from", "to"}}) {
		return
	}
	q := r.URL.Query()
	linhas, err := h.uc.HorasDeBeneficiamento(r.Context(), cmrepo.FiltroHoras{
		CustomerCode:   inteiroDaQueryCM(q.Get("customer_code")),
		SalesOrderCode: inteiroDaQueryCM(q.Get("sales_order_code")),
		De:             dataDaQueryCM(q.Get("from")),
		Ate:            dataDaQueryCM(q.Get("to")),
	})
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	out := make([]horasResponse, 0, len(linhas))
	for _, l := range linhas {
		out = append(out, horasResponse{
			ProductionOrderID: l.ProductionOrderID, OrderNumber: l.OrderNumber, Status: l.Status,
			ItemCode: l.ItemCode, ItemBusinessCode: l.ItemBusinessCode, ItemDescription: l.ItemDescription,
			PlannedQty: l.PlannedQty.String(), ProducedQty: l.ProducedQty.String(),
			CustomerCode: l.CustomerCode, CustomerName: l.CustomerName,
			SalesOrderCode: l.SalesOrderCode, SalesOrderNumber: l.SalesOrderNumber,
			HorasApontadas: l.HorasApontadas.Round(2).StringFixed(2), Apontamentos: l.Apontamentos,
			HorasDeOperacao:  l.HorasDeOperacao.Round(2).StringFixed(2),
			PrimeiroRegistro: dataCM(l.PrimeiroRegistro), UltimoRegistro: dataCM(l.UltimoRegistro),
		})
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func montarPedido(p *cmentity.PedidoDeBeneficiamento) pedidoResponse {
	out := pedidoResponse{Code: p.Code, Numero: p.Numero, CustomerCode: p.CustomerCode, Status: p.Status,
		Linhas: make([]linhaDoPedidoResponse, 0, len(p.Linhas))}
	for _, l := range p.Linhas {
		out.Linhas = append(out.Linhas, linhaDoPedidoResponse{
			Code: l.Code, Sequencia: l.Sequencia, ItemCode: l.ItemCode, ItemBusinessCode: l.ItemBusinessCode,
			Descricao: l.Descricao, Unidade: l.Unidade, Quantidade: l.Quantidade.String(),
			Atendida: l.Atendida.String(), Cancelada: l.Cancelada.String(), EmNota: l.EmNota.String(),
			Saldo: l.Saldo().String(), PrecoUnitario: l.PrecoUnitario.String(), Status: l.Status,
		})
	}
	return out
}

// A query já foi validada por queryValida: aqui só se converte.
func inteiroDaQueryCM(v string) *int64 {
	if ausenteNaQuery(v) {
		return nil
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func dataDaQueryCM(v string) *time.Time {
	if ausenteNaQuery(v) {
		return nil
	}
	t, err := time.Parse(formatoDataCM, strings.TrimSpace(v))
	if err != nil {
		return nil
	}
	return &t
}

func dataCM(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(formatoDataCM)
	return &s
}

type descartarNotaRequest struct {
	Reason string `json:"reason"`
}

// DescartarNota tira do caminho a nota que não chegou a ser autorizada e devolve o
// saldo do cliente.
func (h *CustomerMaterialHandler) DescartarNota(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	var body descartarNotaRequest
	if !decodeCM(w, r, &body) {
		return
	}
	usuario, ok := usuarioDaSessaoCM(w, r)
	if !ok {
		return
	}
	numero, err := h.uc.DescartarNota(r.Context(), id, usuario, body.Reason)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]int64{"discarded_number": numero})
}

type vincularPedidoDaOFRequest struct {
	SalesOrderCode *int64 `json:"sales_order_code"`
}

// VincularPedidoDaOF liga a OF aberta à mão ao pedido; nulo desfaz o vínculo.
func (h *CustomerMaterialHandler) VincularPedidoDaOF(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	var body vincularPedidoDaOFRequest
	if !decodeCM(w, r, &body) {
		return
	}
	if err := h.uc.VincularPedidoDaOF(r.Context(), id, body.SalesOrderCode); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "pedido da ordem atualizado"})
}
