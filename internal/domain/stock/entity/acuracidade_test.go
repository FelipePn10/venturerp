package entity

import "testing"

var tolPadrao = ToleranciaAcuracidade{A: dd("0"), B: dd("2"), C: dd("5")}

func contagem(mes, classe, esperado, contado string) ContagemAcuracidade {
	return ContagemAcuracidade{Mes: mes, ItemCode: 1, Codigo: "X", Classe: classe, WarehouseID: 9, Endereco: "A1", Esperado: dd(esperado), Contado: dd(contado), Custo: dd("10")}
}

func TestAcertoRespeitaAClasse(t *testing.T) {
	casos := []struct {
		classe, esperado, contado string
		acerta                    bool
	}{
		{"A", "100", "100", true}, {"A", "100", "99", false},
		{"B", "100", "98", true}, {"B", "100", "97", false},
		{"C", "100", "105", true}, {"", "100", "105", true}, {"", "100", "106", false},
		{"C", "0", "0", true}, {"C", "0", "1", false},
	}
	for _, c := range casos {
		if contagem("", c.classe, c.esperado, c.contado).Acertou(tolPadrao) != c.acerta {
			t.Fatalf("%+v", c)
		}
	}
}

func TestApurarAcuracidade(t *testing.T) {
	r := ApurarAcuracidade([]ContagemAcuracidade{
		contagem("2026-09", "A", "100", "100"),
		contagem("2026-09", "A", "100", "90"),
		contagem("2026-10", "A", "100", "95"),
		contagem("2026-10", "C", "100", "103"),
	}, tolPadrao)
	if r.Total.Contagens != 4 || r.Total.Acertos != 2 || !r.Total.PctLinhas.Equal(dd("50")) {
		t.Fatalf("2 de 4 linhas: %+v", r.Total)
	}
	// Erro 10+5+3=18 unidades sobre 400 esperadas: 95,5% do valor.
	if r.Total.PctValor == nil || !r.Total.PctValor.Equal(dd("95.5")) || !r.Total.Divergente.Equal(dd("180")) {
		t.Fatalf("valor: %+v", r.Total)
	}
	if len(r.PorMes) != 2 || r.PorMes[0].Chave != "2026-09" || !r.PorMes[1].PctLinhas.Equal(dd("50")) {
		t.Fatalf("por mês: %+v", r.PorMes)
	}
	if len(r.PorClasse) != 2 || r.PorClasse[0].Chave != "A" || r.PorClasse[0].Acertos != 1 {
		t.Fatalf("por classe: %+v", r.PorClasse)
	}
	if len(r.Itens) != 1 || r.Itens[0].Erros != 2 || r.Itens[0].Contagens != 4 || !r.Itens[0].Divergente.Equal(dd("150")) {
		t.Fatalf("item recorrente: %+v", r.Itens)
	}
	if len(r.Enderecos) != 1 || r.Enderecos[0].Endereco != "A1" {
		t.Fatalf("endereço recorrente: %+v", r.Enderecos)
	}
}

func TestAcuracidadeSemBaseDeValor(t *testing.T) {
	r := ApurarAcuracidade([]ContagemAcuracidade{contagem("2026-10", "C", "0", "5")}, tolPadrao)
	if r.Total.PctValor != nil || len(r.Itens) != 0 {
		t.Fatalf("sem valor esperado não há percentual, e um erro não é recorrência: %+v", r)
	}
	if v := ApurarAcuracidade(nil, tolPadrao); v.Total.Contagens != 0 || v.Itens == nil || v.PorMes == nil {
		t.Fatalf("sem contagens devolve listas vazias: %+v", v)
	}
}

func TestFaixasDeIdade(t *testing.T) {
	dias := func(d int) *int { return &d }
	f := FaixasDeIdade([]AnaliseItem{
		{Saldo: dd("1"), Valor: dd("100"), DiasSemMovimento: dias(90)},
		{Saldo: dd("1"), Valor: dd("100"), DiasSemMovimento: dias(91)},
		{Saldo: dd("1"), Valor: dd("200"), DiasSemMovimento: dias(366)},
		{Saldo: dd("1"), Valor: dd("100")},
		{Saldo: dd("0"), Valor: dd("999"), DiasSemMovimento: dias(999)},
	})
	if f[0].Itens != 1 || f[1].Itens != 1 || f[2].Itens != 0 || f[3].Itens != 1 || f[4].Itens != 1 {
		t.Fatalf("%+v", f)
	}
	if !f[3].Pct.Equal(dd("40")) {
		t.Fatalf("200 de 500 é 40%%: %+v", f[3])
	}
}
