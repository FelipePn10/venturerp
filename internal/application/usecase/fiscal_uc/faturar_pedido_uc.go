package fiscal_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	commissionentity "github.com/FelipePn10/panossoerp/internal/domain/sales_commission/entity"
	commissionrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_commission/repository"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

// FaturarPedidoUseCase emite a NF-e de saída a partir do pedido de venda: o
// cliente, o endereço de entrega, os itens com o preço negociado, o frete, a
// condição de pagamento e o representante (de onde nasce a comissão) vêm do
// pedido — ninguém redigita o que o pedido já diz. O pedido pode ser faturado
// por inteiro ou em partes; cada nota leva só o que ainda falta faturar.
//
// A nota nasce em RASCUNHO: a transmissão continua sendo pela prévia e pela
// autorização, que conferem tudo antes de ir para a SEFAZ.
type FaturarPedidoUseCase struct {
	Create      *CreateFiscalExitUseCase
	Fiscal      repository.FiscalRepository
	Faturamento repository.SalesOrderInvoicingRepository
	Itens       interface {
		ItemsByCode(ctx context.Context, codes []int64) (map[int64]repository.ItemCandidate, error)
	}
	Pedidos   salesrepo.SalesOrderRepository
	Clientes  customerrepo.CustomerRepository
	Comissoes commissionrepo.Repository
	Auth      ports.AuthService
}

// FaturarPedidoDTO é o pedido de faturamento. Itens vazios fatura tudo o que
// está pendente; com itens, só as linhas e quantidades informadas.
type FaturarPedidoDTO struct {
	SalesOrderCode   int64               `json:"sales_order_code"`
	DataEmissao      string              `json:"data_emissao"`
	DataSaida        *string             `json:"data_saida,omitempty"`
	Serie            string              `json:"serie"`
	Cfop             string              `json:"cfop"`
	NaturezaOperacao string              `json:"natureza_operacao"`
	ValorFrete       *float64            `json:"valor_frete,omitempty"`
	ValorSeguro      *float64            `json:"valor_seguro,omitempty"`
	ValorDesconto    *float64            `json:"valor_desconto,omitempty"`
	Itens            []FaturarPedidoItem `json:"itens,omitempty"`
}

type FaturarPedidoItem struct {
	SalesOrderItemCode int64   `json:"sales_order_item_code"`
	Quantidade         float64 `json:"quantidade"`
	Cfop               *string `json:"cfop,omitempty"`
}

type linhaPedido struct {
	item     *salesentity.SalesOrderItem
	cadastro repository.ItemCandidate
	emNota   decimal.Decimal
	pendente decimal.Decimal
	unitario decimal.Decimal
}

