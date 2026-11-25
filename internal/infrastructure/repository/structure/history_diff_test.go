package structure

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/shopspring/decimal"
)

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

// Garante que todos os 28 campos serializados em instantaneo(...) possuem
// rótulo traduzido em português, sem vazar chaves snake_case na interface.
func TestInstantaneoERotulosCobremTodosOsCamposSemChaveCrua(t *testing.T) {
	f1, f2 := "COMPRIMENTO * 1", "COMPRIMENTO * 2"
	lf1, lf2 := "LARGURA * 0.01", "LARGURA * 0.02"
	n1, n2 := "Nota A", "Nota B"
	m1, m2 := "1.94M", "2.20M"
	w1, w2 := int64(1), int64(2)
	lw1, lw2 := int64(3), int64(4)
	cc1, cc2 := int64(10), int64(20)
	d1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	d4 := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	a := &entity.ItemStructure{
		Quantity:            decimal.NewFromInt(1),
		QuantityFormula:     &f1,
		QuantityRounding:    "NONE",
		QuantityScale:       2,
		UnitOfMeasurement:   "UN",
		LossPercentage:      decimal.NewFromInt(1),
		LossFormula:         &lf1,
		SetupLoss:           0.5,
		CostLossType:        "PERCENTUAL",
		CostLoss:            1.5,
		Health:              "ATIVO",
		Sequence:            10,
		StartDate:           &d1,
		EndDate:             &d3,
		IsCoproduct:         false,
		IsFixedQty:          false,
		Inherit:             false,
		SubstituteGroup:     0,
		SubstitutePriority:  1,
		WarehouseCode:       &w1,
		LineWarehouseCode:   &lw1,
		CostCenterCode:      &cc1,
		IsCriticalMPS:       false,
		GeneratesInspection: false,
		CustomerSupplied:    false,
		Notes:               &n1,
		ParentMask:          &m1,
		IsActive:            true,
	}
	b := &entity.ItemStructure{
		Quantity:            decimal.NewFromInt(2),
		QuantityFormula:     &f2,
		QuantityRounding:    "UP",
		QuantityScale:       4,
		UnitOfMeasurement:   "KG",
		LossPercentage:      decimal.NewFromInt(5),
		LossFormula:         &lf2,
		SetupLoss:           1.25,
		CostLossType:        "QUANTIDADE",
		CostLoss:            3.0,
		Health:              "FANTASMA",
		Sequence:            20,
		StartDate:           &d2,
		EndDate:             &d4,
		IsCoproduct:         true,
		IsFixedQty:          true,
		Inherit:             true,
		SubstituteGroup:     2,
		SubstitutePriority:  3,
		WarehouseCode:       &w2,
		LineWarehouseCode:   &lw2,
		CostCenterCode:      &cc2,
		IsCriticalMPS:       true,
		GeneratesInspection: true,
		CustomerSupplied:    true,
		Notes:               &n2,
		ParentMask:          &m2,
		IsActive:            false,
	}

	var antes, depois map[string]any
	if err := json.Unmarshal(instantaneo(a), &antes); err != nil {
		t.Fatalf("deserializando antes: %v", err)
	}
	if err := json.Unmarshal(instantaneo(b), &depois); err != nil {
		t.Fatalf("deserializando depois: %v", err)
	}

	mudou := comparar(antes, depois)
	if len(mudou) != 28 {
		t.Fatalf("mudanças = %d, esperado 28 campos cobertos pelo instantâneo: %+v", len(mudou), mudou)
	}
	for _, m := range mudou {
		if strings.Contains(m.Field, "_") {
			t.Fatalf("rótulo sem tradução vazou no histórico: %q", m.Field)
		}
	}
}
