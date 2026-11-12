package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
)

func necessidadeDe(item int64, qtd string, data time.Time) *entity.MRPInput {
	return &entity.MRPInput{PlanCode: 1, ItemCode: item, Quantity: decimal.RequireFromString(qtd), NeedDate: data}
}

func liquido(t *testing.T, input *entity.MRPInput, estoque map[int64]*entity.StockSnapshot, firmes map[int64][]ports.SupplyEntry, item *cachedItemMRP) *entity.MRPOutput {
	t.Helper()
	out, err := (&MRPServiceImpl{}).calcNetReqFast(context.Background(), input, estoque,
		map[int64][]*entity.ConfiguredItemRule{}, firmes, map[int64]*cachedItemMRP{input.ItemCode: item},
		entity.DefaultTypedPlanningParams())
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// O pedido de compra guarda 4 casas e a explosão calcula em 6: a compra de
// 52.593,3173 cobre a necessidade de 52.593,317345. A diferença (0,000045) não
// é falta — antes virava uma sugestão de compra de 0,0000 a cada recálculo.
func TestSobraDoArredondamentoNaoViraSugestao(t *testing.T) {
	dia := time.Date(2026, 12, 18, 0, 0, 0, 0, time.UTC)
	firmes := map[int64][]ports.SupplyEntry{
		20016: {{Quantity: decimal.RequireFromString("52593.3173"), ArrivalDate: dia, SourceType: ports.SupplySourcePlannedOrder, SourceCode: 2}},
	}
	out := liquido(t, necessidadeDe(20016, "52593.317345", dia), map[int64]*entity.StockSnapshot{}, firmes, &cachedItemMRP{engineeringType: 1})
	if len(out.PlannedOrders) != 0 {
		t.Fatalf("resíduo de arredondamento gerou sugestão de %s", out.PlannedOrders[0].Quantity)
	}
	if !out.NetRequirement.IsZero() {
		t.Fatalf("necessidade líquida = %s, esperado zero", out.NetRequirement)
	}
}

// A falta de verdade, mesmo pequena, continua virando ordem.
func TestFaltaNaEscalaDaOrdemViraSugestao(t *testing.T) {
	dia := time.Date(2026, 12, 18, 0, 0, 0, 0, time.UTC)
	out := liquido(t, necessidadeDe(20016, "0.0001", dia), map[int64]*entity.StockSnapshot{}, nil, &cachedItemMRP{engineeringType: 1})
	if len(out.PlannedOrders) != 1 || !out.PlannedOrders[0].Quantity.Equal(decimal.RequireFromString("0.0001")) {
		t.Fatalf("sugestões = %+v", out.PlannedOrders)
	}
}

// A sugestão é gravada com 4 casas: arredondar para baixo cobriria menos que a
// necessidade e o recálculo seguinte acusaria a diferença.
func TestSugestaoArredondaParaCimaNaEscalaGravada(t *testing.T) {
	dia := time.Date(2026, 12, 18, 0, 0, 0, 0, time.UTC)
	out := liquido(t, necessidadeDe(20016, "52593.317345", dia), map[int64]*entity.StockSnapshot{}, nil, &cachedItemMRP{engineeringType: 1})
	if len(out.PlannedOrders) != 1 {
		t.Fatalf("sugestões = %+v", out.PlannedOrders)
	}
	if got := out.PlannedOrders[0].Quantity; !got.Equal(decimal.RequireFromString("52593.3174")) {
		t.Fatalf("quantidade = %s, esperado 52593.3174", got)
	}
}

// Item com necessidade que o cadastro tira da geração de ordens diz por quê, e
// o perfil não mostra como "planejado" o que não tem sugestão.
func TestItemComFaltaForaDaGeracaoDeOrdensExplicaOMotivo(t *testing.T) {
	dia := time.Date(2026, 12, 18, 0, 0, 0, 0, time.UTC)
	casos := []struct {
		nome   string
		item   *cachedItemMRP
		trecho string
	}{
		{"de terceiro", &cachedItemMRP{engineeringType: 2}, "item de terceiro"},
		{"projeto", &cachedItemMRP{typeMRP: 1}, "planejado por projeto"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			out := liquido(t, necessidadeDe(10002, "100", dia), map[int64]*entity.StockSnapshot{}, nil, c.item)
			if len(out.PlannedOrders) != 0 {
				t.Fatalf("gerou sugestão: %+v", out.PlannedOrders)
			}
			if !strings.Contains(out.NaoPlanejado, c.trecho) {
				t.Fatalf("motivo = %q, esperado conter %q", out.NaoPlanejado, c.trecho)
			}
			if got := ordensPlanejadasDoPerfil(out); !got.IsZero() {
				t.Fatalf("perfil mostra %s planejado sem sugestão", got)
			}
		})
	}
	// O item normal segue mostrando o planejado.
	out := liquido(t, necessidadeDe(10002, "100", dia), map[int64]*entity.StockSnapshot{}, nil, &cachedItemMRP{})
	if out.NaoPlanejado != "" || !ordensPlanejadasDoPerfil(out).Equal(decimal.NewFromInt(100)) {
		t.Fatalf("item normal: motivo=%q planejado=%s", out.NaoPlanejado, ordensPlanejadasDoPerfil(out))
	}
}

