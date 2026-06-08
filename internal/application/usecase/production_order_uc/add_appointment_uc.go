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
	structurerepo "github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

type AddAppointmentUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// StructureRepo and StockRepo are optional. When both are set and the
	// appointment carries a backflush warehouse, the BOM components are consumed
	// automatically (OUT) in proportion to the produced quantity.
	StructureRepo structurerepo.ItemStructureRepository
	StockRepo     stockrepo.StockRepository
}

func (uc *AddAppointmentUseCase) Execute(
	ctx context.Context,
	dto request.AddAppointmentDTO,
) (*entity.ProductionAppointment, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	appointmentDate, _ := time.Parse("2006-01-02", dto.AppointmentDate)

	appointment := &entity.ProductionAppointment{
		ProductionOrderID: dto.ProductionOrderID,
		MachineID:         dto.MachineID,
		EmployeeID:        dto.EmployeeID,
		AppointmentDate:   appointmentDate,
		StartTime:         dto.StartTime,
		EndTime:           dto.EndTime,
		ProducedQty:       dto.ProducedQty,
		ScrappedQty:       dto.ScrappedQty,
		ScrapReason:       dto.ScrapReason,
		Notes:             dto.Notes,
		CreatedBy:         dto.CreatedBy,
	}

	saved, err := uc.Repo.AddAppointment(ctx, appointment)
	if err != nil {
		return nil, err
	}

	if uc.shouldBackflush(dto) {
		uc.backflush(ctx, dto, saved)
	}

	return saved, nil
}

func (uc *AddAppointmentUseCase) shouldBackflush(dto request.AddAppointmentDTO) bool {
	return uc.StructureRepo != nil && uc.StockRepo != nil &&
		dto.BackflushWarehouseID != nil && dto.ProducedQty > 0
}

// backflush consumes the BOM components for the produced quantity. Best-effort:
// it never fails the appointment itself.
func (uc *AddAppointmentUseCase) backflush(ctx context.Context, dto request.AddAppointmentDTO, ap *entity.ProductionAppointment) {
	order, err := uc.Repo.GetByCode(ctx, dto.ProductionOrderID)
	if err != nil {
		return
	}

	var children []*structureChild
	var rawErr error
	if order.Mask != "" {
		raw, e := uc.StructureRepo.GetDirectChildrenForMask(ctx, order.ItemCode, order.Mask)
		rawErr = e
		for _, c := range raw {
			children = append(children, &structureChild{code: c.ChildCode, qty: c.Quantity, loss: c.LossPercentage})
		}
	} else {
		raw, e := uc.StructureRepo.GetAllDirectChildren(ctx, order.ItemCode)
		rawErr = e
		for _, c := range raw {
			children = append(children, &structureChild{code: c.ChildCode, qty: c.Quantity, loss: c.LossPercentage})
		}
	}
	if rawErr != nil {
		return
	}

	refType := stockentity.ReferenceTypeProductionOrder
	refCode := dto.ProductionOrderID
	for _, c := range children {
		// Loss formula 1 (default): qty = parentQty × componentQty × (1 + loss/100).
		consumed := dto.ProducedQty * c.qty * (1 + c.loss/100.0)
		if consumed <= 0 {
			continue
		}
		mov := &stockentity.StockMovement{
			ItemCode:      c.code,
			WarehouseID:   *dto.BackflushWarehouseID,
			MovementType:  stockentity.MovementTypeOut,
			Quantity:      consumed,
			ReferenceType: &refType,
			ReferenceCode: &refCode,
			CreatedBy:     dto.CreatedBy,
		}
		_, _ = uc.StockRepo.CreateMovement(ctx, mov)
	}
	_ = ap
}

type structureChild struct {
	code int64
	qty  float64
	loss float64
}
