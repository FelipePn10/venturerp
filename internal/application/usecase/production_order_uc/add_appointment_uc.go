package production_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type AddAppointmentUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
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

	return uc.Repo.AddAppointment(ctx, appointment)
}
