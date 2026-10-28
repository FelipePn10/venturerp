package sales_order_uc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

// ParametrosComerciais: a alçada de desconto da linha do pedido (nil = sem limite).
type ParametrosComerciais interface {
	DescontoMaximo(ctx context.Context) (*decimal.Decimal, error)
}

// PrecosContratados: o preço do contrato de venda da linha de chamada.
type PrecosContratados interface {
	PrecoContratado(ctx context.Context, salesOrderItemCode int64) (*decimal.Decimal, error)
}

type temPolitica interface {
	TemPoliticaAtiva(ctx context.Context, t aprovacao.Tipo) (bool, error)
}

type temAprovada interface {
	AprovadaDoDocumento(ctx context.Context, t aprovacao.Tipo, documento int64) (*aprovacao.Solicitacao, error)
}

// PoliticaDoPedido aplica no pedido de venda as políticas comerciais (as
// mesmas do orçamento: desconto, acréscimo, frete e comissão por cliente,
// segmento, região, condição, tabela, item, faixa e vigência) e a alçada do
// desconto digitado na linha. Política que exige aprovação ou desconto acima da
// alçada bloqueia o pedido e abre a aprovação PEDIDO_POLITICA_COMERCIAL; a
// aprovação conclui o desbloqueio (LiberacaoDeCredito sobre o mesmo pedido).
type PoliticaDoPedido struct {
	Repo       repository.SalesOrderRepository
	Customers  customerrepo.CustomerRepository
	Aprovacoes Aprovacoes
	Parametros ParametrosComerciais
}

// ConferirDesconto antes de gravar a linha: acima da alçada só passa se houver
// quem aprove (política de aprovação ativa); senão a linha é recusada.
func (p *PoliticaDoPedido) ConferirDesconto(ctx context.Context, pct float64) error {
	if p == nil || p.Parametros == nil {
		return nil
	}
	limite, err := p.Parametros.DescontoMaximo(ctx)
	if err != nil || limite == nil || !decimal.NewFromFloat(pct).GreaterThan(*limite) {
		return err
	}
	if t, ok := p.Aprovacoes.(temPolitica); ok {
		if ativa, err := t.TemPoliticaAtiva(ctx, aprovacao.PedidoPoliticaComercial); err != nil || ativa {
			return err
		}
	}
	return errorsuc.NewValidationError(fmt.Sprintf("desconto de %s%% acima da alçada de %s%%: reduza o desconto ou configure a aprovação \"Pedido bloqueado por política comercial\" (VAPR0100)",
		decimal.NewFromFloat(pct).String(), limite.String()))
}

// Aplicar depois de gravar uma linha do pedido: alçada e políticas comerciais.
func (p *PoliticaDoPedido) Aplicar(ctx context.Context, code int64) error {
	return p.aplicar(ctx, code, true)
}

// AplicarAlcada só confere a alçada do desconto — é o que vale para o pedido
// convertido do orçamento, que já passou pelas políticas comerciais (e pela
// aprovação ORCAMENTO_POLITICA_COMERCIAL) no próprio orçamento.
func (p *PoliticaDoPedido) AplicarAlcada(ctx context.Context, code int64) error {
	return p.aplicar(ctx, code, false)
}

func (p *PoliticaDoPedido) aplicar(ctx context.Context, code int64, comPoliticas bool) error {
	if p == nil {
		return nil
	}
	o, err := p.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if o.Status != entity.SalesOrderStatusDraft && o.Status != entity.SalesOrderStatusOrder {
		return nil
	}
	items, err := p.Repo.ListItems(ctx, code)
	if err != nil {
		return err
	}
	exige, motivo := false, ""

	// 1. Alçada do desconto digitado na linha.
	if p.Parametros != nil {
		limite, err := p.Parametros.DescontoMaximo(ctx)
		if err != nil {
			return err
		}
		if limite != nil {
			for _, it := range items {
				if it.Status == entity.SalesOrderItemStatusCancelled {
					continue
				}
				if decimal.NewFromFloat(it.DiscountPct).GreaterThan(*limite) {
					exige = true
					motivo = fmt.Sprintf("desconto de %s%% na linha %d acima da alçada de %s%%", decimal.NewFromFloat(it.DiscountPct).String(), it.Sequence, limite.String())
					break
				}
			}
		}
	}

	// 2. Políticas comerciais.
	if comPoliticas && p.Customers != nil {
		ev, err := p.avaliar(ctx, o, items)
		if err != nil {
			return err
		}
		mudou := false
		if o.DiscountValue.IsZero() && ev.DiscountValue.IsPositive() {
			o.DiscountValue, mudou = ev.DiscountValue.Round(2), true
		}
		if o.SurchargeValue.IsZero() && ev.SurchargeValue.IsPositive() {
			o.SurchargeValue, mudou = ev.SurchargeValue.Round(2), true
		}
		if o.FreightValue.IsZero() && ev.FreightValue.IsPositive() {
			o.FreightValue, mudou = ev.FreightValue.Round(2), true
		}
		if o.CommissionPct == 0 && ev.CommissionValue.IsPositive() && o.TotalGross.IsPositive() {
			// O pedido guarda a comissão como float (coluna legada); o cálculo é decimal.
			o.CommissionPct, mudou = ev.CommissionValue.Div(o.TotalGross).Mul(decimal.NewFromInt(100)).Round(4).InexactFloat64(), true
		}
		if mudou {
			if _, err := p.Repo.Update(ctx, o); err != nil {
				return err
			}
			if err := p.Repo.RecalculateTotals(ctx, code); err != nil {
				return err
			}
		}
		if ev.RequiresApproval && !exige {
			exige, motivo = true, "política comercial exige aprovação"
		}
	}
	if !exige {
		return nil
	}
	if p.Aprovacoes == nil {
		return nil
	}
	if pend, err := p.Aprovacoes.PendenteDoDocumento(ctx, aprovacao.PedidoPoliticaComercial, code); err == nil && pend != nil {
		return nil // já aguardando aprovação
	}
	atual, err := p.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	// A aprovação concedida vale enquanto o motivo é o mesmo e o pedido não
	// cresceu: editar outra linha não bloqueia de novo o pedido aprovado; subir
	// o desconto ou o valor pede nova aprovação.
	if a, ok := p.Aprovacoes.(temAprovada); ok {
		ap, err := a.AprovadaDoDocumento(ctx, aprovacao.PedidoPoliticaComercial, code)
		if err != nil {
			return err
		}
		if ap != nil && ap.Descricao == motivo && atual.TotalGross.Round(2).LessThanOrEqual(ap.Valor) {
			return nil
		}
	}
	s, err := p.Aprovacoes.Solicitar(ctx, aprovacao.PedidoPoliticaComercial, code, atual.TotalGross, motivo)
	if err != nil {
		return fmt.Errorf("abrindo a aprovação do pedido %d: %w", code, err)
	}
	if s == nil {
		// Sem política de aprovação ativa não há quem aprove: o pedido segue (o
		// desconto acima da alçada já foi recusado em ConferirDesconto).
		return nil
	}
	return p.Repo.Block(ctx, code, "aguardando aprovação: "+motivo)
}

