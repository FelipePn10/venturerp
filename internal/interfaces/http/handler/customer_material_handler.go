package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/customer_material_uc"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	cmentity "github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	cmrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// CustomerMaterialHandler expõe o beneficiamento: material do cliente em poder da
// empresa, do recebimento da NF-e de remessa ao retorno fiscal.
type CustomerMaterialHandler struct {
	uc *customer_material_uc.UseCase
}

func NewCustomerMaterialHandler(uc *customer_material_uc.UseCase) *CustomerMaterialHandler {
	return &CustomerMaterialHandler{uc: uc}
}

const formatoDataCM = "2006-01-02"

// ── Entrada ──────────────────────────────────────────────────────────────────

type itemRemessaRequest struct {
	LineNumber       int32   `json:"line_number"`
	CustomerItemCode string  `json:"customer_item_code"`
	ItemCode         *int64  `json:"item_code"`
	Description      string  `json:"description"`
	NCM              string  `json:"ncm"`
	CST              *string `json:"cst"`
	UOM              string  `json:"uom"`
	// Quantidades e valores viajam como TEXTO: passar por float perderia centavo
	// e a nota de retorno sairia divergente da de entrada.
	QtyInvoiced      string  `json:"qty_invoiced"`
	QtyReceived      string  `json:"qty_received"`
	UnitValue        string  `json:"unit_value"`
	DivergenceReason *string `json:"divergence_reason"`
	WarehouseID      *int64  `json:"warehouse_id"`
	Address          *string `json:"address"`
}

type remessaRequest struct {
	CustomerCode         int64                `json:"customer_code"`
	NFeNumber            int64                `json:"nfe_number"`
	NFeSeries            string               `json:"nfe_series"`
	NFeKey               *string              `json:"nfe_key"`
	CFOP                 string               `json:"cfop"`
	IssueDate            string               `json:"issue_date"`
	ReceivedAt           string               `json:"received_at"`
	FiscalReturnDeadline string               `json:"fiscal_return_deadline"`
	TotalValue           string               `json:"total_value"`
	SalesOrderCode       *int64               `json:"sales_order_code"`
	Notes                *string              `json:"notes"`
	Itens                []itemRemessaRequest `json:"itens"`
}

type movimentoRequest struct {
	MovementType      string  `json:"movement_type"`
	Quantity          string  `json:"quantity"`
	UnitValue         string  `json:"unit_value"`
	CFOP              *string `json:"cfop"`
	ProductionOrderID *int64  `json:"production_order_id"`
	FiscalExitID      *int64  `json:"fiscal_exit_id"`
	ScrapDestination  *string `json:"scrap_destination"`
	Reason            *string `json:"reason"`
	IdempotencyKey    string  `json:"idempotency_key"`
}

type motivoRequest struct {
	Reason string `json:"reason"`
}

type devolucaoRequest struct {
	RemittanceItemID int64  `json:"remittance_item_id"`
	Quantity         string `json:"quantity"`
	// RETURN, LEFTOVER ou SCRAP. Vazio vale como RETURN.
	MovementType string `json:"movement_type"`
}

type faturarRequest struct {
	Servico struct {
		CodigoItem string `json:"codigo_item"`
		Descricao  string `json:"descricao"`
		Unidade    string `json:"unidade"`
		Quantidade string `json:"quantidade"`
		ValorUnit  string `json:"valor_unitario"`
	} `json:"servico"`
	Devolucoes   []devolucaoRequest `json:"devolucoes"`
	Destinatario struct {
		CNPJ            string `json:"cnpj"`
		RazaoSocial     string `json:"razao_social"`
		IE              string `json:"inscricao_estadual"`
		UF              string `json:"uf"`
		Logradouro      string `json:"logradouro"`
		Numero          string `json:"numero"`
		Complemento     string `json:"complemento"`
		Bairro          string `json:"bairro"`
		Municipio       string `json:"municipio"`
		CodigoMunicipio string `json:"codigo_municipio"`
		CEP             string `json:"cep"`
		Email           string `json:"email"`
		Telefone        string `json:"telefone"`
	} `json:"destinatario"`
	Serie       string `json:"serie"`
	DataEmissao string `json:"data_emissao"`
}

