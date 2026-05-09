package machine_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ScheduleMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ScheduleMachineUseCase) CreateSchedule(
	ctx context.Context,
	dto request.CreateMachineScheduleDTO) (*entity.MachineSchedule, error) {
	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	date, _ := time.Parse("2006-01-02", dto.ScheduleDate)
	var start, end *time.Time
	if dto.StartTime != nil {
		s, _ := time.Parse("15:04:05", *dto.StartTime)
		start = &s
	}
	if dto.EndTime != nil {
		e, _ := time.Parse("15:04:05", *dto.EndTime)
		end = &e
	}
	schedule := &entity.MachineSchedule{
		MachineCode:      dto.MachineCode,
		OrderCode:        dto.OrderCode,
		ScheduleDate:     date,
		StartTime:        start,
		EndTime:          end,
		PlannedQty:       dto.PlannedQty,
		Sequence:         dto.Sequence,
		PriorityOverride: dto.PriorityOverride,
		Notes:            dto.Notes,
	}
	return uc.Repo.CreateSchedule(ctx, schedule)
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
) (*entity.MachineSchedule, error) {

	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.UpdateScheduleStatus(
		ctx,
		code,
		dto.Status,
		dto.ProducedQty,
	)
}

func (uc *ScheduleMachineUseCase) UpdateTimes(
	ctx context.Context,
	code int64,
	dto request.UpdateScheduleTimesDTO,
) (*entity.MachineSchedule, error) {

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

	return uc.Repo.UpdateScheduleTimes(
		ctx,
		code,
		start,
		end,
	)
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

func (uc *ScheduleMachineUseCase) GetSchedule(ctx context.Context, code int64) (*entity.MachineSchedule, error) {
	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.GetSchedule(ctx, code)
}

func (uc *ScheduleMachineUseCase) ListSchedules(
	ctx context.Context,
	machineID int64,
	date time.Time,
) ([]*entity.MachineSchedule, error) {
	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.ListSchedules(ctx, machineID, date)
}

func (uc *ScheduleMachineUseCase) ListSchedulesByRange(ctx context.Context, machineID int64, start, end time.Time) ([]*entity.MachineSchedule, error) {
	if !uc.Auth.CanSchedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.ListSchedulesByRange(ctx, machineID, start, end)
}
