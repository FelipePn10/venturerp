package entity

import "testing"

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
		if got := area.Atende(c.uf, c.cep); got != c.quer {
			t.Fatalf("%s: atende=%v, queria %v", c.nome, got, c.quer)
		}
	}
}

// Região sem faixa de CEP continua atendendo o estado todo.
func TestRegiaoPorEstadoContinuaValendo(t *testing.T) {
	area := &RegiaoAtendida{IsActive: true, State: str("PR")}
	if !area.Atende("pr", "") {
		t.Fatal("região só por UF deveria atender o estado inteiro")
	}
	if area.Atende("SC", "88000000") {
		t.Fatal("região de PR não pode atender SC")
	}
}

func TestRegiaoInativaNaoAtende(t *testing.T) {
	area := &RegiaoAtendida{IsActive: false, State: str("PR")}
	if area.Atende("PR", "") {
		t.Fatal("região inativa não atende")
	}
}
