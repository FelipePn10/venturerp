package structure

import "testing"

// O histórico só é útil se mostrar o que mudou, em português. Guardar dois
// blocos de JSON e mandar o usuário comparar não resolve.
func TestCompararMostraSomenteOqueMudouComRotuloEmPortugues(t *testing.T) {
	antes := map[string]any{"quantidade": 2.0, "situacao": "ATIVO", "coproduto": false}
	depois := map[string]any{"quantidade": 3.0, "situacao": "ATIVO", "coproduto": true}

	mudou := comparar(antes, depois)
	if len(mudou) != 2 {
		t.Fatalf("mudanças = %d, esperado 2 (quantidade e co-produto): %+v", len(mudou), mudou)
	}
	porCampo := map[string]FieldChangeLike{}
	for _, m := range mudou {
		porCampo[m.Field] = FieldChangeLike{m.Before, m.After}
	}
	if c, ok := porCampo["Quantidade"]; !ok || c.Before != "2" || c.After != "3" {
		t.Fatalf("quantidade mal formatada: %+v", porCampo)
	}
	if c, ok := porCampo["Co-produto"]; !ok || c.Before != "Não" || c.After != "Sim" {
		t.Fatalf("booleano não virou Sim/Não: %+v", porCampo)
	}
	if _, ok := porCampo["Situação na estrutura"]; ok {
		t.Fatal("campo inalterado apareceu no histórico")
	}
}

type FieldChangeLike struct{ Before, After string }

// Datas ISO viram o formato que o usuário lê.
func TestLegivelFormataDataEValorVazio(t *testing.T) {
	if got := legivel("2026-09-01T00:00:00Z"); got != "01/09/2026" {
		t.Fatalf("data = %q, esperado 01/09/2026", got)
	}
	if got := legivel(nil); got != "—" {
		t.Fatalf("nulo = %q, esperado travessão", got)
	}
	if got := legivel(2.5); got != "2.5" {
		t.Fatalf("decimal = %q", got)
	}
}