// Previa mostra o que o faturamento do pedido vai gerar.
func (uc *FaturarPedidoUseCase) Previa(ctx context.Context, salesOrderCode int64) (*response.PedidoParaFaturarResponse, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	pedido, linhas, err := uc.carregar(ctx, salesOrderCode)
	if err != nil {
		return nil, err
	}

	out := &response.PedidoParaFaturarResponse{
		SalesOrderCode:     pedido.Code,
		OrderNumber:        pedido.OrderNumber,
		Status:             string(pedido.Status),
		EmissionDate:       pedido.EmissionDate.Format("2006-01-02"),
		CustomerCode:       pedido.CustomerCode,
		RepresentativeCode: pedido.RepresentativeCode,
		CommissionPct:      pedido.CommissionPct,
		PaymentTermCode:    pedido.PaymentTermCode,
		FreightType:        pedido.FreightType,
		FreightValue:       pedido.FreightValue,
		InsuranceValue:     pedido.InsuranceValue,
		DiscountValue:      pedido.DiscountValue,
		Itens:              []response.PedidoItemParaFaturar{},
	}

	// Destinatário como a nota vai sair (mesma resolução da emissão).
	dest := request.CreateFiscalExitDTO{CustomerCode: pedido.CustomerCode, SalesOrderCode: &pedido.Code}
	resolverDestinatario(ctx, &dest, uc.Clientes, uc.Pedidos)
	out.CustomerName = deref(dest.RazaoSocialDestinatario)
	out.CustomerDocument = deref(dest.CnpjDestinatario)
	out.CustomerIE = deref(dest.IEDestinatario)
	out.CustomerUF = deref(dest.UFDestinatario)
	out.CustomerCity = deref(dest.DestMunicipio)
	out.EnderecoCompleto = dest.DestLogradouro != nil && dest.DestMunicipio != nil && dest.DestCEP != nil && dest.UFDestinatario != nil

	if pedido.RepresentativeCode != nil {
		nome, err := uc.Faturamento.NomeRepresentante(ctx, *pedido.RepresentativeCode)
		if err != nil {
			return nil, err
		}
		out.RepresentativeName = nome
	}
	if uc.Comissoes != nil {
		rateio, err := uc.Comissoes.Listar(ctx, commissionentity.DocumentoPedido, pedido.Code)
		if err != nil {
			return nil, err
		}
		for _, r := range rateio {
			out.Comissoes = append(out.Comissoes, response.ComissaoDoPedido{
				RepresentativeCode: r.RepresentativeCode,
				RepresentativeName: r.RepresentativeName,
				Papel:              string(r.Role),
				CommissionPct:      r.CommissionPct.InexactFloat64(),
			})
		}
	}
	if pedido.PaymentTermCode != nil && uc.Clientes != nil {
		if c, err := uc.Clientes.GetPaymentConditionByCode(ctx, *pedido.PaymentTermCode); err == nil && c != nil {
			out.PaymentTermDescricao = c.Description
		}
	}

	cfg, err := uc.Fiscal.GetFiscalConfig(ctx)
	var semConfig *errorsuc.NotFoundError
	if err != nil && !errors.As(err, &semConfig) {
		return nil, err
	}
	out.CfopSugerido, out.NaturezaSugerida = "5101", "VENDA DE PRODUCAO DO ESTABELECIMENTO"
	if cfg != nil && out.CustomerUF != "" && !strings.EqualFold(cfg.UFEmpresa, out.CustomerUF) {
		out.CfopSugerido = "6101"
	}

	pendente := decimal.Zero
	for _, l := range linhas {
		valor := l.pendente.Mul(l.unitario).Round(2)
		pendente = pendente.Add(valor)
		out.Itens = append(out.Itens, response.PedidoItemParaFaturar{
			SalesOrderItemCode:  l.item.Code,
			Sequence:            l.item.Sequence,
			ItemCode:            l.item.ItemCode,
			ItemName:            l.cadastro.Name,
			NCM:                 l.cadastro.NCM,
			UOM:                 unidadeDaLinha(l),
			Origem:              l.cadastro.Origem,
			QuantidadePedida:    l.item.RequestedQty,
			QuantidadeCancelada: l.item.CancelledQty,
			QuantidadeFaturada:  l.item.AttendedQty,
			QuantidadeEmNota:    l.emNota.InexactFloat64(),
			QuantidadePendente:  l.pendente.InexactFloat64(),
			PrecoUnitario:       l.unitario.InexactFloat64(),
			DescontoPct:         l.item.DiscountPct,
			IPIPct:              l.item.IPIPct,
			ValorPendente:       valor.InexactFloat64(),
		})
	}
	out.ValorPendente = pendente.InexactFloat64()
	out.Impedimentos = impedimentosDoPedido(pedido, linhas, out.EnderecoCompleto)
	out.PodeFaturar = len(out.Impedimentos) == 0

	notas, err := uc.Faturamento.NotasDoPedido(ctx, pedido.Code)
	if err != nil {
		return nil, err
	}
	for _, n := range notas {
		out.NotasDoPedido = append(out.NotasDoPedido, response.NotaDoPedido{ID: n.ID, NumeroNF: n.NumeroNF, Status: n.Status, ValorTotal: n.ValorTotal.InexactFloat64()})
	}
	return out, nil
}

