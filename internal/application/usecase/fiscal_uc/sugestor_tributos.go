package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"

	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

// SugestorTributos calcula os impostos de uma linha de orçamento/pedido pelo
// Tributador — o número que o cliente vê na proposta é o que a nota vai ter.
type SugestorTributos struct {
	Tributador *Tributador
	Customers  customerrepo.CustomerRepository
	Orders     salesrepo.SalesOrderRepository
	Itens      interface {
		ItemsByCode(ctx context.Context, codes []int64) (map[int64]repository.ItemCandidate, error)
	}
}

var _ ports.SugestorTributosVenda = (*SugestorTributos)(nil)

// SugerirTributos devolve nil (sem erro) quando não há como calcular — cliente
// sem endereço, item sem NCM: a linha fica como o vendedor digitou.
func (s *SugestorTributos) SugerirTributos(ctx context.Context, q ports.ConsultaTributosVenda) (*ports.TributosSugeridos, error) {
	if s == nil || s.Tributador == nil || !q.Quantidade.IsPositive() {
		return nil, nil
	}
	dest := request.CreateFiscalExitDTO{CustomerCode: q.CustomerCode, SalesOrderCode: q.SalesOrderCode}
	if err := resolverDestinatario(ctx, &dest, s.Customers, s.Orders); err != nil {
		return nil, err
	}
	uf := strings.ToUpper(strings.TrimSpace(derefStr(dest.UFDestinatario)))
	if uf == "" {
		return nil, nil
	}
	ncm, origem := "", "0"
	if s.Itens != nil {
		cad, err := s.Itens.ItemsByCode(ctx, []int64{q.ItemCode})
		if err != nil {
			return nil, err
		}
		if c, ok := cad[q.ItemCode]; ok {
			ncm = c.NCM
			if strings.TrimSpace(c.Origem) != "" {
				origem = c.Origem
			}
		}
	}
	if strings.TrimSpace(ncm) == "" {
		return nil, nil
	}
	item := q.ItemCode
	op := OperacaoTributavel{
		UFDestino: uf, IE: dest.IEDestinatario, TipoPessoa: dest.TipoPessoa, Cliente: dest.CustomerCode, Pedido: q.SalesOrderCode,
		Itens: []LinhaTributavel{{ItemCode: &item, Ncm: &ncm, Quantidade: q.Quantidade.InexactFloat64(), ValorUnitario: q.ValorUnitario.InexactFloat64(), Origem: origem}},
	}
	if q.Finalidade != nil {
		op.Finalidade = *q.Finalidade
	}
	t, err := s.Tributador.Calcular(ctx, op)
	if err != nil {
		// Empresa sem configuração fiscal ainda não tributa: a linha fica como
		// o vendedor digitou, e o pedido segue.
		if errorsuc.EhNaoEncontrado(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("calculando os impostos sugeridos da linha: %w", err)
	}
	r := t.Resultado.Itens[0]
	cem := decimal.NewFromInt(100)
	out := &ports.TributosSugeridos{
		IPIPct:    decimal.NewFromFloat(r.AliquotaIPI).Mul(cem).Round(4),
		ICMSPct:   decimal.NewFromFloat(r.AliquotaICMS).Mul(cem).Round(4),
		PISPct:    decimal.NewFromFloat(r.AliquotaPIS).Mul(cem).Round(4),
		COFINSPct: decimal.NewFromFloat(r.AliquotaCOFINS).Mul(cem).Round(4),
	}
	if produtos := q.Quantidade.Mul(q.ValorUnitario); produtos.IsPositive() && r.ValorICMSST > 0 {
		out.STPct = decimal.NewFromFloat(r.ValorICMSST).Div(produtos).Mul(cem).Round(4)
	}
	if p := t.Regras[0]; p != nil {
		id := p.ID
		out.RegraICMSIPI = &id
	}
	return out, nil
}
