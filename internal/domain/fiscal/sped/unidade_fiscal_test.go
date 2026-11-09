package sped

import (
	"testing"

	"github.com/shopspring/decimal"
)

// 0190, 0200, 0220, C170 e H010 levam o código fiscal da unidade (TONELADA →
// T) com a descrição do catálogo; sem catálogo, a unidade sai como está.
func TestEFDUsaCodigoFiscalDaUnidade(t *testing.T) {
	d := DadosEFD{
		UnidadesFiscais: map[string]UnidadeFiscal{"TONELADA": {Codigo: "T", Descricao: "Tonelada"}, "KG": {Codigo: "KG", Descricao: "Quilograma"}},
		Entradas: []NotaEntrada{{Itens: []ItemEntrada{{
			Item: ItemCadastro{Cod: "10", Desc: "Aço", UnidInv: "TONELADA"}, Unid: "KG", Qtd: decimal.NewFromInt(1000), Fator: decimal.RequireFromString("0.001"),
		}}}},
	}
	p := Montar(d)
	if len(p.Itens) != 1 || p.Itens[0].UnCom != "T" {
		t.Fatalf("0200 deveria sair em T: %+v", p.Itens)
	}
	if len(p.Itens[0].Conversoes) != 1 || p.Itens[0].Conversoes[0].UnidConv != "KG" {
		t.Fatalf("0220 de KG para T: %+v", p.Itens[0].Conversoes)
	}
	desc := map[string]string{}
	for _, u := range p.Unidades {
		desc[u.CodUnd] = u.DescUnd
		if len(u.CodUnd) > 6 {
			t.Fatalf("0190 com mais de 6 caracteres: %s", u.CodUnd)
		}
	}
	if desc["T"] != "Tonelada" || desc["KG"] != "Quilograma" {
		t.Fatalf("0190 com a descrição do catálogo: %+v", p.Unidades)
	}
	if c := p.DocumentosFiscais[0].Itens[0].UnCom; c != "KG" {
		t.Fatalf("C170 na unidade do documento (KG), veio %s", c)
	}
}