// Execute cria a NF-e de saída (rascunho) com o que foi pedido para faturar.
func (uc *FaturarPedidoUseCase) Execute(ctx context.Context, dto FaturarPedidoDTO) (*response.FiscalExitResponse, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.SalesOrderCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o pedido de venda")
	}
	cfop := soDigitos(dto.Cfop)
	if len(cfop) != 4 || (cfop[0] != '5' && cfop[0] != '6' && cfop[0] != '7') {
		return nil, errorsuc.NewValidationError("informe um CFOP de saída válido (5xxx, 6xxx ou 7xxx)")
	}
	if strings.TrimSpace(dto.NaturezaOperacao) == "" {
		return nil, errorsuc.NewValidationError("informe a natureza da operação")
	}
	emissao, err := time.Parse("2006-01-02", strings.TrimSpace(dto.DataEmissao))
	if err != nil {
		return nil, errorsuc.NewValidationError("data de emissão inválida: use AAAA-MM-DD")
	}

	// Saldo lido e nota criada sob o mesmo bloqueio: sem ele, dois faturamentos
	// simultâneos veem o mesmo saldo e o pedido sai faturado em dobro.
	liberar, err := uc.Faturamento.TravarPedido(ctx, dto.SalesOrderCode)
	if err != nil {
		return nil, err
	}
	defer liberar()

	pedido, linhas, err := uc.carregar(ctx, dto.SalesOrderCode)
	if err != nil {
		return nil, err
	}
	if imp := impedimentosDoPedido(pedido, linhas, true); len(imp) > 0 {
		return nil, errorsuc.NewValidationError("o pedido não pode ser faturado: " + strings.Join(imp, "; "))
	}

	porCodigo := map[int64]*linhaPedido{}
	for i := range linhas {
		porCodigo[linhas[i].item.Code] = &linhas[i]
	}
	type escolha struct {
		linha *linhaPedido
		qtd   decimal.Decimal
		cfop  string
	}
	var escolhidas []escolha
	if len(dto.Itens) == 0 {
		for i := range linhas {
			if linhas[i].pendente.IsPositive() {
				escolhidas = append(escolhidas, escolha{&linhas[i], linhas[i].pendente, cfop})
			}
		}
	} else {
		vistas := map[int64]bool{}
		for _, it := range dto.Itens {
			l, ok := porCodigo[it.SalesOrderItemCode]
			if !ok {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d não pertence ao pedido %d", it.SalesOrderItemCode, dto.SalesOrderCode))
			}
			if vistas[it.SalesOrderItemCode] {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d foi informada duas vezes", it.SalesOrderItemCode))
			}
			vistas[it.SalesOrderItemCode] = true
			q := decimal.NewFromFloat(it.Quantidade)
			if q.IsZero() {
				continue
			}
			if q.IsNegative() {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d: quantidade negativa", l.item.ItemCode))
			}
			if q.GreaterThan(l.pendente) {
				return nil, errorsuc.NewValidationError(fmt.Sprintf(
					"item %d (%s): quer faturar %s, mas só faltam %s", l.item.ItemCode, l.cadastro.Name, q.String(), l.pendente.String()))
			}
			c := cfop
			if it.Cfop != nil && len(soDigitos(*it.Cfop)) == 4 {
				c = soDigitos(*it.Cfop)
			}
			escolhidas = append(escolhidas, escolha{l, q, c})
		}
	}
	if len(escolhidas) == 0 {
		return nil, errorsuc.NewValidationError("não há quantidade pendente para faturar neste pedido")
	}

	itens := make([]request.CreateFiscalExitItemDTO, 0, len(escolhidas))
	produtos := decimal.Zero
	for i, e := range escolhidas {
		total := e.qtd.Mul(e.linha.unitario).Round(2)
		produtos = produtos.Add(total)
		itemCode := e.linha.item.ItemCode
		linhaCode := e.linha.item.Code
		desc := e.linha.cadastro.Name
		if desc == "" {
			desc = fmt.Sprintf("Item %d", itemCode)
		}
		uom := unidadeDaLinha(*e.linha)
		codigoProduto := fmt.Sprintf("%d", itemCode)
		origem := e.linha.cadastro.Origem
		if origem == "" {
			origem = "0"
		}
		itens = append(itens, request.CreateFiscalExitItemDTO{
			Sequence:           i + 1,
			ItemCode:           &itemCode,
			Ncm:                strPtr(e.linha.cadastro.NCM),
			Cfop:               e.cfop,
			Quantity:           e.qtd.InexactFloat64(),
			UnitPrice:          e.linha.unitario.InexactFloat64(),
			TotalPrice:         total.InexactFloat64(),
			OrigemMercadoria:   origem,
			Description:        &desc,
			SalesOrderItemCode: &linhaCode,
			UnidadeComercial:   strPtr(uom),
			CodigoProduto:      &codigoProduto,
		})
	}

	// Frete, seguro e desconto do pedido entram na proporção do que está sendo
	// faturado (a última nota leva o que sobrar), salvo valor informado.
	proporcao := decimal.NewFromInt(1)
	if totalPedido := decimal.NewFromFloat(pedido.TotalNet); totalPedido.IsPositive() && produtos.LessThan(totalPedido) {
		proporcao = produtos.Div(totalPedido)
	}
	rateado := func(informado *float64, doPedido float64) float64 {
		if informado != nil {
			return *informado
		}
		return decimal.NewFromFloat(doPedido).Mul(proporcao).Round(2).InexactFloat64()
	}

	var dataSaida *string
	if dto.DataSaida != nil && strings.TrimSpace(*dto.DataSaida) != "" {
		dataSaida = dto.DataSaida
	}
	origem := "SALES_ORDER"
	create := request.CreateFiscalExitDTO{
		Serie:            dto.Serie,
		DataEmissao:      emissao.Format("2006-01-02"),
		DataSaida:        dataSaida,
		CustomerCode:     pedido.CustomerCode,
		Cfop:             cfop,
		NaturezaOperacao: strings.ToUpper(strings.TrimSpace(dto.NaturezaOperacao)),
		ValorProdutos:    produtos.InexactFloat64(),
		ValorFrete:       rateado(dto.ValorFrete, pedido.FreightValue),
		ValorSeguro:      rateado(dto.ValorSeguro, pedido.InsuranceValue),
		ValorDesconto:    rateado(dto.ValorDesconto, pedido.DiscountValue),
		SalesOrderCode:   &pedido.Code,
		SourceType:       &origem,
		Itens:            itens,
	}
	return uc.Create.Execute(ctx, create)
}

