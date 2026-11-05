//go:build integration

package mrp_calculation_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Estas consultas citavam colunas que não existem: o erro era engolido e o MRP
// seguia sem cartões kanban e sem os parâmetros extras do item.
func TestConsultasDoMRPContraOBanco(t *testing.T) {
	q, pool := testutil.Queries(t)
	var empresa int64
	if err := pool.QueryRow(context.Background(), "SELECT MIN(id) FROM enterprise").Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{EnterpriseID: empresa})
	repo := mrprepo.NewMRPCalculationRepositorySQLC(q, pool)
	if _, err := repo.ListKanbanCards(ctx); err != nil {
		t.Errorf("cartões kanban: %v", err)
	}
	if _, err := repo.ListItemPlanningExtras(ctx); err != nil {
		t.Errorf("parâmetros extras: %v", err)
	}
	if err := repo.UpdatePlannedOrderPriority(ctx, -1, "A"); err != nil {
		t.Errorf("prioridade da sugestão: %v", err)
	}
}
