package governanca

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestConferirEmbalagens(t *testing.T) {
	um := decimal.NewFromInt(1)
	ok := []Embalagem{{GTIN: "7891000100103", Unidade: "un", Quantidade: um}, {GTIN: "17891000100100", Unidade: "CX", Quantidade: decimal.NewFromInt(12)}}
	if err := ConferirEmbalagens(ok, "UN"); err != nil {
		t.Fatal(err)
	}
	if ok[0].Unidade != "UN" {
		t.Fatal("unidade normaliza")
	}
	for nome, l := range map[string][]Embalagem{
		"dígito":   {{GTIN: "7891000100104", Unidade: "UN", Quantidade: um}},
		"sem unid": {{GTIN: "7891000100103", Quantidade: um}},
		"qtd":      {{GTIN: "7891000100103", Unidade: "UN"}},
		"gtin dup": {{GTIN: "7891000100103", Unidade: "UN", Quantidade: um}, {GTIN: "7891000100103", Unidade: "CX", Quantidade: um}},
		"unid dup": {{GTIN: "7891000100103", Unidade: "UN", Quantidade: um}, {GTIN: "96385074", Unidade: "UN", Quantidade: um}},
	} {
		if ConferirEmbalagens(l, "UN") == nil {
			t.Fatalf("%s deveria ser recusado", nome)
		}
	}
}

func TestRevisaoComVigencia(t *testing.T) {
	dia := func(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }
	fim := dia(9)
	lista := []Revisao{{Revisao: "A", De: dia(1), Ate: &fim}, {Revisao: "B", De: dia(10)}}
	if r := RevisaoVigente(lista, dia(5)); r == nil || r.Revisao != "A" {
		t.Fatalf("dia 5: %+v", r)
	}
	if r := RevisaoVigente(lista, dia(10)); r == nil || r.Revisao != "B" {
		t.Fatalf("dia 10: %+v", r)
	}
	antes := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if RevisaoVigente(lista, antes) != nil {
		t.Fatal("antes da primeira não há revisão")
	}
	nova := NovaRevisao{Revisao: " c ", De: dia(5), Motivo: "ECN"}
	if nova.Conferir(lista) == nil {
		t.Fatal("vigência tem de vir depois da última")
	}
	nova.De = dia(20)
	if err := nova.Conferir(lista); err != nil || nova.Revisao != "C" {
		t.Fatalf("%v %+v", err, nova)
	}
	dup := NovaRevisao{Revisao: "b", De: dia(20), Motivo: "x"}
	if dup.Conferir(lista) == nil {
		t.Fatal("revisão repetida")
	}
}

func TestHierarquiaCalculaOTotalDeUnidades(t *testing.T) {
	cx, pl := "CX", "UN"
	dez, quarenta := decimal.NewFromInt(10), decimal.NewFromInt(40)
	lista := []Embalagem{
		{Unidade: "PL", Contem: &cx, ContemQuantidade: &quarenta},
		{GTIN: "7891000100103", Unidade: "CX", Contem: &pl, ContemQuantidade: &dez},
	}
	if err := ConferirEmbalagens(lista, "UN"); err != nil {
		t.Fatal(err)
	}
	if !lista[0].Quantidade.Equal(decimal.NewFromInt(400)) || !lista[1].Quantidade.Equal(dez) {
		t.Fatalf("palete de 40 caixas de 10 = 400 unidades: %+v", lista)
	}
}

func TestHierarquiaRecusaOQueNaoFechaConta(t *testing.T) {
	cx, pl, fd, un := "CX", "PL", "FD", "UN"
	dois := decimal.NewFromInt(2)
	casos := map[string][]Embalagem{
		"ciclo":                    {{Unidade: "CX", Contem: &pl, ContemQuantidade: &dois}, {Unidade: "PL", Contem: &cx, ContemQuantidade: &dois}},
		"nível inexistente":        {{Unidade: "PL", Contem: &fd, ContemQuantidade: &dois}},
		"sem quantidade":           {{Unidade: "CX", Contem: &un}},
		"contém a si mesma":        {{Unidade: "CX", Contem: &cx, ContemQuantidade: &dois}},
		"sem GTIN nem nível":       {{Unidade: "CX", Quantidade: dois}},
		"unidade do item sem GTIN": {{Unidade: "UN", Quantidade: dois}},
	}
	for nome, lista := range casos {
		if ConferirEmbalagens(lista, "UN") == nil {
			t.Fatalf("%s devia ser recusado", nome)
		}
	}
}

func TestAtributosLivres(t *testing.T) {
	lista := []AtributoLivre{{Nome: "  tonalidade  da  madeira", Valor: " Nogueira   escura "}}
	if err := ConferirAtributosLivres(lista); err != nil {
		t.Fatal(err)
	}
	if lista[0].Nome != "TONALIDADE DA MADEIRA" || lista[0].Valor != "Nogueira escura" {
		t.Fatalf("normaliza: %+v", lista[0])
	}
	if ConferirAtributosLivres([]AtributoLivre{{Nome: "COR", Valor: "A"}, {Nome: "cor", Valor: "B"}}) == nil {
		t.Fatal("o mesmo nome em outra grafia repete o atributo")
	}
	if ConferirAtributosLivres([]AtributoLivre{{Nome: "COR"}}) == nil {
		t.Fatal("atributo sem valor não entra")
	}
	if ConferirAtributosLivres(make([]AtributoLivre, MaximoDeAtributosLivres+1)) == nil {
		t.Fatal("passa do limite")
	}
}

func TestEmbalagemRecusaOQueOBancoNaoGuarda(t *testing.T) {
	un, cx := "UN", "CX"
	enorme := decimal.New(1, 12)
	ummilhao := decimal.NewFromInt(1000000)
	longa := strings.Repeat("x", 121)
	casos := map[string][]Embalagem{
		"unidade longa":      {{Unidade: strings.Repeat("A", 21), Contem: &un, ContemQuantidade: &ummilhao}},
		"descrição longa":    {{Unidade: "CX", Contem: &un, ContemQuantidade: &ummilhao, Descricao: &longa}},
		"quantidade enorme":  {{Unidade: "CX", Contem: &un, ContemQuantidade: &enorme}},
		"produto estoura":    {{Unidade: "CX", Contem: &un, ContemQuantidade: &ummilhao}, {Unidade: "PL", Contem: &cx, ContemQuantidade: &ummilhao}},
		"unidade base com 5": {{GTIN: "7891000100103", Unidade: "UN", Quantidade: decimal.NewFromInt(5)}},
	}
	for nome, lista := range casos {
		if ConferirEmbalagens(lista, "UN") == nil {
			t.Fatalf("%s devia ser recusado", nome)
		}
	}
}

func TestQuantidadeSemNivelNaoFicaOrfa(t *testing.T) {
	dois := decimal.NewFromInt(2)
	lista := []Embalagem{{GTIN: "17891000100100", Unidade: "CX", Quantidade: dois, ContemQuantidade: &dois}}
	if err := ConferirEmbalagens(lista, "UN"); err != nil {
		t.Fatal(err)
	}
	if lista[0].ContemQuantidade != nil {
		t.Fatalf("sem o nível, a quantidade do nível não se grava: %+v", lista[0])
	}
}