type linhaDaNotaResponse struct {
	Sequencia   int     `json:"sequencia"`
	Descricao   string  `json:"descricao"`
	NCM         string  `json:"ncm"`
	CFOP        string  `json:"cfop"`
	CSTICMS     string  `json:"cst_icms"`
	Unidade     string  `json:"unidade"`
	Quantidade  string  `json:"quantidade"`
	ValorUnit   string  `json:"valor_unitario"`
	ValorTotal  string  `json:"valor_total"`
	CSTPIS      *string `json:"cst_pis"`
	ValorPIS    string  `json:"valor_pis"`
	CSTCOFINS   *string `json:"cst_cofins"`
	ValorCOFINS string  `json:"valor_cofins"`
}

type faturamentoResponse struct {
	FiscalExitID     int64                 `json:"fiscal_exit_id"`
	NumeroNF         int64                 `json:"numero_nf"`
	Serie            string                `json:"serie"`
	NaturezaOperacao string                `json:"natureza_operacao"`
	Observacao       string                `json:"observacao"`
	ValorServico     string                `json:"valor_servico"`
	ValorMaterial    string                `json:"valor_material"`
	ValorTotal       string                `json:"valor_total"`
	ValorPIS         string                `json:"valor_pis"`
	ValorCOFINS      string                `json:"valor_cofins"`
	SaldoRestante    string                `json:"saldo_restante"`
	Linhas           []linhaDaNotaResponse `json:"linhas"`
}

// ── Saída ────────────────────────────────────────────────────────────────────

type itemRemessaResponse struct {
	ID               int64   `json:"id"`
	LineNumber       int32   `json:"line_number"`
	CustomerItemCode string  `json:"customer_item_code"`
	ItemCode         *int64  `json:"item_code"`
	Description      string  `json:"description"`
	NCM              string  `json:"ncm"`
	CST              *string `json:"cst"`
	UOM              string  `json:"uom"`
	QtyInvoiced      string  `json:"qty_invoiced"`
	QtyReceived      string  `json:"qty_received"`
	QtyReturned      string  `json:"qty_returned"`
	QtyLeftover      string  `json:"qty_leftover"`
	QtyScrapped      string  `json:"qty_scrapped"`
	BalanceQty       string  `json:"balance_qty"`
	DivergenceQty    string  `json:"divergence_qty"`
	DivergenceReason *string `json:"divergence_reason"`
	UnitValue        string  `json:"unit_value"`
	WarehouseID      *int64  `json:"warehouse_id"`
	Address          *string `json:"address"`
}

type remessaResponse struct {
	ID                   int64   `json:"id"`
	CustomerCode         int64   `json:"customer_code"`
	NFeNumber            int64   `json:"nfe_number"`
	NFeSeries            string  `json:"nfe_series"`
	NFeKey               *string `json:"nfe_key"`
	CFOP                 string  `json:"cfop"`
	IssueDate            string  `json:"issue_date"`
	ReceivedAt           string  `json:"received_at"`
	FiscalReturnDeadline string  `json:"fiscal_return_deadline"`
	// DiasParaOPrazo é negativo quando o prazo já passou. Vem calculado do
	// servidor para a tela não depender do relógio da máquina do usuário.
	DiasParaOPrazo int                   `json:"dias_para_o_prazo"`
	TotalValue     string                `json:"total_value"`
	Status         string                `json:"status"`
	SalesOrderCode *int64                `json:"sales_order_code"`
	Blocked        bool                  `json:"blocked"`
	BlockReason    *string               `json:"block_reason"`
	CloseReason    *string               `json:"close_reason"`
	Notes          *string               `json:"notes"`
	SaldoTotal     string                `json:"saldo_total"`
	Itens          []itemRemessaResponse `json:"itens"`
}

type movimentoResponse struct {
	ID                int64   `json:"id"`
	RemittanceItemID  int64   `json:"remittance_item_id"`
	MovementType      string  `json:"movement_type"`
	Quantity          string  `json:"quantity"`
	UnitValue         string  `json:"unit_value"`
	CFOP              *string `json:"cfop"`
	ProductionOrderID *int64  `json:"production_order_id"`
	FiscalExitID      *int64  `json:"fiscal_exit_id"`
	ScrapDestination  *string `json:"scrap_destination"`
	Reason            *string `json:"reason"`
	CreatedAt         string  `json:"created_at"`
	// Estorno: movimento estornado não baixa mais saldo. A tela mostra a linha
	// marcada em vez de esconder — "saiu e voltou" é informação de conferência.
	ReversedAt     *string `json:"reversed_at"`
	ReversalReason *string `json:"reversal_reason"`
}

