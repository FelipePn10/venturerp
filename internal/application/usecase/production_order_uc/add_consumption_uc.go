package production_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type AddConsumptionUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// StockRepo is optional. When set, registering a consumption also posts an
	// OUT stock movement (which updates the balance) for the consumed item.
	StockRepo stockrepo.StockRepository
	// Transacao é opcional na montagem: sem ela cada gravação vale sozinha.
	Transacao EmTransacaoApontamento
}

func (uc *AddConsumptionUseCase) Execute(
	ctx context.Context,
	dto request.AddConsumptionDTO,
) (*entity.ProductionConsumption, error) {
	if !uc.Auth.CanReportProduction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	dto.CreatedBy = actor

	consumptionDate := datetime.ParseDateOrDefault(dto.ConsumptionDate, time.Now())

	consumption := &entity.ProductionConsumption{
		ProductionOrderID: dto.ProductionOrderID,
		AppointmentID:     dto.AppointmentID,
		ItemCode:          dto.ItemCode,
		ConsumedQty:       dto.ConsumedQty,
		WarehouseID:       dto.WarehouseID,
		Lot:               dto.Lot,
		ConsumptionDate:   consumptionDate,
		Notes:             dto.Notes,
		CreatedBy:         dto.CreatedBy,
	}

	// O consumo e a saída valem juntos. Antes a saída barrada por falta de saldo
	// deixava o consumo gravado: a pessoa repetia e a OF ficava com consumo em dobro.
	emTransacao := uc.Transacao
	if emTransacao == nil {
		emTransacao = func(ctx context.Context, fn func(ApontamentoRepos) error) error {
			return fn(ApontamentoRepos{Ordens: uc.Repo, Estoque: uc.StockRepo})
		}
	}
	var saved *entity.ProductionConsumption
	err = emTransacao(ctx, func(r ApontamentoRepos) error {
		var err error
		if saved, err = r.Ordens.AddConsumption(ctx, consumption); err != nil {
			return err
		}
		// Post the OUT movement for the consumed input so the warehouse balance is
		// reduced automatically. Requires a target warehouse on the consumption.
		if r.Estoque == nil || saved.WarehouseID == nil {
			return nil
		}
		refType := stockentity.ReferenceTypeProductionOrder
		refCode := saved.ProductionOrderID
		_, err = r.Estoque.CreateMovement(ctx, &stockentity.StockMovement{
			ItemCode:      saved.ItemCode,
			WarehouseID:   *saved.WarehouseID,
			MovementType:  stockentity.MovementTypeOut,
			Quantity:      saved.ConsumedQty,
			ReferenceType: &refType,
			ReferenceCode: &refCode,
			Lot:           saved.Lot,
			Notes:         saved.Notes,
			CreatedBy:     saved.CreatedBy,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}
