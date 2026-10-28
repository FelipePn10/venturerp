package contratovenda

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func contrato() *Contrato {
	teto := d("1000")
	return &Contrato{Situacao: Ativo, CustomerCode: 1, Descricao: "x", VigenciaInicio: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		VigenciaFim: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), ValorContratado: &teto, ValorConsumido: d("600"),
		Itens: []Item{{ID: 1, ItemCode: 10, ItemNome: "CHAPA", Quantidade: d("100"), PrecoUnitario: d("10"), Consumido: d("60"), Saldo: d("40")}}}
}

func TestConferirChamada(t *testing.T) {
	c := contrato()
	jun := time.Date(2026, 6, 10, 15, 0, 0, 0, time.UTC)
	total, err := ConferirChamada(c, []LinhaChamada{{ContratoItemID: 1, Quantidade: d("30")}}, jun)
	if err != nil || !total.Equal(d("300")) {
		t.Fatalf("dentro do saldo: %s %v", total, err)
	}
	casos := map[string]struct {
		l    []LinhaChamada
		data time.Time
	}{
		"acima do saldo":         {[]LinhaChamada{{ContratoItemID: 1, Quantidade: d("41")}}, jun},
		"acima do teto":          {[]LinhaChamada{{ContratoItemID: 1, Quantidade: d("40")}}, jun}, // 400 > 400? (teto 1000 - 600 = 400) — igual passa
		"fora da vigência":       {[]LinhaChamada{{ContratoItemID: 1, Quantidade: d("1")}}, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		"item de outro contrato": {[]LinhaChamada{{ContratoItemID: 9, Quantidade: d("1")}}, jun},
		"sem linha":              {nil, jun},
	}
	for nome, cs := range casos {
		_, err := ConferirChamada(c, cs.l, cs.data)
		if nome == "acima do teto" {
			if err != nil {
				t.Errorf("400 de saldo em valor cabe 400: %v", err)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s: devia recusar", nome)
		}
	}
	c.ValorConsumido = d("700")
	if _, err := ConferirChamada(c, []LinhaChamada{{ContratoItemID: 1, Quantidade: d("31")}}, jun); err == nil {
		t.Fatal("310 passa dos 300 de saldo em valor")
	}
	c.Situacao = Rascunho
	if _, err := ConferirChamada(c, []LinhaChamada{{ContratoItemID: 1, Quantidade: d("1")}}, jun); err == nil {
		t.Fatal("rascunho não recebe chamada")
	}
}

func TestTransicao(t *testing.T) {
	c := contrato()
	c.Situacao = Rascunho
	if Transicao(&Contrato{Situacao: Rascunho}, Ativo, false) == nil {
		t.Fatal("ativar sem item")
	}
	if Transicao(c, Ativo, false) != nil {
		t.Fatal("ativar com item")
	}
	c.Situacao = Ativo
	if Transicao(c, Cancelado, true) == nil || Transicao(c, Encerrado, true) != nil {
		t.Fatal("com chamada: encerra, não cancela")
	}
	c.Situacao = Encerrado
	if Transicao(c, Ativo, false) == nil {
		t.Fatal("encerrado é final")
	}
}

func TestCadastro(t *testing.T) {
	c := contrato()
	if c.ConferirCadastro() != nil {
		t.Fatal("válido")
	}
	c.Itens = append(c.Itens, Item{ItemCode: 10, Quantidade: d("1")})
	if c.ConferirCadastro() == nil {
		t.Fatal("item repetido")
	}
	c = contrato()
	c.VigenciaFim = c.VigenciaInicio.AddDate(0, 0, -1)
	if c.ConferirCadastro() == nil {
		t.Fatal("vigência invertida")
	}
}
