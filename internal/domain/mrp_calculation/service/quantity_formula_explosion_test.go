package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

func formulaPtr(s string) *string { return &s }

// A fórmula de quantidade cadastrada na estrutura precisa chegar até a demanda
// dependente — é isso que faz a OF consumir a metragem real da configuração.
func TestExplodeWithVars_UsesQuantityFormula(t *testing.T) {
	bomMap := map[int64][]*structentity.ItemStructure{
		1: {
			// Perfil: perímetro do quadro em metros.
			{ChildCode: 2, Quantity: 1, QuantityFormula: formulaPtr("2*(COMPRIMENTO/1000)+2*(PROFUNDIDADE/1000)")},
			// Parafuso: quantidade fixa, sem fórmula.
			{ChildCode: 3, Quantity: 8},
		},
	}
	vars := map[string]float64{"COMPRIMENTO": 1200, "PROFUNDIDADE": 600}

	inputs := explodeFromBOMWithVars(bomMap, 1, "1200#600", 10, 1, 1, vars, time.Time{})

	got := map[int64]float64{}
	for _, in := range inputs {
		got[in.ItemCode] = in.Quantity
	}
	if math.Abs(got[2]-36) > 1e-9 {
		t.Errorf("perfil = %v, quer 36 (3,6 m × 10 peças)", got[2])
	}
	if got[3] != 80 {
		t.Errorf("parafuso = %v, quer 80 (8 × 10)", got[3])
	}
}

// Sem as variáveis respondidas a explosão não pode falhar: vale a quantidade
// fixa cadastrada como reserva.
func TestExplodeWithVars_FallsBackToFixedQuantity(t *testing.T) {
	bomMap := map[int64][]*structentity.ItemStructure{
		1: {{ChildCode: 2, Quantity: 4, QuantityFormula: formulaPtr("COMPRIMENTO/1000")}},
	}
	inputs := explodeFromBOMWithVars(bomMap, 1, "", 10, 1, 1, nil, time.Time{})
	if len(inputs) != 1 || inputs[0].Quantity != 40 {
		t.Fatalf("explosão = %+v, quer 40 (4 × 10)", inputs)
	}
}

// A perda incide sobre o resultado da fórmula, não sobre a quantidade nominal.
func TestExplodeWithVars_LossAppliesOverFormulaResult(t *testing.T) {
	bomMap := map[int64][]*structentity.ItemStructure{
		1: {{ChildCode: 2, Quantity: 1, LossPercentage: 10, QuantityFormula: formulaPtr("COMPRIMENTO/1000")}},
	}
	inputs := explodeFromBOMWithVars(bomMap, 1, "2000", 5, 1, 1, map[string]float64{"COMPRIMENTO": 2000}, time.Time{})
	if len(inputs) != 1 {
		t.Fatalf("explosão = %+v", inputs)
	}
	// 2 m × 5 peças × 1,10 = 11
	if math.Abs(inputs[0].Quantity-11) > 1e-9 {
		t.Fatalf("quantidade = %v, quer 11", inputs[0].Quantity)
	}
}

type stubMaskVars struct {
	vars map[string]float64
	err  error
	hits int
}

func (s *stubMaskVars) GetMaskAnswersWithNames(context.Context, int64, string) (map[string]float64, error) {
	s.hits++
	return s.vars, s.err
}

// O cache evita reconsultar as respostas do mesmo par item/máscara.
func TestMaskVarCache_ReadsOncePerItemAndMask(t *testing.T) {
	stub := &stubMaskVars{vars: map[string]float64{"COMPRIMENTO": 1000}}
	cache := newMaskVarCache(stub)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if got := cache.vars(ctx, 1, "1000"); got["COMPRIMENTO"] != 1000 {
			t.Fatalf("variáveis = %v", got)
		}
	}
	if stub.hits != 1 {
		t.Fatalf("consultas ao repositório = %d, quer 1", stub.hits)
	}
	cache.vars(ctx, 2, "1000")
	if stub.hits != 2 {
		t.Fatalf("consultas para outro item = %d, quer 2", stub.hits)
	}
}

func TestMaskVarCache_IsNilSafe(t *testing.T) {
	if got := newMaskVarCache(nil).vars(context.Background(), 1, "x"); got != nil {
		t.Fatalf("sem leitor as variáveis devem ser nulas, veio %v", got)
	}
	// Item sem máscara não tem configuração para consultar.
	stub := &stubMaskVars{err: errors.New("indisponível")}
	if got := newMaskVarCache(stub).vars(context.Background(), 1, ""); got != nil || stub.hits != 0 {
		t.Fatalf("máscara vazia consultou o repositório: %v hits=%d", got, stub.hits)
	}
	// Falha de leitura não derruba a explosão.
	if got := newMaskVarCache(stub).vars(context.Background(), 1, "m"); got != nil {
		t.Fatalf("erro do repositório deveria virar variáveis nulas, veio %v", got)
	}
}
