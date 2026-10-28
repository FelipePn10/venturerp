package sales_order_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

type sugestorFixo struct {
	chamadas int
	ultima   ports.ConsultaTributosVenda
}

func (s *sugestorFixo) SugerirTributos(_ context.Context, q ports.ConsultaTributosVenda) (*ports.TributosSugeridos, error) {
	s.chamadas++
	s.ultima = q
	return &ports.TributosSugeridos{IPIPct: decimal.NewFromInt(5), ICMSPct: decimal.RequireFromString("19.5"),
		PISPct: decimal.RequireFromString("1.65"), COFINSPct: decimal.RequireFromString("7.6"), STPct: decimal.RequireFromString("4.2")}, nil
}

func TestPreencherTributos(t *testing.T) {
	ctx := context.Background()
	linha := func() (percentuaisDaLinha, *[5]float64) {
		v := &[5]float64{}
		return percentuaisDaLinha{IPI: &v[0], ICMS: &v[1], PIS: &v[2], COFINS: &v[3], ST: &v[4]}, v
	}

	// Sem nada informado: calcula pela regra, sobre o preço com desconto.
	s := &sugestorFixo{}
	p, v := linha()
	if err := preencherTributos(ctx, s, false, 10, 99, decimal.NewFromInt(2), decimal.NewFromInt(100), 10, p); err != nil {
		t.Fatal(err)
	}
	if *v != [5]float64{5, 19.5, 1.65, 7.6, 4.2} || !s.ultima.ValorUnitario.Equal(decimal.NewFromInt(90)) || *s.ultima.SalesOrderCode != 10 {
		t.Fatalf("calculado: %v consulta %+v", *v, s.ultima)
	}

	// Percentual digitado não é sobrescrito.
	s = &sugestorFixo{}
	p, v = linha()
	v[0] = 3
	_ = preencherTributos(ctx, s, false, 10, 99, decimal.NewFromInt(1), decimal.NewFromInt(100), 0, p)
	if s.chamadas != 0 || v[0] != 3 || v[1] != 0 {
		t.Fatalf("digitado: %v, chamadas %d", *v, s.chamadas)
	}

	// Manual, mesmo zerado, fica zerado (venda sem imposto por decisão do vendedor).
	s = &sugestorFixo{}
	p, v = linha()
	_ = preencherTributos(ctx, s, true, 10, 99, decimal.NewFromInt(1), decimal.NewFromInt(100), 0, p)
	if s.chamadas != 0 || *v != [5]float64{} {
		t.Fatalf("manual: %v, chamadas %d", *v, s.chamadas)
	}

	// Sem sugestor configurado, nada muda.
	p, v = linha()
	if err := preencherTributos(ctx, nil, false, 10, 99, decimal.NewFromInt(1), decimal.NewFromInt(100), 0, p); err != nil || *v != [5]float64{} {
		t.Fatalf("sem sugestor: %v %v", *v, err)
	}
}

func TestNormalizarFinalidade(t *testing.T) {
	if f, err := normalizarFinalidade(strPtrTeste(" uso_consumo ")); err != nil || *f != "USO_CONSUMO" {
		t.Fatalf("%v %v", f, err)
	}
	if f, err := normalizarFinalidade(strPtrTeste("")); err != nil || *f != "" {
		t.Fatalf("vazio limpa: %v %v", f, err)
	}
	if _, err := normalizarFinalidade(strPtrTeste("REVENDER")); err == nil {
		t.Fatal("finalidade inválida tem de ser recusada")
	}
	if f, err := normalizarFinalidade(nil); err != nil || f != nil {
		t.Fatalf("nil: %v %v", f, err)
	}
}

func strPtrTeste(s string) *string { return &s }