type saldoResponse struct {
	CustomerCode     int64   `json:"customer_code"`
	CustomerName     string  `json:"customer_name"`
	CustomerItemCode string  `json:"customer_item_code"`
	ItemCode         *int64  `json:"item_code"`
	Description      string  `json:"description"`
	UOM              string  `json:"uom"`
	Balance          string  `json:"balance"`
	RemessasAbertas  int32   `json:"remessas_abertas"`
	PrazoMaisProximo *string `json:"prazo_mais_proximo"`
}

// ── Rotas ────────────────────────────────────────────────────────────────────

func (h *CustomerMaterialHandler) Receber(w http.ResponseWriter, r *http.Request) {
	var body remessaRequest
	if !decodeCM(w, r, &body) {
		return
	}
	usuario, ok := usuarioDaSessaoCM(w, r)
	if !ok {
		return
	}
	emissao, err := dataObrigatoriaCM(body.IssueDate, "data de emissão")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	recebimento, err := dataOpcionalCM(body.ReceivedAt, "data de recebimento")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	prazo, err := dataOpcionalCM(body.FiscalReturnDeadline, "prazo de retorno")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	nova := customer_material_uc.NovaRemessa{
		CustomerCode: body.CustomerCode, NFeNumber: body.NFeNumber, NFeSeries: body.NFeSeries,
		NFeKey: body.NFeKey, CFOP: body.CFOP, IssueDate: emissao,
		FiscalReturnDeadline: prazo, TotalValue: body.TotalValue,
		SalesOrderCode: body.SalesOrderCode, Notes: body.Notes,
	}
	if recebimento != nil {
		nova.ReceivedAt = *recebimento
	}
	for _, item := range body.Itens {
		nova.Itens = append(nova.Itens, customer_material_uc.ItemRecebido{
			LineNumber: item.LineNumber, CustomerItemCode: item.CustomerItemCode,
			ItemCode: item.ItemCode, Description: item.Description, NCM: item.NCM,
			CST: item.CST, UOM: item.UOM, QtyInvoiced: item.QtyInvoiced,
			QtyReceived: item.QtyReceived, UnitValue: item.UnitValue,
			DivergenceReason: item.DivergenceReason, WarehouseID: item.WarehouseID,
			Address: item.Address,
		})
	}

	remessa, err := h.uc.Receber(r.Context(), nova, usuario)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, montarRemessa(remessa))
}

func (h *CustomerMaterialHandler) Listar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filtro := cmrepo.FiltroRemessa{
		Busca:            q.Get("q"),
		SomenteBloqueada: q.Get("blocked") == "true",
		SomenteComSaldo:  q.Get("with_balance") == "true",
	}
	if v := q.Get("customer_code"); v != "" {
		if code, err := strconv.ParseInt(v, 10, 64); err == nil {
			filtro.CustomerCode = &code
		}
	}
	if v := q.Get("nfe_number"); v != "" {
		if numero, err := strconv.ParseInt(v, 10, 64); err == nil {
			filtro.NFeNumber = &numero
		}
	}
	if v := q.Get("sales_order_code"); v != "" {
		if pedido, err := strconv.ParseInt(v, 10, 64); err == nil {
			filtro.SalesOrderCode = &pedido
		}
	}
	for _, s := range strings.Split(q.Get("status"), ",") {
		if s = strings.ToUpper(strings.TrimSpace(s)); s != "" {
			filtro.Status = append(filtro.Status, cmentity.StatusRemessa(s))
		}
	}
	if v := q.Get("due_until"); v != "" {
		prazo, err := dataOpcionalCM(v, "prazo limite")
		if err != nil {
			security.RespondUseCaseError(w, err)
			return
		}
		filtro.VencendoAte = prazo
	}
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 {
		filtro.Limit = v
	}
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v > 0 {
		filtro.Offset = v
	}

	lista, err := h.uc.Listar(r.Context(), filtro)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	out := make([]remessaResponse, 0, len(lista))
	for _, remessa := range lista {
		out = append(out, montarRemessa(remessa))
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CustomerMaterialHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	remessa, err := h.uc.Obter(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, montarRemessa(remessa))
}

