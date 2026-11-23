package service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestAtenderPeloQueExiste(t *testing.T) {
	d := decimal.NewFromInt
	hoje := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC) // segunda
	s := SituacaoDoItem{Livre: d(10),
		Entradas:     []Movimento{{Data: hoje.AddDate(0, 0, 5), Quantidade: d(40), Documento: "OF 12"}},
		Compromissos: []Movimento{{Data: hoje.AddDate(0, 0, 10), Quantidade: d(30), Documento: "PV 7"}}}
	if _, o, _, ok := Atender(hoje, d(8), d(0), s); !ok || o != OrigemEstoque {
		t.Fatalf("8 com 10 livres sai do estoque: %s %v", o, ok)
	}
	// 20: a OF traz 40 no dia 5 (saldo 50), o PV 7 leva 30 no dia 10 (saldo 20) — dá.
	if data, o, _, ok := Atender(hoje, d(20), d(0), s); !ok || o != OrigemEmAndamento || !data.Equal(hoje.AddDate(0, 0, 5)) {
		t.Fatalf("20 pela OF: %v %s %v", data, o, ok)
	}
	// 25: depois do PV 7 sobram 20 — a entrada não cobre sem furar o outro pedido.
	if _, _, _, ok := Atender(hoje, d(25), d(0), s); ok {
		t.Fatal("25 tiraria do PV 7: não pode prometer pela OF")
	}
	// No mesmo cenário, outra linha já levou 5 do estoque.
	if _, o, _, _ := Atender(hoje, d(8), d(5), s); o == OrigemEstoque {
		t.Fatal("com 5 já consumidos, 8 não sai do estoque")
	}
}

func TestNovaReposicao(t *testing.T) {
	hoje := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC) // segunda
	compra, o, _ := NovaReposicao(hoje, SituacaoDoItem{PrazoCompraDias: 5}, time.Time{})
	if o != OrigemNovaCompra || !compra.Equal(hoje.AddDate(0, 0, 7)) {
		t.Fatalf("compra em 5 dias úteis: %v %s", compra, o)
	}
	// Componentes chegam na quarta; 2 dias de fila + 3 de fabricação = segunda+...
	fab, o, exp := NovaReposicao(hoje, SituacaoDoItem{Fabricado: true, PrazoFabricacaoDias: 3, FilaDias: 2}, hoje.AddDate(0, 0, 2))
	if o != OrigemNovaProducao || !fab.Equal(hoje.AddDate(0, 0, 9)) || len(exp) != 3 {
		t.Fatalf("produção: %v %s %v", fab, o, exp)
	}
}
