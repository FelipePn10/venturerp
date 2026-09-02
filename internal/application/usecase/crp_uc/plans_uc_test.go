package crp_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// fakePlanCatalog imita o catálogo de planos: o plano 1 tem carga calculada, o
// plano 2 ainda não, e o plano 99 pertence a outra empresa.
type fakePlanCatalog struct {
	lastOnlyCalculated bool
	err                error
}

func (c *fakePlanCatalog) ListCRPPlans(_ context.Context, enterpriseID int64, onlyCalculated bool) ([]sqlc.CRPPlanRow, error) {
	c.lastOnlyCalculated = onlyCalculated
	if c.err != nil {
		return nil, c.err
	}
	when := "2026-09-01T10:00:00-03:00"
	all := []sqlc.CRPPlanRow{
		{PlanCode: 1, Name: "Plano setembro", TotalEntries: 12, OverloadCount: 3, MaxLoadPct: 137.5, LastCalculated: &when},
		{PlanCode: 2, Name: "Plano rascunho", TotalEntries: 0},
	}
	if onlyCalculated {
		return all[:1], nil
	}
	return all, nil
}

func (c *fakePlanCatalog) CRPPlanBelongsToEnterprise(_ context.Context, planCode, enterpriseID int64) (bool, error) {
	return planCode != 99, nil
}

func tenantCtx(enterpriseID int64) context.Context {
	return context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
}

// O modal precisa distinguir o plano já calculado do que ainda devolveria uma
// exportação vazia.
func TestListPlans_FlagsCalculatedPlans(t *testing.T) {
	uc := New(nil).WithPlanCatalog(&fakePlanCatalog{})
	got, err := uc.ListPlans(tenantCtx(7), false)
	if err != nil {
		t.Fatalf("listagem recusada: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("planos = %d, quer 2", len(got))
	}
	if !got[0].Calculated || got[0].OverloadCount != 3 || got[0].MaxLoadPct != 137.5 {
		t.Fatalf("plano calculado: %+v", got[0])
	}
	if got[1].Calculated || got[1].LastCalculated != nil {
		t.Fatalf("plano sem carga marcado como calculado: %+v", got[1])
	}
}

func TestListPlans_OnlyCalculatedIsForwarded(t *testing.T) {
	catalog := &fakePlanCatalog{}
	uc := New(nil).WithPlanCatalog(catalog)
	got, err := uc.ListPlans(tenantCtx(7), true)
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.lastOnlyCalculated {
		t.Fatal("filtro only_calculated não chegou ao catálogo")
	}
	if len(got) != 1 || !got[0].Calculated {
		t.Fatalf("planos = %+v", got)
	}
}

func TestListPlans_RequiresTenant(t *testing.T) {
	uc := New(nil).WithPlanCatalog(&fakePlanCatalog{})
	if _, err := uc.ListPlans(context.Background(), false); err == nil {
		t.Fatal("listagem sem empresa autenticada foi aceita")
	}
}

// Sem catálogo configurado, a tela apenas não recebe opções — nada quebra.
func TestListPlans_WithoutCatalogReturnsEmpty(t *testing.T) {
	got, err := New(nil).ListPlans(tenantCtx(7), false)
	if err != nil || len(got) != 0 {
		t.Fatalf("planos = %v err=%v, quer lista vazia", got, err)
	}
}

func TestListPlans_PropagatesCatalogFailure(t *testing.T) {
	uc := New(nil).WithPlanCatalog(&fakePlanCatalog{err: errors.New("banco indisponível")})
	if _, err := uc.ListPlans(tenantCtx(7), false); err == nil {
		t.Fatal("falha do catálogo foi engolida")
	}
}

// A carga de um plano de outra empresa não pode ser lida.
func TestAssertPlanTenant_BlocksForeignPlan(t *testing.T) {
	uc := New(nil).WithPlanCatalog(&fakePlanCatalog{})
	if err := uc.assertPlanTenant(tenantCtx(7), 1); err != nil {
		t.Fatalf("plano da própria empresa recusado: %v", err)
	}
	err := uc.assertPlanTenant(tenantCtx(7), 99)
	if _, ok := errorsuc.AsNotFound(err); !ok {
		t.Fatalf("esperado NotFoundError, veio %T (%v)", err, err)
	}
}
