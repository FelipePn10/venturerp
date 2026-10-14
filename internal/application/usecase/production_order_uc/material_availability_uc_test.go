package production_order_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type disponibilidadeAuth struct{ ports.AuthService }

func (disponibilidadeAuth) CanGetSalesOrder(context.Context) bool { return true }

type disponibilidadeRepo struct {
	repository.ProductionOrderRepository
	linhas []entity.MaterialAvailabilityLine
}

func (r disponibilidadeRepo) MaterialAvailability(context.Context, []int64) ([]entity.MaterialAvailabilityLine, error) {
	return r.linhas, nil
}

func (r disponibilidadeRepo) GetByCode(_ context.Context, id int64) (*entity.ProductionOrder, error) {
	if id == 404 {
		return nil, errorsuc.NewNotFoundError("ordem de produção não encontrada")
	}
	return &entity.ProductionOrder{ID: id}, nil
}

func dq(n int64) decimal.Decimal { return decimal.NewFromInt(n) }

// A falta considera o que já foi baixado e o que as OFs da frente vão consumir
// do mesmo saldo.
func TestFaltasDaOFAntesDeLiberar(t *testing.T) {
	repo := disponibilidadeRepo{linhas: []entity.MaterialAvailabilityLine{
		// OF 1: chapa 10, já baixou 4, saldo 20 mas 18 comprometido → 2 livres, falta 4.
		{ProductionOrderID: 1, MaterialID: 1, ItemCode: 10, Required: dq(10), Issued: dq(4), FreeStock: dq(20), CommittedEarlier: dq(18)},
		// OF 1: parafuso coberto.
		{ProductionOrderID: 1, MaterialID: 2, ItemCode: 11, Required: dq(8), FreeStock: dq(100)},
		// OF 1: tinta já baixada por completo — não entra na checagem.
		{ProductionOrderID: 1, MaterialID: 3, ItemCode: 12, Required: dq(1), Issued: dq(1)},
		// OF 2: nada em estoque.
		{ProductionOrderID: 2, MaterialID: 4, ItemCode: 10, Required: dq(5), FreeStock: dq(0)},
		// OF 3: tudo coberto.
		{ProductionOrderID: 3, MaterialID: 5, ItemCode: 11, Required: dq(5), FreeStock: dq(5)},
	}}
	uc := &ProductionMaterialControlUseCase{Repo: repo, Auth: disponibilidadeAuth{}}

	um, err := uc.Availability(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if um.Status != AvailabilityPartial || um.Items != 2 || um.MissingItems != 1 || len(um.Lines) != 2 {
		t.Fatalf("OF 1: %+v", um)
	}
	chapa := um.Lines[0]
	if !chapa.Remaining.Equal(dq(6)) || !chapa.Available.Equal(dq(2)) || !chapa.Shortage.Equal(dq(4)) || chapa.Status != AvailabilityPartial {
		t.Fatalf("chapa: %+v", chapa)
	}

	if _, err := uc.Availability(context.Background(), 404); err == nil {
		t.Fatal("OF inexistente respondeu como se existisse")
	}

	resumo, err := uc.AvailabilitySummary(context.Background(), []int64{1, 2, 3, 4}, false)
	if err != nil {
		t.Fatal(err)
	}
	esperado := []string{AvailabilityPartial, AvailabilityMissing, AvailabilityComplete, AvailabilityNone}
	for i, s := range resumo {
		if s.Status != esperado[i] || s.Lines != nil {
			t.Fatalf("OF %d: %+v, esperado %s sem linhas", s.ProductionOrderID, s, esperado[i])
		}
	}
}
