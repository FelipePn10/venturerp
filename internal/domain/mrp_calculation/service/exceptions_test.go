package service

import (
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
)

func diaOut(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }

func firme(code int64, qtd float64, chegada time.Time) ports.SupplyEntry {
	return ports.SupplyEntry{Quantity: qtd, ArrivalDate: chegada, SourceType: ports.SupplySourcePlannedOrder, SourceCode: code}
}

// Cada ordem firme é comparada com a necessidade que ela cobre, depois do
// estoque — não com a primeira necessidade do item nem com o que sobrou do
// netting.
func TestExcecoesPorPegging(t *testing.T) {
	b := newExceptionBasis(
		map[int64]*entity.StockSnapshot{1: {Quantity: 10}},
		map[int64][]ports.SupplyEntry{
			// Item 1: estoque 10 cobre parte do dia 5; a ordem 100 cobre o
			// resto do dia 5 mas chega dia 12 (7 d → antecipar); a 101 cobre o
			// dia 20 e chega no dia 20 (no prazo, sem mensagem).
			1: {firme(100, 5, diaOut(12)), firme(101, 30, diaOut(20))},
			// Item 2: a 200 chega 3 dias depois da necessidade (acelerar); a
			// 201 não tem necessidade para cobrir (cancelar).
			2: {firme(200, 8, diaOut(9)), firme(201, 4, diaOut(25))},
			// Item 3: chega 40 dias antes (postergar) e com sobra (excesso).
			3: {firme(300, 100, diaOut(1).AddDate(0, 0, -40))},
		},
	)
	b.addRequirement(1, diaOut(5), 15, "", nil)
	b.addRequirement(1, diaOut(20), 30, "", nil)
	b.addRequirement(2, diaOut(6), 8, "", nil)
	b.addRequirement(3, diaOut(1), 20, "", nil)

	por := map[int64]*entity.MRPExceptionMessage{}
	excesso := 0
	for _, m := range b.messages() {
		if m.MessageType == entity.ExceptionExcess {
			excesso++
			if m.ItemCode != 3 {
				t.Fatalf("excesso no item errado: %+v", m)
			}
			continue
		}
		por[*m.SourceCode] = m
	}
	if m := por[100]; m == nil || m.MessageType != entity.ExceptionRescheduleIn || !m.SuggestedDate.Equal(diaOut(5)) {
		t.Fatalf("ordem 100: %+v", m)
	}
	if m := por[101]; m != nil {
		t.Fatalf("ordem 101 no prazo gerou mensagem: %+v", m)
	}
	if m := por[200]; m == nil || m.MessageType != entity.ExceptionExpedite || !m.SuggestedDate.Equal(diaOut(6)) {
		t.Fatalf("ordem 200: %+v", m)
	}
	if m := por[201]; m == nil || m.MessageType != entity.ExceptionCancel {
		t.Fatalf("ordem 201: %+v", m)
	}
	if m := por[300]; m == nil || m.MessageType != entity.ExceptionRescheduleOut {
		t.Fatalf("ordem 300: %+v", m)
	}
	if excesso != 1 {
		t.Fatalf("excesso = %d, esperado 1", excesso)
	}
}

// O netting consome o mapa de suprimento; a base de exceções guarda a cópia.
func TestBaseDeExcecoesNaoEnxergaONetting(t *testing.T) {
	supply := map[int64][]ports.SupplyEntry{1: {firme(1, 5, diaOut(12))}}
	b := newExceptionBasis(nil, supply)
	supply[1] = supply[1][:0]
	b.addRequirement(1, diaOut(5), 5, "", nil)
	if len(b.messages()) != 1 {
		t.Fatal("a ordem atrasada sumiu porque o netting esvaziou o mapa")
	}
}

// A mensagem usa o código de negócio do item e a quantidade como o usuário
// lê (80 e 12,5); item sem código de negócio cai no código interno.
func TestExcecaoComCodigoDeNegocio(t *testing.T) {
	b := newExceptionBasis(nil, map[int64][]ports.SupplyEntry{
		9003: {firme(1, 80, diaOut(25))},
		9004: {firme(2, 12.5, diaOut(25))},
	})
	b.labels = map[int64]string{9003: "RODA-200"}
	b.addRequirement(9003, diaOut(18), 80, "", nil)
	b.addRequirement(9004, diaOut(18), 12.5, "", nil)
	textos := map[int64]string{}
	for _, m := range b.messages() {
		textos[m.ItemCode] = m.Description
	}
	if d := textos[9003]; !strings.HasPrefix(d, "RODA-200: ordem firme de 80 un.") || strings.Contains(d, "9003") {
		t.Fatalf("mensagem com rótulo: %q", d)
	}
	if d := textos[9004]; !strings.HasPrefix(d, "Item 9004: ordem firme de 12,5 un.") {
		t.Fatalf("mensagem sem rótulo: %q", d)
	}
}
