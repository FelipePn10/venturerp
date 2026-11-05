package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func dd(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func ddp(s string) *decimal.Decimal {
	v := dd(s)
	return &v
}

func TestQuantidadeDeReposicao(t *testing.T) {
	casos := []struct {
		nome, alvo, projetado, min, multiplo, esperado string
	}{
		// Barra de 6,35 M: faltam 10 M, sobem para 2 barras — 12,7 M, não 13.
		{"múltiplo fracionado", "10", "0", "0", "6.35", "12.7"},
		{"múltiplo inteiro", "12", "1.5", "0", "5", "15"},
		{"lote mínimo fracionado sem múltiplo sobe ao inteiro", "1", "0", "2.5", "0", "3"},
		{"lote mínimo e múltiplo fracionados", "1", "0", "7", "6.35", "12.7"},
		{"projeção que alcança o alvo não pede nada", "5", "5", "0", "0", "0"},
		{"projeção acima do alvo não pede nada", "5", "9", "10", "0", "0"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := QuantidadeDeReposicao(dd(c.alvo), dd(c.projetado), dd(c.min), dd(c.multiplo)); !got.Equal(dd(c.esperado)) {
				t.Fatalf("esperado %s, veio %s", c.esperado, got)
			}
		})
	}
}

func itemComprado1() ItemReposicao {
	return ItemReposicao{ItemCode: 1, TipoItem: itemComprado, PorAlmox: map[int64]SaldoReposicao{}}
}

// A segurança já está no ponto: antes ela somava de novo na quantidade.
func TestSemPoliticaNaoContaASegurancaDuasVezes(t *testing.T) {
	it := itemComprado1()
	it.Ponto, it.Seguranca = dd("15"), dd("5")
	it.Total = SaldoReposicao{Fisico: dd("10")}
	s := it.Sugestoes()
	if len(s) != 1 || !s[0].Sugerido.Equal(dd("5")) || s[0].Origem != OrigemCompra || s[0].WarehouseID != nil {
		t.Fatalf("repõe até o ponto, 15 − 10 = 5: %+v", s)
	}
}

func TestSemPoliticaComMaximoRepoeAteOMaximo(t *testing.T) {
	it := itemComprado1()
	it.Ponto, it.Maximo = dd("10"), dd("50")
	it.Total = SaldoReposicao{Fisico: dd("10"), Chegando: dd("0")}
	if s := it.Sugestoes(); len(s) != 1 || !s[0].Sugerido.Equal(dd("40")) {
		t.Fatalf("no ponto dispara e vai ao máximo: %+v", s)
	}
}

func TestOrigemPadraoPeloTipoDoItem(t *testing.T) {
	for tipo, esperado := range map[int]string{itemFabricado: OrigemProducao, itemComprado: OrigemCompra, itemDeTerceiro: "", itemServico: ""} {
		it := ItemReposicao{TipoItem: tipo, Ponto: dd("5")}
		s := it.Sugestoes()
		if esperado == "" {
			if len(s) != 0 {
				t.Fatalf("tipo %d não se repõe: %+v", tipo, s)
			}
			continue
		}
		if len(s) != 1 || s[0].Origem != esperado {
			t.Fatalf("tipo %d: %+v", tipo, s)
		}
	}
}

// Retido e requisitado não estão livres: a falta aparece antes de faltar.
func TestLivreDescontaRetidoERequisitado(t *testing.T) {
	it := itemComprado1()
	it.Ponto = dd("10")
	it.Total = SaldoReposicao{Fisico: dd("30"), Reservado: dd("5"), Retido: dd("10"), Requisitado: dd("8")}
	if s := it.Sugestoes(); len(s) != 1 || !s[0].Projetado.Equal(dd("7")) || !s[0].Sugerido.Equal(dd("3")) {
		t.Fatalf("30 − 5 − 10 − 8 = 7: %+v", s)
	}
}

func TestPoliticaHerdaDoCadastroOQueNaoDiz(t *testing.T) {
	it := itemComprado1()
	it.Ponto, it.Maximo = dd("10"), dd("40")
	it.Politicas = []PoliticaReposicao{{WarehouseID: 7, Origem: OrigemCompra}, {WarehouseID: 8, Minimo: ddp("3"), Maximo: ddp("6"), Origem: OrigemCompra}}
	it.PorAlmox[7] = SaldoReposicao{Fisico: dd("10")}
	it.PorAlmox[8] = SaldoReposicao{Fisico: dd("2")}
	s := it.Sugestoes()
	if len(s) != 2 || !s[0].Sugerido.Equal(dd("30")) || !s[1].Sugerido.Equal(dd("4")) {
		t.Fatalf("7 herda 10/40 e pede 30; 8 usa 3/6 e pede 4: %+v", s)
	}
	if *s[0].WarehouseID != 7 || *s[1].WarehouseID != 8 {
		t.Fatalf("cada sugestão é do seu almoxarifado: %+v", s)
	}
}

