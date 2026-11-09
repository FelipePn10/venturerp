package service

import "testing"

// A segunda unidade do cadastro do item perde para a conversão explícita do
// mesmo par e ganha da padrão da empresa.
func TestPrecedenciaDaSegundaUnidade(t *testing.T) {
	segunda := aresta(0, "UN", "KG", "1", "25", OrigemSegundaUnidade)
	item := aresta(1, "UN", "KG", "1", "24", OrigemItem)
	padrao := aresta(2, "UN", "KG", "1", "30", OrigemPadrao)

	c, ok := Resolver([]Aresta{padrao, segunda}, "UN", "KG")
	if !ok || !c.Fator().Equal(d("25")) || c.OrigemPrincipal() != OrigemSegundaUnidade {
		t.Fatalf("segunda unidade deveria ganhar da padrão: %+v", c)
	}
	c, ok = Resolver([]Aresta{segunda, padrao, item}, "KG", "UN")
	if !ok || !c.Converter(d("48")).Equal(d("2")) || c.OrigemPrincipal() != OrigemItem {
		t.Fatalf("a conversão do item (24) deveria ganhar da segunda unidade: %+v", c)
	}
	if OrigemSegundaUnidade.Rotulo() != "segunda unidade do item" {
		t.Fatalf("rótulo inesperado: %s", OrigemSegundaUnidade.Rotulo())
	}
	// A conversão nova do mesmo par não "contradiz" a segunda unidade: ela a substitui.
	if conf := VerificarConsistencia([]Aresta{segunda}, item); conf != nil {
		t.Fatalf("o mesmo par não é contradição de ciclo: %v", conf)
	}
}

// A embalagem do cadastro (1 CX = 12 UN) converte sem conversão na VSUP0110,
// perde para a do item e para a segunda unidade e ganha da padrão.
func TestPrecedenciaDaEmbalagem(t *testing.T) {
	emb := aresta(0, "CX", "UN", "1", "12", OrigemEmbalagem)
	item := aresta(1, "CX", "UN", "1", "10", OrigemItem)
	padrao := aresta(2, "CX", "UN", "1", "20", OrigemPadrao)
	if c, ok := Resolver([]Aresta{padrao, emb}, "CX", "UN"); !ok || !c.Fator().Equal(d("12")) || c.OrigemPrincipal() != OrigemEmbalagem {
		t.Fatalf("embalagem ganha da padrão: %+v", c)
	}
	if c, ok := Resolver([]Aresta{emb, item, padrao}, "UN", "CX"); !ok || !c.Converter(d("30")).Equal(d("3")) || c.OrigemPrincipal() != OrigemItem {
		t.Fatalf("a conversão do item ganha da embalagem: %+v", c)
	}
	if OrigemEmbalagem.Rotulo() != "embalagem do item" {
		t.Fatalf("rótulo inesperado: %s", OrigemEmbalagem.Rotulo())
	}
	// Encadeia: 2 CX → KG com 1 UN = 0,5 KG do item = 12 KG.
	kg := aresta(3, "UN", "KG", "1", "0.5", OrigemItem)
	if c, ok := Resolver([]Aresta{emb, kg}, "CX", "KG"); !ok || !c.Converter(d("2")).Equal(d("12")) {
		t.Fatalf("CX → KG pela embalagem e pela conversão do item: %+v", c)
	}
}
