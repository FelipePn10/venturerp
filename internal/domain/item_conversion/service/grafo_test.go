package service

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func aresta(id int64, de, para, qde, qpara string, o Origem) Aresta {
	return Aresta{ID: id, De: de, Para: para, QtdDe: d(qde), QtdPara: d(qpara), Origem: o}
}

// O caso que motivou o motor: compra em KG, estoque em UN, estrutura em MM.
func TestResolverCadeiaCompraEstoqueEstrutura(t *testing.T) {
	arestas := []Aresta{
		aresta(1, "UN", "KG", "1", "23.5", OrigemItem),  // 1 chapa = 23,5 kg
		aresta(2, "UN", "MM", "1", "6000", OrigemItem),  // 1 barra = 6000 mm
		aresta(3, "M", "MM", "1", "1000", OrigemPadrao), // padrão da empresa
	}
	c, ok := Resolver(arestas, "KG", "UN")
	if !ok || !c.Converter(d("47")).Equal(d("2")) {
		t.Fatalf("47 KG deveriam ser 2 UN, deu %v (ok=%v)", c.Converter(d("47")), ok)
	}
	c, ok = Resolver(arestas, "KG", "MM")
	if !ok || len(c.Passos) != 2 {
		t.Fatalf("KG→MM deveria sair em dois passos, deu %+v", c)
	}
	if got := c.Converter(d("23.5")); !got.Equal(d("6000")) {
		t.Fatalf("23,5 KG = 6000 MM, deu %v", got)
	}
	// Três saltos, com a padrão no meio: KG → UN → MM → M.
	c, ok = Resolver(arestas, "KG", "M")
	if !ok || !c.Converter(d("23.5")).Equal(d("6")) {
		t.Fatalf("23,5 KG = 6 M, deu %v", c.Converter(d("23.5")))
	}
	if !strings.Contains(c.Explicacao(), "padrão da empresa") {
		t.Fatalf("a explicação deveria citar a conversão padrão: %s", c.Explicacao())
	}
}

// A fração vai exata até o fim: a inversa de 6000 não pode virar 0,999999.
func TestResolverFracaoExata(t *testing.T) {
	arestas := []Aresta{aresta(1, "BARRA", "MM", "1", "6000", OrigemItem)}
	c, _ := Resolver(arestas, "MM", "BARRA")
	if got := c.Converter(d("6000")); !got.Equal(d("1")) {
		t.Fatalf("6000 MM = 1 BARRA exato, deu %v", got)
	}
	// 3 UN = 1 PCT: a dízima 0,333… não pode contaminar o retorno.
	c, _ = Resolver([]Aresta{aresta(1, "PCT", "UN", "1", "3", OrigemItem)}, "UN", "PCT")
	if got := c.Converter(d("3")); !got.Equal(d("1")) {
		t.Fatalf("3 UN = 1 PCT exato, deu %v", got)
	}
}

// O mais específico ganha: lote > item com máscara > item > padrão.
func TestResolverPrecedencia(t *testing.T) {
	arestas := []Aresta{
		aresta(1, "UN", "KG", "1", "23", OrigemItem),
		aresta(2, "UN", "KG", "1", "24.1", OrigemLote),
		aresta(3, "KG", "UN", "25", "1", OrigemItemMascara),
	}
	c, _ := Resolver(arestas, "UN", "KG")
	if !c.Fator().Equal(d("24.1")) || c.OrigemPrincipal() != OrigemLote {
		t.Fatalf("o lote deveria ganhar, deu %v (%s)", c.Fator(), c.OrigemPrincipal())
	}
	c, _ = Resolver(arestas[:1:1], "UN", "KG")
	if !c.Fator().Equal(d("23")) {
		t.Fatalf("sem lote vale o item, deu %v", c.Fator())
	}
	c, _ = Resolver([]Aresta{arestas[0], arestas[2]}, "UN", "KG")
	if !c.Fator().Equal(d("25")) {
		t.Fatalf("a máscara deveria ganhar do item genérico, deu %v", c.Fator())
	}
}

func TestResolverSemCaminhoEMesmaUnidade(t *testing.T) {
	if _, ok := Resolver([]Aresta{aresta(1, "UN", "KG", "1", "2", OrigemItem)}, "M", "KG"); ok {
		t.Fatal("M→KG não tem caminho")
	}
	c, ok := Resolver(nil, "kg", " KG ")
	if !ok || !c.Fator().Equal(d("1")) {
		t.Fatal("mesma unidade é fator 1")
	}
}

func TestResolverLimiteDePassos(t *testing.T) {
	var arestas []Aresta
	us := []string{"A", "B", "C", "D", "E", "F"}
	for i := 0; i+1 < len(us); i++ {
		arestas = append(arestas, aresta(int64(i), us[i], us[i+1], "1", "2", OrigemItem))
	}
	if _, ok := Resolver(arestas, "A", "E"); !ok {
		t.Fatal("quatro passos ainda resolvem")
	}
	if _, ok := Resolver(arestas, "A", "F"); ok {
		t.Fatal("cinco passos passam do limite")
	}
}

func TestVerificarConsistencia(t *testing.T) {
	existentes := []Aresta{
		aresta(1, "UN", "M", "1", "6", OrigemItem),
		aresta(2, "M", "KG", "1", "4", OrigemItem),
	}
	if c := VerificarConsistencia(existentes, aresta(0, "UN", "KG", "1", "23", OrigemItem)); c == nil {
		t.Fatal("UN→KG = 23 contradiz 6 × 4 = 24")
	} else if !strings.Contains(c.Error(), "24") {
		t.Fatalf("a mensagem deveria mostrar o fator existente: %s", c.Error())
	}
	if c := VerificarConsistencia(existentes, aresta(0, "KG", "UN", "24", "1", OrigemItem)); c != nil {
		t.Fatalf("24 KG = 1 UN é coerente: %v", c)
	}
	// Substituir a própria conversão não é conflito.
	if c := VerificarConsistencia(existentes, aresta(0, "UN", "M", "1", "5", OrigemItem)); c != nil {
		t.Fatalf("trocar o fator do mesmo par não conflita: %v", c)
	}
	// Caminho só com padrões não "contradiz" o item.
	padrao := []Aresta{aresta(1, "M", "MM", "1", "1000", OrigemPadrao)}
	if c := VerificarConsistencia(padrao, aresta(0, "M", "MM", "1", "999", OrigemItem)); c != nil {
		t.Fatalf("padrão não bloqueia exceção do item: %v", c)
	}
}

func TestConverterPreco(t *testing.T) {
	c, _ := Resolver([]Aresta{aresta(1, "UN", "KG", "1", "25", OrigemItem)}, "KG", "UN")
	// R$ 10/kg → R$ 250/un
	if got := c.ConverterPreco(d("10")); !got.Equal(d("250")) {
		t.Fatalf("preço por UN deveria ser 250, deu %v", got)
	}
}

func TestPoliticaDeArredondamento(t *testing.T) {
	a := aresta(1, "CX", "UN", "1", "12", OrigemItem)
	a.ArredondamentoPct, a.Tolerancia, a.TipoTolerancia = d("1"), d("2"), "PERCENT"
	c, _ := Resolver([]Aresta{a, aresta(2, "DZ", "UN", "1", "12", OrigemPadrao)}, "DZ", "CX")
	arr, tol, tipo := c.PoliticaDeArredondamento()
	if !arr.Equal(d("1")) || !tol.Equal(d("2")) || tipo != "PERCENT" {
		t.Fatalf("política deveria vir da conversão do item: %v %v %s", arr, tol, tipo)
	}
}