// Requisição de compra não tem destino: cobre uma vez só, não em cada almoxarifado.
func TestCompraSemDestinoCobreUmaVezSo(t *testing.T) {
	it := itemComprado1()
	it.SemDestino = dd("10")
	it.Politicas = []PoliticaReposicao{{WarehouseID: 1, Minimo: ddp("10"), Origem: OrigemCompra}, {WarehouseID: 2, Minimo: ddp("10"), Origem: OrigemCompra}}
	s := it.Sugestoes()
	if len(s) != 1 || *s[0].WarehouseID != 2 {
		t.Fatalf("a compra pendente cobre o 1 e só o 2 precisa: %+v", s)
	}
	if !s[0].Sugerido.Equal(dd("10")) {
		t.Fatalf("o que sobra é pedir 10: %+v", s)
	}
}

func TestTransferenciaRespeitaOPontoDaOrigemEOQueJaFoiUsado(t *testing.T) {
	origem := int64(1)
	it := itemComprado1()
	it.Politicas = []PoliticaReposicao{
		{WarehouseID: 1, Minimo: ddp("5"), Origem: OrigemCompra},
		{WarehouseID: 2, Minimo: ddp("10"), Origem: OrigemTransferencia, AlmoxOrigem: &origem},
		{WarehouseID: 3, Minimo: ddp("10"), Origem: OrigemTransferencia, AlmoxOrigem: &origem},
	}
	it.PorAlmox[1] = SaldoReposicao{Fisico: dd("17")}
	s := it.Sugestoes()
	if len(s) != 2 {
		t.Fatalf("os dois destinos aparecem: %+v", s)
	}
	if !s[0].Sugerido.Equal(dd("10")) || s[0].Aviso != "" {
		t.Fatalf("17 − 5 de ponto = 12 livres, o primeiro leva 10: %+v", s[0])
	}
	if !s[1].Sugerido.Equal(dd("2")) || s[1].Aviso == "" {
		t.Fatalf("sobram 2 para o segundo, com aviso: %+v", s[1])
	}
}

func TestTransferenciaSemSaldoNaOrigemAvisaSemPedir(t *testing.T) {
	origem := int64(1)
	it := itemComprado1()
	it.Politicas = []PoliticaReposicao{{WarehouseID: 2, Minimo: ddp("10"), Origem: OrigemTransferencia, AlmoxOrigem: &origem}}
	s := it.Sugestoes()
	if len(s) != 1 || !s[0].Sugerido.IsZero() || s[0].Aviso == "" {
		t.Fatalf("aparece com quantidade zero e aviso: %+v", s)
	}
}

func TestValidarPolitica(t *testing.T) {
	origem := int64(1)
	mesmo := int64(2)
	ruins := []PoliticaReposicao{
		{WarehouseID: 2, Origem: OrigemCompra},
		{ItemCode: 1, Origem: OrigemCompra},
		{ItemCode: 1, WarehouseID: 2, Origem: "X"},
		{ItemCode: 1, WarehouseID: 2, Origem: OrigemTransferencia},
		{ItemCode: 1, WarehouseID: 2, Origem: OrigemTransferencia, AlmoxOrigem: &mesmo},
		{ItemCode: 1, WarehouseID: 2, Origem: OrigemCompra, AlmoxOrigem: &origem},
		{ItemCode: 1, WarehouseID: 2, Origem: OrigemCompra, Minimo: ddp("10"), Maximo: ddp("5")},
		{ItemCode: 1, WarehouseID: 2, Origem: OrigemCompra, Minimo: ddp("-1")},
	}
	for i, p := range ruins {
		if p.Validar() == nil {
			t.Fatalf("caso %d deveria ser recusado: %+v", i, p)
		}
	}
	if err := (PoliticaReposicao{ItemCode: 1, WarehouseID: 2, Origem: OrigemTransferencia, AlmoxOrigem: &origem}).Validar(); err != nil {
		t.Fatal(err)
	}
	if OrigemAceitaParaItem(OrigemProducao, itemComprado) == nil || OrigemAceitaParaItem(OrigemCompra, itemServico) == nil || OrigemAceitaParaItem(OrigemCompra, itemFabricado) != nil {
		t.Fatal("regra de origem por tipo do item")
	}
}
