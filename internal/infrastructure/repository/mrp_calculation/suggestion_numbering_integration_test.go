//go:build integration

package mrp_calculation

import (
	"context"
	"errors"
	"github.com/shopspring/decimal"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrprepository "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Número de ordem ocupado — por ordem planejada ativa ou por sugestão de outro
// plano — volta como ErrOrderNumberInUse (o cálculo pula para o próximo), e a
// sugestão firmada sai da lista com a ordem planejada que gerou.
func TestSuggestionNumberInUseAndFirmedOrder(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM planned_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM mrp_planned_suggestions WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_plans WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	planoA, planoB := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'A',$3,$4),($2,'B',$3,$4)`, planoA, planoB, actor, a.ID)
	numero := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,status,demand_type,demand_code,need_date,created_by,enterprise_id,is_active)
		VALUES($1,$2,1,'PRODUCTION','PLANNED','INDEPENDENT',0,$3::date,$4,$5,true)`, numero, item, time.Now(), actor, a.ID)

	repo := NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool)
	sugestao := func(plano, n int64) *entity.PlannedOrderSuggestion {
		return &entity.PlannedOrderSuggestion{OrderNumber: &n, PlanCode: plano, ItemCode: item, Quantity: decimal.NewFromFloat(1), NeedDate: time.Now(), OrderType: "FABRICACAO", DemandType: "INDEPENDENTE"}
	}
	if _, err := repo.CreatePlannedOrderSuggestion(a.Ctx, sugestao(planoA, numero)); !errors.Is(err, mrprepository.ErrOrderNumberInUse) {
		t.Fatalf("número de ordem planejada ativa: %v, esperado ErrOrderNumberInUse", err)
	}
	livre := numero + 1
	criada, err := repo.CreatePlannedOrderSuggestion(a.Ctx, sugestao(planoA, livre))
	if err != nil {
		t.Fatalf("número livre recusado: %v", err)
	}
	// Mesmo número em outro plano: o índice único recusaria e abortaria a
	// transação do cálculo; agora volta como número ocupado.
	if _, err := repo.CreatePlannedOrderSuggestion(a.Ctx, sugestao(planoB, livre)); !errors.Is(err, mrprepository.ErrOrderNumberInUse) {
		t.Fatalf("número de sugestão de outro plano: %v, esperado ErrOrderNumberInUse", err)
	}

	lista, err := repo.ListSuggestionsByPlan(a.Ctx, planoA)
	if err != nil || len(lista) != 1 || lista[0].PlannedOrderCode != nil {
		t.Fatalf("sugestão pendente: %+v (%v)", lista, err)
	}
	var ordem int64
	if err := pool.QueryRow(base, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,status,demand_type,demand_code,need_date,created_by,enterprise_id,is_active,mrp_suggestion_code,is_firm)
		VALUES($1,$2,1,'PRODUCTION','RELEASED','INDEPENDENT',0,$3::date,$4,$5,true,$6,true) RETURNING code`, livre, item, time.Now(), actor, a.ID, criada.Code).Scan(&ordem); err != nil {
		t.Fatal(err)
	}
	lista, err = repo.ListSuggestionsByPlan(a.Ctx, planoA)
	if err != nil || len(lista) != 1 || lista[0].PlannedOrderCode == nil || *lista[0].PlannedOrderCode != ordem {
		t.Fatalf("sugestão firmada sem a ordem %d: %+v (%v)", ordem, lista, err)
	}
}
