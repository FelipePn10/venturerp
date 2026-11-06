//go:build integration

package planned_order_uc_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	planned "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

type authPlanejada struct{ ports.AuthService }

func (authPlanejada) CanCreatePlannedOrder(context.Context) bool { return true }

// A reposição gera várias ordens, e duas pessoas podem gerar juntas: a numeração é o
// maior + 1, e quem perde a corrida toma o próximo em vez de receber erro.
func TestOrdensPlanejadasSimultaneasNaoDisputamONumero(t *testing.T) {
	q, pool := testutil.Queries(t)
	ator := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(context.Background(), `DELETE FROM planned_orders WHERE enterprise_id=$1`, a.ID)
	})
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	uc := &planned_order_uc.CreatePlannedOrderUseCase{Repo: planned.NewPlannedOrderRepositorySQLC(q), Auth: authPlanejada{}}
	const total = 6
	erros := make([]error, total)
	var fila sync.WaitGroup
	for i := range total {
		fila.Add(1)
		go func() {
			defer fila.Done()
			_, erros[i] = uc.Execute(a.Ctx, request.CreatePlannedOrderDTO{ItemCode: item, Quantity: decimal.NewFromInt(1), OrderType: string(types.OrderProduction),
				NeedDate: time.Now().Format("2006-01-02"), CreatedBy: ator})
		}()
	}
	fila.Wait()
	for i, err := range erros {
		if err != nil {
			t.Fatalf("ordem %d: %v", i, err)
		}
	}
	var ordens, numeros int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*), COUNT(DISTINCT order_number) FROM planned_orders WHERE enterprise_id=$1`, a.ID).Scan(&ordens, &numeros); err != nil {
		t.Fatal(err)
	}
	if ordens != total || numeros != total {
		t.Fatalf("%d ordens com %d números", ordens, numeros)
	}
}