func TestMensagemDeItemNaoPlanejadoSomaAFaltaDoItem(t *testing.T) {
	b := newExceptionBasis(nil, nil)
	b.labels = map[int64]string{10002: "SUP-002"}
	b.addNotPlanned(10002, diaOut(20), decimal.NewFromInt(60), "é item de terceiro")
	b.addNotPlanned(10002, diaOut(5), decimal.NewFromInt(40), "é item de terceiro")
	b.addNotPlanned(10003, diaOut(9), decimal.Zero, "é item de terceiro") // sem falta: não entra

	msgs := b.notPlannedMessages()
	if len(msgs) != 1 {
		t.Fatalf("mensagens = %d, esperado 1", len(msgs))
	}
	m := msgs[0]
	if m.MessageType != entity.ExceptionNotPlanned || m.ItemCode != 10002 {
		t.Fatalf("mensagem = %+v", m)
	}
	if m.SuggestedDate == nil || !m.SuggestedDate.Equal(diaOut(5)) {
		t.Fatalf("data = %v, esperado a primeira necessidade", m.SuggestedDate)
	}
	for _, trecho := range []string{"SUP-002", "100 un.", "05/10/2026", "item de terceiro"} {
		if !strings.Contains(m.Description, trecho) {
			t.Fatalf("descrição %q não contém %q", m.Description, trecho)
		}
	}
	if itens := b.itens(); len(itens) != 1 || itens[0] != 10002 {
		t.Fatalf("itens para rótulo = %v", itens)
	}
}

func TestMensagemDeInicioNoPassado(t *testing.T) {
	hoje := diaOut(20)
	sugestao := func(code int64, inicio, chegada time.Time) *entity.PlannedOrderSuggestion {
		return &entity.PlannedOrderSuggestion{Code: code, ItemCode: 7, Quantity: decimal.NewFromInt(10),
			StartDate: &inicio, NeedDate: chegada, OrderType: "FABRICACAO"}
	}
	b := newExceptionBasis(nil, nil)
	b.addPastStart(sugestao(1, diaOut(15), diaOut(22)), hoje) // 5 dias atrás
	b.addPastStart(sugestao(2, diaOut(20), diaOut(25)), hoje) // começa hoje: no prazo
	b.addPastStart(sugestao(3, diaOut(21), diaOut(26)), hoje) // futuro
	// Empurrada pelo horizonte firme: já tem a mensagem dela.
	fence := sugestao(4, diaOut(10), diaOut(12))
	pedida := diaOut(8)
	fence.InicioPedido = &pedida
	b.addPastStart(fence, hoje)
	// Ordem de serviço acompanha a de fabricação: não duplica o aviso.
	servico := sugestao(5, diaOut(10), diaOut(12))
	servico.OrderType = "SERVICO"
	b.addPastStart(servico, hoje)

	msgs := b.pastStartMessages()
	if len(msgs) != 1 {
		t.Fatalf("mensagens = %d, esperado 1", len(msgs))
	}
	m := msgs[0]
	if m.MessageType != entity.ExceptionPastStart || m.SourceCode == nil || *m.SourceCode != 1 {
		t.Fatalf("mensagem = %+v", m)
	}
	// Começando hoje, com o mesmo prazo, chega 5 dias depois do pedido.
	if m.SuggestedDate == nil || !m.SuggestedDate.Equal(diaOut(27)) {
		t.Fatalf("data possível = %v, esperado 27/10", m.SuggestedDate)
	}
	for _, trecho := range []string{"15/10/2026", "5 dia(s) atrás", "22/10/2026", "27/10/2026"} {
		if !strings.Contains(m.Description, trecho) {
			t.Fatalf("descrição %q não contém %q", m.Description, trecho)
		}
	}
}
