package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func str(s string) *string { return &s }

// Uma transportadora cadastrada para uma faixa estreita de CEP era cotada como
// se atendesse o estado inteiro sempre que a cotação vinha só com a UF — e a
// cotação permite destino só com UF.
func TestFaixaDeCepNaoViraEstadoInteiro(t *testing.T) {
	area := &RegiaoAtendida{
		IsActive: true, State: str("SP"),
		PostalCodeFrom: str("13000000"), PostalCodeTo: str("13099999"),
	}
	casos := []struct {
		nome string
		uf   string
		cep  string
		quer bool
	}{
		{"CEP dentro da faixa", "SP", "13050-000", true},
		{"CEP fora da faixa", "SP", "01310-100", false},
		{"só a UF, sem CEP", "SP", "", false},
		{"CEP incompleto", "SP", "13050", false},
		{"outra UF", "RJ", "", false},
	}
	for _, c := range casos {
		if got := area.Atende(c.uf, "", c.cep); got != c.quer {
			t.Fatalf("%s: atende=%v, queria %v", c.nome, got, c.quer)
		}
	}
}

// Região sem faixa de CEP continua atendendo o estado todo.
func TestRegiaoPorEstadoContinuaValendo(t *testing.T) {
	area := &RegiaoAtendida{IsActive: true, State: str("PR")}
	if !area.Atende("pr", "", "") {
		t.Fatal("região só por UF deveria atender o estado inteiro")
	}
	if area.Atende("SC", "", "88000000") {
		t.Fatal("região de PR não pode atender SC")
	}
}

func TestRegiaoInativaNaoAtende(t *testing.T) {
	area := &RegiaoAtendida{IsActive: false, State: str("PR")}
	if area.Atende("PR", "", "") {
		t.Fatal("região inativa não atende")
	}
}

// Região de uma cidade vale só para ela; e, quando várias regiões atendem o
// destino, a mais específica (CEP > cidade > UF) define preço e prazo.
func TestRegiaoDaCidadeEEspecificidade(t *testing.T) {
	cidade := &RegiaoAtendida{IsActive: true, State: str("PR"), City: str("Curitiba"), LeadDays: 1}
	if !cidade.Atende("PR", "curitíba ", "") || cidade.Atende("PR", "Londrina", "") || cidade.Atende("PR", "", "") {
		t.Fatal("região de cidade deve atender só a cidade (sem diferença de acento/caixa)")
	}
	c := &Transportadora{IsActive: true, Modal: ModalAereo, FreightKgRate: decimal.NewFromInt(1), ServiceAreas: []*RegiaoAtendida{
		{IsActive: true, State: str("PR"), LeadDays: 3},
		cidade,
		{IsActive: true, State: str("PR"), PostalCodeFrom: str("80010000"), PostalCodeTo: str("80010999"), LeadDays: 0, KgRate: decimal.NewFromInt(2)},
	}}
	sexta := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	casos := []struct {
		nome, cidade, cep string
		prazo             int16
	}{
		{"estado", "Londrina", "", 3},
		{"cidade", "Curitiba", "", 1},
		{"faixa de CEP vence a cidade", "Curitiba", "80010-500", 0},
	}
	for _, k := range casos {
		cot, err := c.Cotar(Carga{PesoKg: decimal.NewFromInt(10), UF: "PR", Cidade: k.cidade, CEP: k.cep, Base: sexta})
		if err != nil {
			t.Fatalf("%s: %v", k.nome, err)
		}
		if cot.PrazoDias != k.prazo {
			t.Errorf("%s: prazo %d, queria %d (região %s)", k.nome, cot.PrazoDias, k.prazo, *cot.Regiao.State)
		}
	}
}

func TestSomarDiasUteisPulaFimDeSemana(t *testing.T) {
	sexta := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	if d := SomarDiasUteis(sexta, 1); d.Weekday() != time.Monday || d.Day() != 23 {
		t.Fatalf("sexta + 1 dia útil = %s", d)
	}
	if d := SomarDiasUteis(sexta, 0); !d.Equal(sexta) {
		t.Fatalf("prazo zero muda a data: %s", d)
	}
}