// Saldo é a visão que o inventário mostra ao lado do estoque próprio.
func (h *CustomerMaterialHandler) Saldo(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filtro := cmrepo.FiltroSaldo{
		CustomerItemCode: q.Get("customer_item_code"),
		Busca:            q.Get("q"),
	}
	if v := q.Get("customer_code"); v != "" {
		if code, err := strconv.ParseInt(v, 10, 64); err == nil {
			filtro.CustomerCode = &code
		}
	}
	if v := q.Get("item_code"); v != "" {
		if code, err := strconv.ParseInt(v, 10, 64); err == nil {
			filtro.ItemCode = &code
		}
	}
	saldos, err := h.uc.Saldo(r.Context(), filtro)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	out := make([]saldoResponse, 0, len(saldos))
	for _, s := range saldos {
		linha := saldoResponse{
			CustomerCode: s.CustomerCode, CustomerName: s.CustomerName,
			CustomerItemCode: s.CustomerItemCode, ItemCode: s.ItemCode,
			Description: s.Description, UOM: s.UOM, Balance: s.Balance.String(),
			RemessasAbertas: s.RemessasAbertas,
		}
		if s.PrazoMaisProximo != nil {
			prazo := s.PrazoMaisProximo.Format(formatoDataCM)
			linha.PrazoMaisProximo = &prazo
		}
		out = append(out, linha)
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CustomerMaterialHandler) Movimentar(w http.ResponseWriter, r *http.Request) {
	itemID, ok := idDaRotaCM(w, r, "itemId")
	if !ok {
		return
	}
	var body movimentoRequest
	if !decodeCM(w, r, &body) {
		return
	}
	usuario, ok := usuarioDaSessaoCM(w, r)
	if !ok {
		return
	}

	mov := cmrepo.NovoMovimento{
		RemittanceItemID:  itemID,
		MovementType:      cmentity.TipoMovimento(strings.ToUpper(strings.TrimSpace(body.MovementType))),
		Quantity:          body.Quantity,
		UnitValue:         body.UnitValue,
		CFOP:              body.CFOP,
		ProductionOrderID: body.ProductionOrderID,
		FiscalExitID:      body.FiscalExitID,
		Reason:            body.Reason,
		IdempotencyKey:    body.IdempotencyKey,
	}
	if !tipoDeMovimentoValido(mov.MovementType) {
		security.RespondUseCaseError(w, errorsuc.NewValidationError(
			"tipo de movimento inválido; use RETURN, LEFTOVER, SCRAP ou ADJUSTMENT"))
		return
	}
	if body.ScrapDestination != nil {
		destino := cmentity.DestinoSucata(strings.ToUpper(strings.TrimSpace(*body.ScrapDestination)))
		if !destinoDeSucataValido(destino) {
			security.RespondUseCaseError(w, errorsuc.NewValidationError(
				"destinação de sucata inválida; use CLIENTE, DESCARTE, RETENCAO ou OUTRA"))
			return
		}
		mov.ScrapDestination = &destino
	}

	criado, err := h.uc.Movimentar(r.Context(), mov, usuario)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, montarMovimento(criado))
}

