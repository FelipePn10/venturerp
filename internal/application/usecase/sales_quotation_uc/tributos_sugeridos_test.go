package sales_quotation_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

type sugestorOrcamento struct{ chamadas int }

func (s *sugestorOrcamento) SugerirTributos(context.Context, ports.ConsultaTributosVenda) (*ports.TributosSugeridos, error) {
	s.chamadas++
	return &ports.TributosSugeridos{IPIPct: decimal.NewFromInt(5), STPct: decimal.RequireFromString("3.5")}, nil
}

func TestOrcamentoPreencheIPIESTPelaRegra(t *testing.T) {
	cliente := int64(7)
	q := &entity.SalesQuotation{CustomerCode: &cliente}
	s := &sugestorOrcamento{}
	uc := &UseCase{Tributos: s}
	it := &entity.SalesQuotationItem{RequestedQty: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100)}
	if err := uc.preencherTributos(context.Background(), q, 1, it, false); err != nil {
		t.Fatal(err)
	}
	if !it.IPIPct.Equal(decimal.NewFromInt(5)) || !it.STPct.Equal(decimal.RequireFromString("3.5")) {
		t.Fatalf("IPI %s ST %s", it.IPIPct, it.STPct)
	}
	// IPI digitado, manual ou orçamento sem cliente: não recalcula.
	digitado := &entity.SalesQuotationItem{IPIPct: decimal.NewFromInt(2)}
	_ = uc.preencherTributos(context.Background(), q, 1, digitado, false)
	manual := &entity.SalesQuotationItem{}
	_ = uc.preencherTributos(context.Background(), q, 1, manual, true)
	semCliente := &entity.SalesQuotationItem{}
	_ = uc.preencherTributos(context.Background(), &entity.SalesQuotation{}, 1, semCliente, false)
	if s.chamadas != 1 || !digitado.IPIPct.Equal(decimal.NewFromInt(2)) || !manual.IPIPct.IsZero() || !semCliente.IPIPct.IsZero() {
		t.Fatalf("chamadas %d digitado %s manual %s sem cliente %s", s.chamadas, digitado.IPIPct, manual.IPIPct, semCliente.IPIPct)
	}
}
