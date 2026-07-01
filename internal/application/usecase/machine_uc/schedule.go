package machine_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type ScheduleMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ScheduleMachineUseCase) CreateSchedule(
	ctx context.Context,
	dto request.CreateMachineScheduleDTO) (*response.MachineScheduleResponse, error) {
	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.MachineCode == 0 {
		return nil, errorsuc.NewValidationError("machine_code is required")
	}

	// Persist the real scheduled date; fall back to today rather than the zero
	// time (0001-01-01) when the client omits or malforms the date.
	date := datetime.ParseDateOrDefault(dto.ScheduleDate, time.Now())
	var start, end *time.Time
	if dto.StartTime != nil {
		s, _ := time.Parse("15:04:05", *dto.StartTime)
		start = &s
	}
	if dto.EndTime != nil {
		e, _ := time.Parse("15:04:05", *dto.EndTime)
		end = &e
	}
	var orderCode *int64
	if dto.OrderCode != 0 {
		oc := dto.OrderCode
		orderCode = &oc
	}
	schedule := &entity.MachineSchedule{
		MachineCode:      dto.MachineCode,
		OrderCode:        orderCode,
		ScheduleDate:     date,
		StartTime:        start,
		EndTime:          end,
		PlannedQty:       dto.PlannedQty,
		Sequence:         dto.Sequence,
		PriorityOverride: dto.PriorityOverride,
		Notes:            dto.Notes,
	}
	created, err := uc.Repo.CreateSchedule(ctx, schedule)
	if err != nil {
		return nil, err
	}
	return toMachineScheduleResponse(created), nil
}

func (uc *ScheduleMachineUseCase) ReorderSchedule(
	ctx context.Context,
	dto request.ReorderScheduleDTO,
) error {
	_, err := uc.Repo.UpdateScheduleSequence(
		ctx,
		dto.ScheduleCode,
		dto.NewSequence,
		dto.PriorityOverride,
	)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ScheduleMachineUseCase) UpdateStatus(
	ctx context.Context,
	code int64,
	dto request.UpdateScheduleStatusDTO,
) (*response.MachineScheduleResponse, error) {

	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	updated, err := uc.Repo.UpdateScheduleStatus(
		ctx,
		code,
		dto.Status,
		dto.ProducedQty,
	)
	if err != nil {
		return nil, err
	}
	return toMachineScheduleResponse(updated), nil
}

func (uc *ScheduleMachineUseCase) UpdateTimes(
	ctx context.Context,
	code int64,
	dto request.UpdateScheduleTimesDTO,
) (*response.MachineScheduleResponse, error) {

	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	var start *time.Time
	var end *time.Time

	if dto.StartTime != nil {
		t, err := time.Parse("15:04:05", *dto.StartTime)
		if err != nil {
			return nil, err
		}

		start = &t
	}

	if dto.EndTime != nil {
		t, err := time.Parse("15:04:05", *dto.EndTime)
		if err != nil {
			return nil, err
		}

		end = &t
	}

	updated, err := uc.Repo.UpdateScheduleTimes(
		ctx,
		code,
		start,
		end,
	)
	if err != nil {
		return nil, err
	}
	return toMachineScheduleResponse(updated), nil
}

func (uc *ScheduleMachineUseCase) DeleteSchedule(
	ctx context.Context,
	code int64,
) error {

	if !uc.Auth.CanSchedule(ctx) {
		return errorsuc.ErrUnauthorized
	}

	return uc.Repo.DeleteSchedule(ctx, code)
}

func (uc *ScheduleMachineUseCase) GetSchedule(ctx context.Context, code int64) (*response.MachineScheduleResponse, error) {
	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	s, err := uc.Repo.GetSchedule(ctx, code)
	if err != nil {
		return nil, err
	}
	return toMachineScheduleResponse(s), nil
}

func (uc *ScheduleMachineUseCase) ListSchedules(
	ctx context.Context,
	machineID int64,
	date time.Time,
) ([]*response.MachineScheduleResponse, error) {
	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	list, err := uc.Repo.ListSchedules(ctx, machineID, date)
	if err != nil {
		return nil, err
	}
	return toMachineScheduleResponses(list), nil
}

func (uc *ScheduleMachineUseCase) ListSchedulesByRange(ctx context.Context, machineID int64, start, end time.Time) ([]*response.MachineScheduleResponse, error) {
	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	list, err := uc.Repo.ListSchedulesByRange(ctx, machineID, start, end)
	if err != nil {
		return nil, err
	}
	return toMachineScheduleResponses(list), nil
}
