package production_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type CloseProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// SettleUC is optional. When set, closing the order also settles its actual
	// cost (material + conversion) and variance against the standard, so a closed
	// OF always carries its real cost without a manual step.
	SettleUC *SettleProductionCostUseCase
}

func (uc *CloseProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CloseProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	order, err := uc.Repo.Close(ctx, dto.ID)
	if err != nil {
		return nil, err
	}

	// O encerramento vale (é o estado oficial da ordem). Se a apuração do custo
	// real falhar, a OF fica marcada com "custo pendente" e o motivo — visível
	// até "Apurar custo" dar certo. Antes a falha era ignorada e a OF ficava sem
	// custo sem ninguém saber.
	if uc.SettleUC != nil {
		if _, errCusto := uc.SettleUC.Execute(ctx, dto.ID); errCusto != nil {
			motivo := "o custo real não foi apurado no encerramento: " + errCusto.Error()
			if m, ok := uc.Repo.(custoPendenteRepo); ok {
				if err := m.MarcarCustoPendente(ctx, dto.ID, &motivo); err != nil {
					motivo += "; e a pendência não pôde ser registrada: " + err.Error()
				}
			}
			order.CostPending = &motivo
		}
	}

	return order, nil
}

// custoPendenteRepo: repositórios que guardam o motivo da OF sem custo apurado.
type custoPendenteRepo interface {
	MarcarCustoPendente(ctx context.Context, id int64, motivo *string) error
	CustoPendente(ctx context.Context, id int64) (*string, error)
}