func (h *CustomerMaterialHandler) Movimentos(w http.ResponseWriter, r *http.Request) {
	itemID, ok := idDaRotaCM(w, r, "itemId")
	if !ok {
		return
	}
	movimentos, err := h.uc.Movimentos(r.Context(), itemID)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	out := make([]movimentoResponse, 0, len(movimentos))
	for _, m := range movimentos {
		out = append(out, montarMovimento(m))
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// Faturar cria a nota que cobra o serviço e devolve o material do cliente, e baixa
// o saldo de terceiro. A nota nasce em RASCUNHO: transmitir é passo separado.
func (h *CustomerMaterialHandler) Faturar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	var body faturarRequest
	if !decodeCM(w, r, &body) {
		return
	}
	usuario, ok := usuarioDaSessaoCM(w, r)
	if !ok {
		return
	}
	emissao, err := dataOpcionalCM(body.DataEmissao, "data de emissão")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	pedido := customer_material_uc.PedidoDeFaturamento{
		RemittanceID: id,
		Serie:        body.Serie,
		DataEmissao:  emissao,
		Servico: customer_material_uc.ServicoFaturado{
			CodigoItem: body.Servico.CodigoItem,
			Descricao:  body.Servico.Descricao,
			Unidade:    body.Servico.Unidade,
			Quantidade: body.Servico.Quantidade,
			ValorUnit:  body.Servico.ValorUnit,
		},
	}

	// Destinatário informado é exceção: o comum é a tela não mandar nada e o
	// servidor resolver pelo cadastro do cliente da remessa. Só sobrescreve quando
	// veio algo de verdade.
	if strings.TrimSpace(body.Destinatario.CNPJ) != "" || strings.TrimSpace(body.Destinatario.RazaoSocial) != "" {
		pedido.Destinatario = &customer_material_uc.DadosDoDestinatario{
			CNPJ: body.Destinatario.CNPJ, RazaoSocial: body.Destinatario.RazaoSocial,
			IE: body.Destinatario.IE, UF: body.Destinatario.UF,
			Logradouro: body.Destinatario.Logradouro, Numero: body.Destinatario.Numero,
			Complemento: body.Destinatario.Complemento, Bairro: body.Destinatario.Bairro,
			Municipio: body.Destinatario.Municipio, CodigoMunicipio: body.Destinatario.CodigoMunicipio,
			CEP: body.Destinatario.CEP, Email: body.Destinatario.Email,
			Telefone: body.Destinatario.Telefone,
		}
	}
	for _, dev := range body.Devolucoes {
		tipo := cmentity.TipoMovimento(strings.ToUpper(strings.TrimSpace(dev.MovementType)))
		if tipo == "" {
			tipo = cmentity.MovimentoRetorno
		}
		pedido.Devolucoes = append(pedido.Devolucoes, customer_material_uc.DevolucaoDeMaterial{
			RemittanceItemID: dev.RemittanceItemID,
			Quantidade:       dev.Quantity,
			Tipo:             tipo,
		})
	}

	resultado, err := h.uc.Faturar(r.Context(), pedido, usuario)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, montarFaturamento(resultado))
}

func (h *CustomerMaterialHandler) Bloquear(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	var body motivoRequest
	if !decodeCM(w, r, &body) {
		return
	}
	usuario, ok := usuarioDaSessaoCM(w, r)
	if !ok {
		return
	}
	if err := h.uc.Bloquear(r.Context(), id, body.Reason, usuario); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "remessa bloqueada"})
}

func (h *CustomerMaterialHandler) Desbloquear(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	usuario, ok := usuarioDaSessaoCM(w, r)
	if !ok {
		return
	}
	if err := h.uc.Desbloquear(r.Context(), id, usuario); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "remessa liberada"})
}

func (h *CustomerMaterialHandler) Encerrar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	var body motivoRequest
	if !decodeCM(w, r, &body) {
		return
	}
	usuario, ok := usuarioDaSessaoCM(w, r)
	if !ok {
		return
	}
	if err := h.uc.Encerrar(r.Context(), id, body.Reason, usuario); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "remessa encerrada"})
}

// ── Apoio ────────────────────────────────────────────────────────────────────

func montarRemessa(r *cmentity.Remessa) remessaResponse {
	out := remessaResponse{
		ID: r.ID, CustomerCode: r.CustomerCode, NFeNumber: r.NFeNumber,
		NFeSeries: r.NFeSeries, NFeKey: r.NFeKey, CFOP: r.CFOP,
		IssueDate:            r.IssueDate.Format(formatoDataCM),
		ReceivedAt:           r.ReceivedAt.Format(formatoDataCM),
		FiscalReturnDeadline: r.FiscalReturnDeadline.Format(formatoDataCM),
		// Truncar para o dia evita que uma diferença de horas vire um dia a mais
		// ou a menos na contagem que a tela mostra.
		DiasParaOPrazo: int(r.FiscalReturnDeadline.Truncate(24*time.Hour).
			Sub(time.Now().UTC().Truncate(24*time.Hour)).Hours() / 24),
		TotalValue: r.TotalValue.String(), Status: string(r.Status),
		SalesOrderCode: r.SalesOrderCode, Blocked: r.Blocked, BlockReason: r.BlockReason,
		CloseReason: r.CloseReason, Notes: r.Notes,
		SaldoTotal: r.SaldoTotal().String(),
		Itens:      make([]itemRemessaResponse, 0, len(r.Itens)),
	}
	for _, item := range r.Itens {
		out.Itens = append(out.Itens, itemRemessaResponse{
			ID: item.ID, LineNumber: item.LineNumber, CustomerItemCode: item.CustomerItemCode,
			ItemCode: item.ItemCode, Description: item.Description, NCM: item.NCM,
			CST: item.CST, UOM: item.UOM,
			QtyInvoiced: item.QtyInvoiced.String(), QtyReceived: item.QtyReceived.String(),
			QtyReturned: item.QtyReturned.String(), QtyLeftover: item.QtyLeftover.String(),
			QtyScrapped: item.QtyScrapped.String(), BalanceQty: item.BalanceQty.String(),
			DivergenceQty: item.DivergenceQty.String(), DivergenceReason: item.DivergenceReason,
			UnitValue: item.UnitValue.String(), WarehouseID: item.WarehouseID, Address: item.Address,
		})
	}
	return out
}

