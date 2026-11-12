package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
)

func d(v string) decimal.Decimal { return decimal.RequireFromString(v) }

func liquidoComHorizonte(t *testing.T, dias int, input *entity.MRPInput, firmes map[int64][]ports.SupplyEntry) *entity.MRPOutput {
	t.Helper()
	params := entity.DefaultTypedPlanningParams()
	params.HorizonteReprogramacaoDias = dias
	out, err := (&MRPServiceImpl{}).calcNetReqFast(context.Background(), input, map[int64]*entity.StockSnapshot{},
		map[int64][]*entity.ConfiguredItemRule{}, firmes, map[int64]*cachedItemMRP{input.ItemCode: {engineeringType: 1}}, params)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func compra(code int64, qtd string, chegada time.Time) ports.SupplyEntry {
	return ports.SupplyEntry{ItemCode: 7, Quantity: d(qtd), ArrivalDate: chegada, SourceType: ports.SupplySourcePlannedOrder, SourceCode: code}
}

// O pedido de compra de 100 chega 10 dias depois da necessidade de 100. Com o
// horizonte ligado ele cobre a necessidade (o planejador antecipa); antes o MRP
// sugeria outras 100 e mandava antecipar o pedido — compra em dobro.
func TestHorizonteDeReprogramacaoUsaAFirmeAtrasadaEmVezDeSugerirOutra(t *testing.T) {
	hoje := today()
	necessidade, chegada := hoje.AddDate(0, 0, 5), hoje.AddDate(0, 0, 15)
	firmes := map[int64][]ports.SupplyEntry{7: {compra(1, "100", chegada)}}

	out := liquidoComHorizonte(t, 30, necessidadeDe(7, "100", necessidade), firmes)
	if len(out.PlannedOrders) != 0 {
		t.Fatalf("gerou sugestão de %s apesar da firme dentro do horizonte", out.PlannedOrders[0].Quantity)
	}
	if !out.FirmeAntecipada.Equal(d("100")) || !out.NetRequirement.IsZero() {
		t.Fatalf("antecipada=%s líquida=%s, esperado 100 e 0", out.FirmeAntecipada, out.NetRequirement)
	}
	if len(firmes[7]) != 0 {
		t.Fatalf("a firme consumida por inteiro continuou disponível: %+v", firmes[7])
	}
}

// Desligado (0), o comportamento anterior é mantido: a firme atrasada não cobre.
func TestHorizonteDeReprogramacaoDesligadoMantemASugestao(t *testing.T) {
	hoje := today()
	firmes := map[int64][]ports.SupplyEntry{7: {compra(1, "100", hoje.AddDate(0, 0, 15))}}
	out := liquidoComHorizonte(t, 0, necessidadeDe(7, "100", hoje.AddDate(0, 0, 5)), firmes)
	if len(out.PlannedOrders) != 1 || !out.PlannedOrders[0].Quantity.Equal(d("100")) {
		t.Fatalf("sugestões = %+v, esperado uma de 100", out.PlannedOrders)
	}
	if !firmes[7][0].Quantity.Equal(d("100")) {
		t.Fatalf("a firme foi consumida com o horizonte desligado: %s", firmes[7][0].Quantity)
	}
}

// A firme que chega depois do fim do horizonte está longe demais para ser
// antecipada: a falta vira sugestão.
func TestHorizonteDeReprogramacaoIgnoraAFirmeForaDoHorizonte(t *testing.T) {
	hoje := today()
	firmes := map[int64][]ports.SupplyEntry{7: {compra(1, "100", hoje.AddDate(0, 0, 45))}}
	out := liquidoComHorizonte(t, 30, necessidadeDe(7, "100", hoje.AddDate(0, 0, 5)), firmes)
	if len(out.PlannedOrders) != 1 || !out.FirmeAntecipada.IsZero() {
		t.Fatalf("sugestões=%d antecipada=%s, esperado 1 e 0", len(out.PlannedOrders), out.FirmeAntecipada)
	}
}

// A firme cobre parte; a sugestão sai só pelo que falta, e o que sobra da firme
// maior que a necessidade continua para a próxima.
func TestHorizonteDeReprogramacaoCobreParteESugereOResto(t *testing.T) {
	hoje := today()
	necessidade := hoje.AddDate(0, 0, 5)

	parcial := map[int64][]ports.SupplyEntry{7: {compra(1, "60", hoje.AddDate(0, 0, 12))}}
	out := liquidoComHorizonte(t, 30, necessidadeDe(7, "100", necessidade), parcial)
	if len(out.PlannedOrders) != 1 || !out.PlannedOrders[0].Quantity.Equal(d("40")) {
		t.Fatalf("sugestões = %+v, esperado uma de 40", out.PlannedOrders)
	}

	sobra := map[int64][]ports.SupplyEntry{7: {compra(2, "250", hoje.AddDate(0, 0, 12))}}
	out = liquidoComHorizonte(t, 30, necessidadeDe(7, "100", necessidade), sobra)
	if len(out.PlannedOrders) != 0 {
		t.Fatalf("gerou sugestão com firme sobrando: %+v", out.PlannedOrders)
	}
	if len(sobra[7]) != 1 || !sobra[7][0].Quantity.Equal(d("150")) {
		t.Fatalf("sobra da firme = %+v, esperado 150", sobra[7])
	}
}

// Usa primeiro a que chega mais cedo, e não toca na que já chega no prazo (essa
// é somada pelo netting normal) nem na que está fora do horizonte.
func TestAnteciparFirmesPegaAMaisProximaPrimeiro(t *testing.T) {
	base := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	necessidade, limite := base.AddDate(0, 0, 5), base.AddDate(0, 0, 30)
	firmes := map[int64][]ports.SupplyEntry{7: {
		compra(1, "50", base.AddDate(0, 0, 20)), // dentro, mais tarde
		compra(2, "30", base.AddDate(0, 0, 3)),  // no prazo: não é "atrasada"
		compra(3, "40", base.AddDate(0, 0, 8)),  // dentro, mais cedo
		compra(4, "90", base.AddDate(0, 0, 60)), // fora do horizonte
	}}
	coberto := anteciparFirmes(firmes, 7, necessidade, limite, d("60"))
	if !coberto.Equal(d("60")) {
		t.Fatalf("coberto = %s, esperado 60", coberto)
	}
	resto := map[int64]string{}
	for _, e := range firmes[7] {
		resto[e.SourceCode] = e.Quantity.String()
	}
	// 40 da ordem 3 (consumida por inteiro, sai) + 20 da ordem 1 (sobram 30).
	if len(resto) != 3 || resto[1] != "30" || resto[2] != "30" || resto[4] != "90" {
		t.Fatalf("restante = %v", resto)
	}
	if got := anteciparFirmes(firmes, 7, necessidade, limite, decimal.Zero); !got.IsZero() {
		t.Fatalf("sem falta não há o que antecipar: %s", got)
	}
}

func TestParametro51CarregaOHorizonteDeReprogramacao(t *testing.T) {
	var p entity.TypedPlanningParams
	p.LoadFromDB(map[int]string{51: "21"})
	if p.HorizonteReprogramacaoDias != 21 {
		t.Fatalf("horizonte = %d, esperado 21", p.HorizonteReprogramacaoDias)
	}
	p.LoadFromDB(map[int]string{51: "-5"})
	if p.HorizonteReprogramacaoDias != 0 {
		t.Fatalf("valor negativo virou %d, esperado 0", p.HorizonteReprogramacaoDias)
	}
	// Sem o parâmetro gravado vale o padrão: ligado, com 30 dias.
	p.LoadFromDB(map[int]string{})
	if p.HorizonteReprogramacaoDias != 30 {
		t.Fatalf("padrão = %d, esperado 30", p.HorizonteReprogramacaoDias)
	}
	// Zero é escolha explícita de desligar.
	p.LoadFromDB(map[int]string{51: "0"})
	if p.HorizonteReprogramacaoDias != 0 {
		t.Fatalf("0 deveria desligar, veio %d", p.HorizonteReprogramacaoDias)
	}
}