func (p *PoliticaDoPedido) avaliar(ctx context.Context, o *entity.SalesOrder, items []*entity.SalesOrderItem) (customerentity.CommercialPolicyEvaluation, error) {
	var zero customerentity.CommercialPolicyEvaluation
	policies, err := p.Customers.ListCommercialPolicies(ctx, true, nil)
	if err != nil {
		return zero, err
	}
	for _, pol := range policies {
		lines, err := p.Customers.ListCommercialPolicyLines(ctx, pol.Code)
		if err != nil {
			return zero, err
		}
		pol.Lines = lines
	}
	quantidade := decimal.Zero
	for _, it := range items {
		if it.Status != entity.SalesOrderItemStatusCancelled {
			quantidade = quantidade.Add(it.Balance)
		}
	}
	pc := customerentity.CommercialPolicyContext{GrossValue: o.TotalGross, Quantity: quantidade, CustomerCode: o.CustomerCode}
	if o.CustomerCode != nil {
		c, err := p.Customers.GetCustomerByCode(ctx, *o.CustomerCode)
		if err != nil {
			return zero, err
		}
		pc.CustomerTypeID, pc.MarketSegmentID, pc.RegionID = c.CustomerTypeID, c.MarketSegmentID, c.RegionID
	}
	if o.PaymentTermCode != nil {
		if cond, err := p.Customers.GetPaymentConditionByCode(ctx, *o.PaymentTermCode); err == nil {
			pc.PaymentConditionID = &cond.ID
		}
	}
	if o.PriceTableCode != nil {
		if t, err := p.Customers.GetSalesTableByCode(ctx, *o.PriceTableCode); err == nil {
			pc.SalesTableID = &t.ID
		}
	}
	if o.CarrierCode != nil {
		if c, err := p.Customers.GetCarrierByCode(ctx, *o.CarrierCode); err == nil {
			pc.CarrierID = &c.ID
		}
	}
	cabeca := make([]*customerentity.CommercialPolicy, 0, len(policies))
	porItem := make([]*customerentity.CommercialPolicy, 0, len(policies))
	for _, pol := range policies {
		if pol.AppliesToItems || pol.ItemCode != nil || pol.ItemMask != nil || pol.ProductLineID != nil || pol.ItemClassification != nil {
			porItem = append(porItem, pol)
		} else {
			cabeca = append(cabeca, pol)
		}
	}
	ev, err := customerentity.EvaluateCommercialPolicies(cabeca, pc)
	if err != nil {
		return zero, err
	}
	if ev == nil {
		ev = &customerentity.CommercialPolicyEvaluation{}
	}
	for _, it := range items {
		if it.Status == entity.SalesOrderItemStatusCancelled {
			continue
		}
		code := strconv.FormatInt(it.ItemCode, 10)
		mask := it.Mask
		ic := pc
		ic.GrossValue, ic.Quantity, ic.ItemCode, ic.ItemMask = it.TotalGross, it.Balance, &code, &mask
		ie, err := customerentity.EvaluateCommercialPolicies(porItem, ic)
		if err != nil {
			return zero, err
		}
		if ie == nil {
			continue
		}
		ev.DiscountValue = ev.DiscountValue.Add(ie.DiscountValue)
		ev.SurchargeValue = ev.SurchargeValue.Add(ie.SurchargeValue)
		ev.FreightValue = ev.FreightValue.Add(ie.FreightValue)
		ev.CommissionValue = ev.CommissionValue.Add(ie.CommissionValue)
		ev.RequiresApproval = ev.RequiresApproval || ie.RequiresApproval
	}
	return *ev, nil
}