func montarFaturamento(r *customer_material_uc.ResultadoDoFaturamento) faturamentoResponse {
	out := faturamentoResponse{
		FiscalExitID: r.FiscalExitID, NumeroNF: r.NumeroNF, Serie: r.Serie,
		NaturezaOperacao: r.Nota.NaturezaOperacao, Observacao: r.Nota.Observacao,
		ValorServico: r.Nota.ValorServico.String(), ValorMaterial: r.Nota.ValorMaterial.String(),
		ValorTotal: r.Nota.ValorTotal.String(), ValorPIS: r.Nota.ValorPIS.String(),
		ValorCOFINS: r.Nota.ValorCOFINS.String(), SaldoRestante: r.SaldoRestante.String(),
		Linhas: make([]linhaDaNotaResponse, 0, len(r.Nota.Linhas)),
	}
	for _, linha := range r.Nota.Linhas {
		item := linhaDaNotaResponse{
			Sequencia: linha.Sequencia, Descricao: linha.Descricao, NCM: linha.NCM,
			CFOP: linha.CFOP, CSTICMS: linha.CSTICMS, Unidade: linha.Unidade,
			Quantidade: linha.Quantidade.String(), ValorUnit: linha.ValorUnit.String(),
			ValorTotal: linha.ValorTotal.String(),
			ValorPIS:   linha.ValorPIS.String(), ValorCOFINS: linha.ValorCOFINS.String(),
		}
		if linha.CSTPIS != "" {
			cst := linha.CSTPIS
			item.CSTPIS = &cst
		}
		if linha.CSTCOFINS != "" {
			cst := linha.CSTCOFINS
			item.CSTCOFINS = &cst
		}
		out.Linhas = append(out.Linhas, item)
	}
	return out
}

func montarMovimento(m *cmentity.Movimento) movimentoResponse {
	out := movimentoResponse{
		ID: m.ID, RemittanceItemID: m.RemittanceItemID,
		MovementType: string(m.MovementType), Quantity: m.Quantity.String(),
		UnitValue: m.UnitValue.String(), CFOP: m.CFOP,
		ProductionOrderID: m.ProductionOrderID, FiscalExitID: m.FiscalExitID,
		Reason: m.Reason, CreatedAt: m.CreatedAt.Format(time.RFC3339),
	}
	if m.ScrapDestination != nil {
		destino := string(*m.ScrapDestination)
		out.ScrapDestination = &destino
	}
	if m.ReversedAt != nil {
		quando := m.ReversedAt.Format(time.RFC3339)
		out.ReversedAt = &quando
		out.ReversalReason = m.ReversalReason
	}
	return out
}

func tipoDeMovimentoValido(t cmentity.TipoMovimento) bool {
	// RECEIPT fica de fora: a entrada é criada pelo recebimento da remessa, e
	// aceitá-la aqui permitiria inflar o saldo do cliente sem nota que a sustente.
	for _, valido := range cmentity.TiposDeSaida() {
		if t == valido {
			return true
		}
	}
	return false
}

func destinoDeSucataValido(d cmentity.DestinoSucata) bool {
	switch d {
	case cmentity.DestinoCliente, cmentity.DestinoDescarte, cmentity.DestinoRetencao, cmentity.DestinoOutra:
		return true
	}
	return false
}

func decodeCM(w http.ResponseWriter, r *http.Request, alvo any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(alvo); err != nil {
		security.RespondUseCaseError(w, errorsuc.NewValidationError("JSON inválido: "+err.Error()))
		return false
	}
	return true
}

func idDaRotaCM(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil || id <= 0 {
		security.RespondUseCaseError(w, errorsuc.NewValidationError("identificador inválido"))
		return 0, false
	}
	return id, true
}

func dataObrigatoriaCM(bruto, campo string) (time.Time, error) {
	if strings.TrimSpace(bruto) == "" {
		return time.Time{}, errorsuc.NewValidationError("informe a " + campo)
	}
	d, err := time.Parse(formatoDataCM, strings.TrimSpace(bruto))
	if err != nil {
		return time.Time{}, errorsuc.NewValidationError(campo + " inválida; use AAAA-MM-DD")
	}
	return d, nil
}

