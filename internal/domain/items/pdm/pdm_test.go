package pdm

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

func texto(s string) *string { return &s }

func modelo() Modelo {
	return Modelo{GroupCode: 10, ModifierID: 7, ModifierDescription: "Parafuso sextavado", Atributos: []Atributo{
		{Nome: "DIAMETRO", Tipo: TipoNumero, Casas: 2, Prefixo: texto("M"), Obrigatorio: true, NaDescricao: true, NaAssinatura: true, Ativo: true},
		{Nome: "COMPRIMENTO", Tipo: TipoNumero, Unidade: texto("MM"), Obrigatorio: true, NaDescricao: true, NaAssinatura: true, Ativo: true},
		{Nome: "ACABAMENTO", Tipo: TipoLista, NaDescricao: true, NaAssinatura: true, Ativo: true, Valores: []Valor{
			{Valor: "ZINCADO BRANCO", Abreviacao: texto("ZB"), Ativo: true}, {Valor: "INOX", Ativo: true}, {Valor: "OXIDADO", Ativo: false}}},
		{Nome: "NORMA", Tipo: TipoTexto, NaDescricao: false, NaAssinatura: false, Ativo: true},
		{Nome: "ANTIGO", Tipo: TipoTexto, Ativo: false, Obrigatorio: true},
	}}
}

func attr(n, v string) valueobject.Attribute { return valueobject.Attribute{Name: n, Value: v} }

func TestAplicarGeraDescricaoNaOrdemDoModelo(t *testing.T) {
	r := modelo().Aplicar([]valueobject.Attribute{attr("acabamento", "zincado branco"), attr("comprimento", "30"), attr("diametro", "6,00"), attr("norma", "din 933")})
	if len(r.Erros) != 0 {
		t.Fatalf("erros inesperados: %v", r.Erros)
	}
	if r.Descricao != "PARAFUSO SEXTAVADO M6 30MM ZB" {
		t.Fatalf("descrição = %q", r.Descricao)
	}
	if r.Atributos[0].Value != "6.00" {
		t.Fatalf("valor normalizado = %q", r.Atributos[0].Value)
	}
}

func TestAssinaturaIgnoraGrafiaEAtributoForaDaAssinatura(t *testing.T) {
	a := modelo().Aplicar([]valueobject.Attribute{attr("DIAMETRO", "6"), attr("COMPRIMENTO", "30"), attr("NORMA", "DIN 933")})
	b := modelo().Aplicar([]valueobject.Attribute{attr("diametro", "6,0"), attr("comprimento", " 30 "), attr("norma", "ISO 4017")})
	if a.Assinatura != b.Assinatura {
		t.Fatal("a mesma peça escrita de outro jeito deve ter a mesma assinatura")
	}
	c := modelo().Aplicar([]valueobject.Attribute{attr("DIAMETRO", "8"), attr("COMPRIMENTO", "30")})
	if a.Assinatura == c.Assinatura {
		t.Fatal("peças diferentes não podem ter a mesma assinatura")
	}
}

func TestAplicarApontaOQueFalta(t *testing.T) {
	r := modelo().Aplicar([]valueobject.Attribute{attr("DIAMETRO", "abc"), attr("ACABAMENTO", "OXIDADO"), attr("COR", "AZUL")})
	if len(r.Erros) != 4 {
		t.Fatalf("esperava número inválido, comprimento faltando, valor inativo e atributo estranho: %v", r.Erros)
	}
}

func TestModeloSemAtributoAtivoNaoTemPadrao(t *testing.T) {
	m := Modelo{Atributos: []Atributo{{Nome: "X", Ativo: false}}}
	if m.TemPadrao() {
		t.Fatal("modelo só com atributo inativo não tem padrão")
	}
}

func TestDescricaoLongaAvisaDoLimiteDaNota(t *testing.T) {
	longo := ""
	for i := 0; i < 130; i++ {
		longo += "A"
	}
	r := Modelo{ModifierDescription: longo}.Aplicar(nil)
	if !r.PassaDaNota {
		t.Fatal("descrição acima de 120 caracteres deve avisar")
	}
}

func TestConferirAtributoDeListaExigeValorAtivo(t *testing.T) {
	a := Atributo{Nome: "cor", GroupCode: 1, ModifierID: 1, Tipo: TipoLista, Valores: []Valor{{Valor: "azul"}}}
	if err := a.Conferir(); err == nil {
		t.Fatal("lista sem valor ativo deve ser recusada")
	}
}

func TestItemAntigoSalvaComAtributoEValorInativados(t *testing.T) {
	m := modelo()
	m.Atributos[2].Valores[1].Ativo = false // INOX saiu da lista
	informados := []valueobject.Attribute{attr("DIAMETRO", "6"), attr("COMPRIMENTO", "30"), attr("ACABAMENTO", "INOX"), attr("ANTIGO", "X")}
	if r := m.Aplicar(informados); len(r.Erros) != 2 {
		t.Fatalf("item novo não usa o que foi inativado: %v", r.Erros)
	}
	if r := m.AplicarNaAlteracao(informados); len(r.Erros) != 0 {
		t.Fatalf("o item antigo mantém o que já tinha: %v", r.Erros)
	}
	if r := m.AplicarNaAlteracao(informados[:3]); len(r.Erros) != 0 {
		t.Fatalf("atributo inativo não vira obrigatório: %v", r.Erros)
	}
}
