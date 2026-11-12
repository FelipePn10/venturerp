package entity

import (
	"strings"
	"testing"
)

func TestPlanoRecusaPecaEEstoqueSemAMedidaDoTipoDeCorte(t *testing.T) {
	barra := &CuttingPlan{CutType: CutTypeLinear1D}
	chapa := &CuttingPlan{CutType: CutTypeGuillotine2D}
	contorno := &CuttingPlan{CutType: CutTypeTrueShape2D}

	peca1D := &CuttingPlanPart{Label: "travessa", LengthMM: 500, Quantity: 2}
	peca2D := &CuttingPlanPart{Label: "lateral", WidthMM: 300, HeightMM: 600, Quantity: 2}
	barraDoEstoque := &CuttingStockPiece{LengthMM: 6000, Quantity: 3}
	chapaDoEstoque := &CuttingStockPiece{WidthMM: 1830, HeightMM: 2750, Quantity: 3}
	retalhoSemMedida := &CuttingStockPiece{Quantity: 1, IsRemnant: true}

	casos := []struct {
		nome   string
		plano  *CuttingPlan
		parts  []*CuttingPlanPart
		stock  []*CuttingStockPiece
		trecho string // vazio = aceita
	}{
		{"barra coerente", barra, []*CuttingPlanPart{peca1D}, []*CuttingStockPiece{barraDoEstoque}, ""},
		{"chapa coerente", chapa, []*CuttingPlanPart{peca2D}, []*CuttingStockPiece{chapaDoEstoque}, ""},
		{"contorno coerente", contorno, []*CuttingPlanPart{peca2D}, []*CuttingStockPiece{chapaDoEstoque}, ""},
		{"barra com peça de chapa", barra, []*CuttingPlanPart{peca2D}, []*CuttingStockPiece{barraDoEstoque}, `"lateral" não tem comprimento`},
		{"barra com estoque de chapa", barra, []*CuttingPlanPart{peca1D}, []*CuttingStockPiece{chapaDoEstoque}, "a barra do estoque não tem comprimento"},
		{"chapa com peça de barra", chapa, []*CuttingPlanPart{peca1D}, []*CuttingStockPiece{chapaDoEstoque}, `"travessa" não tem largura e altura`},
		{"chapa com estoque de barra", chapa, []*CuttingPlanPart{peca2D}, []*CuttingStockPiece{barraDoEstoque}, "a chapa do estoque não tem largura e altura"},
		{"retalho sem medida", contorno, []*CuttingPlanPart{peca2D}, []*CuttingStockPiece{retalhoSemMedida}, "o retalho do estoque"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			err := c.plano.ConferirDimensoes(c.parts, c.stock)
			if c.trecho == "" {
				if err != nil {
					t.Fatalf("recusou plano coerente: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.trecho) {
				t.Fatalf("erro = %v, esperado conter %q", err, c.trecho)
			}
		})
	}
}