func dataOpcionalCM(bruto, campo string) (*time.Time, error) {
	if strings.TrimSpace(bruto) == "" {
		return nil, nil
	}
	d, err := dataObrigatoriaCM(bruto, campo)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// Estornar devolve ao saldo do cliente o que uma nota de retorno baixou. O
// cancelamento da NF-e já chama isto sozinho; a rota existe para concluir quando o
// cancelamento passou na SEFAZ e o estorno falhou no meio. Repetir não devolve
// duas vezes.
func (h *CustomerMaterialHandler) Estornar(w http.ResponseWriter, r *http.Request) {
	fiscalExitID, ok := idDaRotaCM(w, r, "fiscalExitId")
	if !ok {
		return
	}
	var body motivoRequest
	if !decodeCM(w, r, &body) {
		return
	}
	usuario, ok := usuarioDaSessaoCM(w, r)
	if !ok {
		return
	}
	movimentos, err := h.uc.EstornarNota(r.Context(), fiscalExitID, usuario, body.Reason)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	saida := make([]movimentoResponse, 0, len(movimentos))
	for _, m := range movimentos {
		saida = append(saida, montarMovimento(m))
	}
	security.RespondJSON(w, http.StatusOK, map[string]any{
		"message":   "saldo devolvido ao cliente",
		"reversed":  len(saida),
		"movements": saida,
	})
}

// auditoriaResponse é uma linha do histórico da remessa. `before`/`after` saem
// como JSON cru porque é assim que a trilha guarda — a tela decide quais campos
// mostrar, e um campo novo na tabela aparece no histórico sem mexer aqui.
type auditoriaResponse struct {
	ID            int64           `json:"id"`
	EntityType    string          `json:"entity_type"`
	EntityID      int64           `json:"entity_id"`
	Action        string          `json:"action"`
	ChangedFields []string        `json:"changed_fields,omitempty"`
	Before        json.RawMessage `json:"before,omitempty"`
	After         json.RawMessage `json:"after,omitempty"`
	Reason        *string         `json:"reason,omitempty"`
	ActorID       *string         `json:"actor_id,omitempty"`
	ActorName     *string         `json:"actor_name,omitempty"`
	ActorEmail    *string         `json:"actor_email,omitempty"`
	OccurredAt    string          `json:"occurred_at"`
}

// Trilha responde o histórico e a auditoria da remessa: quem, quando, o que fez,
// como estava, como ficou e por quê.
func (h *CustomerMaterialHandler) Trilha(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaCM(w, r, "id")
	if !ok {
		return
	}
	limite := 0
	if bruto := strings.TrimSpace(r.URL.Query().Get("limit")); bruto != "" {
		n, err := strconv.Atoi(bruto)
		if err != nil || n <= 0 {
			security.RespondUseCaseError(w, errorsuc.NewValidationError("limit inválido"))
			return
		}
		limite = n
	}
	eventos, err := h.uc.TrilhaDaRemessa(r.Context(), id, limite)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	saida := make([]auditoriaResponse, 0, len(eventos))
	for _, e := range eventos {
		saida = append(saida, auditoriaResponse{
			ID: e.ID, EntityType: e.EntityType, EntityID: e.EntityID, Action: e.Acao,
			ChangedFields: e.Alterados, Before: json.RawMessage(e.Antes),
			After: json.RawMessage(e.Depois), Reason: e.Motivo, ActorID: e.AtorID,
			ActorName: e.AtorNome, ActorEmail: e.AtorEmail,
			OccurredAt: e.OcorridoEm.Format(time.RFC3339),
		})
	}
	security.RespondJSON(w, http.StatusOK, saida)
}

// usuarioDaSessaoCM devolve quem está assinando a operação. Movimento e
// encerramento gravam o autor: num controle de material que não é da empresa, a
// pergunta "quem deu baixa nisso" precisa ter resposta.
func usuarioDaSessaoCM(w http.ResponseWriter, r *http.Request) (string, bool) {
	if u, ok := r.Context().Value(contextkey.UserKey).(*appsecurity.AuthUser); ok && u != nil && u.ID != "" {
		return u.ID, true
	}
	security.RespondUseCaseError(w, errorsuc.NewValidationError("sessão sem usuário identificado"))
	return "", false
}