func (uc *FaturarPedidoUseCase) carregar(ctx context.Context, code int64) (*salesentity.SalesOrder, []linhaPedido, error) {
	if uc.Pedidos == nil || uc.Faturamento == nil || uc.Itens == nil || uc.Create == nil {
		return nil, nil, fmt.Errorf("faturamento do pedido não configurado")
	}
	pedido, err := uc.Pedidos.GetByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if pedido == nil {
		return nil, nil, errorsuc.NewNotFoundError(fmt.Sprintf("pedido de venda %d não encontrado", code))
	}
	itens, err := uc.Pedidos.ListItems(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	emNota, err := uc.Faturamento.QuantidadesEmNota(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	codigos := make([]int64, 0, len(itens))
	for _, it := range itens {
		codigos = append(codigos, it.ItemCode)
	}
	cadastro, err := uc.Itens.ItemsByCode(ctx, codigos)
	if err != nil {
		return nil, nil, err
	}
	linhas := make([]linhaPedido, 0, len(itens))
	for _, it := range itens {
		if !it.IsActive || it.Status == salesentity.SalesOrderItemStatusCancelled {
			continue
		}
		pend := decimal.NewFromFloat(it.RequestedQty).Sub(decimal.NewFromFloat(it.CancelledQty)).
			Sub(decimal.NewFromFloat(it.AttendedQty)).Sub(emNota[it.Code])
		if pend.IsNegative() {
			pend = decimal.Zero
		}
		linhas = append(linhas, linhaPedido{
			item:     it,
			cadastro: cadastro[it.ItemCode],
			emNota:   emNota[it.Code],
			pendente: pend,
			unitario: precoLiquido(it),
		})
	}
	return pedido, linhas, nil
}

// precoLiquido é o preço unitário negociado já com o desconto do item. O
// total líquido da linha é a fonte mais fiel (é o que o pedido somou); na
// falta dele, aplica o percentual de desconto sobre o preço.
func precoLiquido(it *salesentity.SalesOrderItem) decimal.Decimal {
	qtd := decimal.NewFromFloat(it.RequestedQty)
	if tn := decimal.NewFromFloat(it.TotalNet); tn.IsPositive() && qtd.IsPositive() {
		return tn.Div(qtd).Round(4)
	}
	preco := decimal.NewFromFloat(it.UnitPrice)
	if it.DiscountPct > 0 {
		preco = preco.Mul(decimal.NewFromInt(1).Sub(decimal.NewFromFloat(it.DiscountPct).Div(cemDec)))
	}
	return preco.Round(4)
}

func unidadeDaLinha(l linhaPedido) string {
	if l.item.SalesUOM != nil && strings.TrimSpace(*l.item.SalesUOM) != "" {
		return strings.ToUpper(strings.TrimSpace(*l.item.SalesUOM))
	}
	return strings.ToUpper(l.cadastro.UOM)
}

// impedimentosDoPedido diz por que o pedido não pode virar nota agora.
func impedimentosDoPedido(p *salesentity.SalesOrder, linhas []linhaPedido, enderecoCompleto bool) []string {
	var out []string
	switch p.Status {
	case salesentity.SalesOrderStatusCancelled:
		out = append(out, "o pedido está cancelado")
	case salesentity.SalesOrderStatusDraft, salesentity.SalesOrderStatusBudget, salesentity.SalesOrderStatusBudgetAnalysis:
		out = append(out, "o pedido ainda é rascunho/orçamento: confirme-o como pedido antes de faturar")
	case salesentity.SalesOrderStatusAnalysis:
		out = append(out, "o pedido está em análise: aprove-o antes de faturar")
	case salesentity.SalesOrderStatusInvoiced:
		out = append(out, "o pedido já foi faturado por inteiro")
	}
	if !p.IsActive {
		out = append(out, "o pedido está inativo")
	}
	if p.IsBlocked {
		motivo := ""
		if p.BlockReason != nil {
			motivo = ": " + *p.BlockReason
		}
		out = append(out, "o pedido está bloqueado"+motivo)
	}
	if p.ReleaseStatus == salesentity.SalesOrderReleaseBlocked {
		out = append(out, "o pedido não foi liberado (crédito/comercial)")
	}
	if p.CustomerCode == nil {
		out = append(out, "o pedido não tem cliente")
	} else if !enderecoCompleto {
		out = append(out, "o cadastro do cliente não tem endereço completo (logradouro, município, CEP e UF): a SEFAZ recusa a nota")
	}
	temPendente := false
	for _, l := range linhas {
		if l.pendente.IsPositive() {
			temPendente = true
		}
		if l.pendente.IsPositive() && l.unitario.Sign() <= 0 {
			out = append(out, fmt.Sprintf("o item %d está sem preço no pedido", l.item.ItemCode))
		}
		if l.pendente.IsPositive() && soDigitos(l.cadastro.NCM) == "" {
			out = append(out, fmt.Sprintf("o item %d (%s) está sem NCM na classificação fiscal do cadastro", l.item.ItemCode, l.cadastro.Name))
		}
	}
	if !temPendente && p.Status != salesentity.SalesOrderStatusInvoiced {
		out = append(out, "não há quantidade pendente: tudo já foi faturado ou está em nota não autorizada")
	}
	return out
}
