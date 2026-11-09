package unit_of_measure_uc

import (
	"context"
	"strings"
	"testing"
)

type catalogo map[string]bool

func (c catalogo) UnidadesForaDoCatalogo(_ context.Context, codigos ...string) ([]string, error) {
	var fora []string
	for _, x := range codigos {
		if !c[x] {
			fora = append(fora, x)
		}
	}
	return fora, nil
}

func TestConferirNoCatalogo(t *testing.T) {
	ctx := context.Background()
	cat := catalogo{"KG": true, "UN": true}
	kg, kgs, vazio := " kg ", "KGS", "  "
	if err := ConferirNoCatalogo(ctx, cat, "unidade do preço", &kg, nil, &vazio); err != nil {
		t.Fatalf("KG (com espaço e minúscula), nil e vazio deveriam passar: %v", err)
	}
	err := ConferirNoCatalogo(ctx, cat, "unidade do preço", &kgs)
	if err == nil || !strings.Contains(err.Error(), "unidade do preço") || !strings.Contains(err.Error(), "KGS") {
		t.Fatalf("esperava recusa de KGS citando o campo, veio %v", err)
	}
	if err := ConferirNoCatalogo(ctx, nil, "x", &kgs); err != nil {
		t.Fatalf("sem catálogo não confere: %v", err)
	}
	if u := NormalizarUnidade(&kg); u == nil || *u != "KG" {
		t.Fatalf("normalização: %v", u)
	}
	if NormalizarUnidade(&vazio) != nil || NormalizarUnidade(nil) != nil {
		t.Fatal("vazio e nil viram nil")
	}
}
