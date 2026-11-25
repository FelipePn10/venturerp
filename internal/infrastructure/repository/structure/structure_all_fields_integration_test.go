//go:build integration

package structure_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure_query/service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	structuremapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure_query"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure_query"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestStructureAllFieldsMultiLevelAndHistoryRoundTrip garante que todos os
// campos, indicadores, descrições sem PDM, navegação multinível e rótulos do
// histórico funcionam de ponta a ponta e isolados entre empresas.
func TestStructureAllFieldsMultiLevelAndHistoryRoundTrip(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	raiz := int64(1_760_000_000 + testutil.UniqueCode()%10_000_000)
	subconjunto := raiz + 1
	insumo := raiz + 2

	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM item_structure_history WHERE parent_code = ANY($1)`, []int64{raiz, subconjunto})
		_, _ = pool.Exec(base, `DELETE FROM item_structures WHERE parent_code = ANY($1)`, []int64{raiz, subconjunto})
		_, _ = pool.Exec(base, `DELETE FROM items WHERE code = ANY($1)`, []int64{raiz, subconjunto, insumo})
	})
	actor := testutil.Actor(t, pool)

	testutil.SeedItem(t, pool, a.Ctx, raiz, actor)
	testutil.SeedItem(t, pool, a.Ctx, subconjunto, actor)
	testutil.SeedItem(t, pool, a.Ctx, insumo, actor)

	// Simula item sem descrição técnica do PDM, apenas com `name` preenchido.
	if _, err := pool.Exec(base,
		`UPDATE items SET pdm_description_technique = '', name = 'INSUMO CORTADO ESPECIAL' WHERE code = $1`,
		insumo,
	); err != nil {
		t.Fatalf("ajustando nome do insumo: %v", err)
	}

	q := sqlc.New(pool)
	repo := structure.NewItemStructureRepository(q).WithHistory(pool)
	queryRepo := structure_query.NewStructureQueryRepository(q)
	resolver := service.NewResolver(queryRepo)

	inicio := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	fim := time.Date(2027, 12, 20, 0, 0, 0, 0, time.UTC)
	qtyFormula := "COMPRIMENTO * 2"
	lossFormula := "LARGURA * 0.1"
	notasIniciais := "Montagem inicial"
	almox := int64(11)
	almoxLinha := int64(12)
	centro := int64(21)

	// Nível 1: raiz → subconjunto (com todos os indicadores e campos)
	compNivel1 := &entity.ItemStructure{
		ParentCode:             raiz,
		ChildCode:              subconjunto,
		Quantity:               decimal.NewFromFloat(2.5),
		QuantityStockUOM:       decimal.NewFromFloat(2.5),
		ConversionFactor:       decimal.NewFromInt(1),
		StockUnitOfMeasurement: "UN",
		UnitOfMeasurement:      "UN",
		Health:                 "FANTASMA",
		LossPercentage:         decimal.NewFromFloat(3.5),
		LossFormula:            &lossFormula,
		QuantityFormula:        &qtyFormula,
		QuantityRounding:       "UP",
		QuantityScale:          3,
		Sequence:               10,
		Notes:                  &notasIniciais,
		StartDate:              &inicio,
		EndDate:                &fim,
		IsFixedQty:             true,
		Inherit:                true,
		SubstituteGroup:        1,
		SubstitutePriority:     2,
		WarehouseCode:          &almox,
		LineWarehouseCode:      &almoxLinha,
		SetupLoss:              0.75,
		CostLossType:           "PERCENTUAL",
		CostLoss:               4.5,
		CostCenterCode:         &centro,
		IsCriticalMPS:          true,
		GeneratesInspection:    true,
		CustomerSupplied:       true,
		IsActive:               true,
		CreatedBy:              actor,
	}
	if _, err := repo.Create(a.Ctx, compNivel1); err != nil {
		t.Fatalf("criando nível 1: %v", err)
	}

	// Nível 2: subconjunto → insumo (para validar árvore multinível e fallback de name)
	if _, err := repo.Create(a.Ctx, &entity.ItemStructure{
		ParentCode:        subconjunto,
		ChildCode:         insumo,
		Quantity:          decimal.NewFromInt(4),
		UnitOfMeasurement: "UN",
		Health:            "ATIVO",
		Sequence:          10,
		IsActive:          true,
		CreatedBy:         actor,
	}); err != nil {
		t.Fatalf("criando nível 2: %v", err)
	}

	// Resolve árvore completa como a VENT0210 faz em /api/items/structure/resolve/{code}
	nodes, err := resolver.ResolveGeneric(a.Ctx, raiz, 1, make(map[int64]bool))
	if err != nil {
		t.Fatalf("resolvendo árvore multinível: %v", err)
	}
	mapped := structuremapper.MapNodes(nodes)
	if len(mapped) != 1 || len(mapped[0].Children) != 1 {
		t.Fatalf("árvore multinível esperada com 2 níveis (hasChildren=true no nível 1), veio: %+v", mapped)
	}

	c1 := mapped[0].Component
	if !c1.Inherit || !c1.IsFixedQty || !c1.IsCriticalMPS || !c1.GeneratesInspection || !c1.CustomerSupplied {
		t.Fatalf("indicadores perdidos no resolve da árvore: %+v", c1)
	}
	if c1.WarehouseCode == nil || *c1.WarehouseCode != almox ||
		c1.LineWarehouseCode == nil || *c1.LineWarehouseCode != almoxLinha ||
		c1.CostCenterCode == nil || *c1.CostCenterCode != centro {
		t.Fatalf("almoxarifados/centro de custo perdidos no resolve: %+v", c1)
	}
	if c1.SetupLoss != 0.75 || c1.CostLossType != "PERCENTUAL" || c1.CostLoss != 4.5 {
		t.Fatalf("perdas de setup/custo perdidas no resolve: %+v", c1)
	}
	if c1.StockUnitOfMeasurement == "" || c1.ConversionFactor <= 0 {
		t.Fatalf("unidade/fator de estoque perdidos no resolve: %+v", c1)
	}

	// Nível 2 deve trazer o `name` do insumo cujo `pdm_description_technique` está vazio.
	c2 := mapped[0].Children[0].Component
	if c2.ChildDescription != "INSUMO CORTADO ESPECIAL" {
		t.Fatalf("fallback de descrição sem PDM falhou: veio %q", c2.ChildDescription)
	}

	// Atualiza invertendo indicadores e observações para validar Update + Histórico
	notasNovas := "Montagem revisada"
	compNivel1.Inherit = false
	compNivel1.CustomerSupplied = false
	compNivel1.IsCriticalMPS = false
	compNivel1.GeneratesInspection = false
	compNivel1.Notes = &notasNovas
	compNivel1.Sequence = 20
	if _, err := repo.Update(a.Ctx, compNivel1); err != nil {
		t.Fatalf("atualizando nível 1: %v", err)
	}

	// Empresa B não enxerga nada da árvore nem do histórico da empresa A
	if nodesB, err := resolver.ResolveGeneric(b.Ctx, raiz, 1, make(map[int64]bool)); err != nil || len(nodesB) != 0 {
		t.Fatalf("empresa B resolveu árvore da empresa A: %+v (%v)", nodesB, err)
	}

	hist, err := repo.ListByParent(a.Ctx, raiz, 20)
	if err != nil || len(hist) < 2 {
		t.Fatalf("histórico deveria conter inclusão e alteração, veio: %+v (%v)", hist, err)
	}
	var alt *int
	for i := range hist {
		if hist[i].Action == "ALTERACAO" {
			alt = &i
			break
		}
	}
	if alt == nil {
		t.Fatalf("registro de ALTERACAO não encontrado no histórico: %+v", hist)
	}
	camposVistos := map[string]bool{}
	for _, ch := range hist[*alt].Changes {
		if strings.Contains(ch.Field, "_") {
			t.Fatalf("chave interna sem tradução vazou no histórico: %q", ch.Field)
		}
		camposVistos[ch.Field] = true
	}
	for _, esperado := range []string{
		"Herdar máscara do pai",
		"Fornecido pelo cliente",
		"Observações",
		"Crítico para o plano mestre",
		"Gera inspeção",
		"Posição",
	} {
		if !camposVistos[esperado] {
			t.Fatalf("campo %q deveria constar nas diferenças do histórico, vieram: %+v", esperado, hist[*alt].Changes)
		}
	}
}
