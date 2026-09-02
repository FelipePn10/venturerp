package entity

import (
	"math"
	"testing"
)

func strPtr(s string) *string { return &s }

func nearly(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

// O exemplo da própria especificação: perímetro em metros de um quadro cujas
// medidas são respondidas em milímetros no configurador.
func TestResolvedQuantity_PerimeterFormula(t *testing.T) {
	s := &ItemStructure{
		Quantity:        1,
		QuantityFormula: strPtr("2*(COMPRIMENTO/1000)+2*(PROFUNDIDADE/1000)"),
	}
	got, applied := s.ResolvedQuantity(map[string]float64{"COMPRIMENTO": 1200, "PROFUNDIDADE": 600})
	if !applied {
		t.Fatal("fórmula não foi aplicada")
	}
	if !nearly(got, 3.6) {
		t.Fatalf("quantidade = %v, quer 3.6", got)
	}
}

func TestResolvedQuantity_FallsBackToFixedQuantity(t *testing.T) {
	s := &ItemStructure{Quantity: 4, QuantityFormula: strPtr("2*COMPRIMENTO")}
	// Sem a variável respondida, a fórmula não é avaliável.
	got, applied := s.ResolvedQuantity(nil)
	if applied {
		t.Fatal("fórmula aplicada sem as variáveis")
	}
	if got != 4 {
		t.Fatalf("quantidade = %v, quer a fixa 4", got)
	}
}

func TestResolvedQuantity_NoFormulaKeepsFixedQuantity(t *testing.T) {
	s := &ItemStructure{Quantity: 2.5}
	got, applied := s.ResolvedQuantity(map[string]float64{"COMPRIMENTO": 1000})
	if applied || got != 2.5 {
		t.Fatalf("quantidade = %v applied=%v, quer 2.5/false", got, applied)
	}
}

func TestResolvedQuantity_Rounding(t *testing.T) {
	cases := []struct {
		mode  string
		scale int16
		want  float64
	}{
		{RoundNone, 4, 1.3333333333333333},
		{RoundUp, 2, 1.34},
		{RoundDown, 2, 1.33},
		{RoundNearest, 2, 1.33},
		{RoundUp, 0, 2},
		{RoundDown, 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			s := &ItemStructure{
				Quantity:         1,
				QuantityFormula:  strPtr("4/3"),
				QuantityRounding: tc.mode,
				QuantityScale:    tc.scale,
			}
			got, applied := s.ResolvedQuantity(nil)
			if !applied {
				t.Fatal("fórmula constante deveria ser avaliável")
			}
			if !nearly(got, tc.want) {
				t.Fatalf("quantidade = %v, quer %v", got, tc.want)
			}
		})
	}
}

// A perda continua sendo aplicada sobre o resultado da fórmula.
func TestEffectiveQuantityWith_AppliesLossOverFormula(t *testing.T) {
	s := &ItemStructure{
		Quantity:        1,
		LossPercentage:  10,
		QuantityFormula: strPtr("COMPRIMENTO/1000"),
	}
	got := s.EffectiveQuantityWith(map[string]float64{"COMPRIMENTO": 2000})
	if !nearly(got, 2.2) {
		t.Fatalf("quantidade efetiva = %v, quer 2.2", got)
	}
}

func TestFormulaVariables_ListsQuestionsInOrder(t *testing.T) {
	s := &ItemStructure{QuantityFormula: strPtr("2*(COMPRIMENTO/1000)+2*(PROFUNDIDADE/1000)+COMPRIMENTO")}
	got := s.FormulaVariables()
	want := []string{"COMPRIMENTO", "PROFUNDIDADE"}
	if len(got) != len(want) {
		t.Fatalf("variáveis = %v, quer %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("variáveis = %v, quer %v", got, want)
		}
	}
}

func TestResolvedQuantity_NegativeResultFallsBack(t *testing.T) {
	s := &ItemStructure{Quantity: 3, QuantityFormula: strPtr("0-LARGURA")}
	got, applied := s.ResolvedQuantity(map[string]float64{"LARGURA": 5})
	if applied || got != 3 {
		t.Fatalf("resultado negativo aceito: %v applied=%v", got, applied)
	}
}
