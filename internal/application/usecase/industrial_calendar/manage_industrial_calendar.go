package industrial_calendar_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository"
)

type ManageCalendarUseCase struct {
	Repo repository.IndustrialCalendarRepository
	Auth ports.AuthService
}

func (uc *ManageCalendarUseCase) CreateDay(
	ctx context.Context,
	dto request.CreateCalendarDayDTO,
) (*entity.IndustrialCalendar, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	cal := &entity.IndustrialCalendar{
		Year:        dto.Year,
		Month:       dto.Month,
		Day:         dto.Day,
		IsWorkday:   dto.IsWorkday,
		Description: dto.Description,
	}
	return uc.Repo.CreateDay(ctx, cal)
}

func (uc *ManageCalendarUseCase) GetMonth(ctx context.Context, year, month int) ([]*entity.IndustrialCalendar, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListMonth(ctx, year, month)
}

func (uc *ManageCalendarUseCase) IsWorkday(ctx context.Context, year, month, day int) (bool, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return false, errorsuc.ErrUnauthorized
	}
	return uc.Repo.IsWorkday(ctx, year, month, day)
}
func (uc *ManageCalendarUseCase) GetNextWorkday(ctx context.Context, year, month, day int) (time.Time, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return time.Time{}, errorsuc.ErrUnauthorized
	}

	return uc.Repo.GetNextWorkday(ctx, year, month, day)
}

func (uc *ManageCalendarUseCase) GetWorkdaysInMonth(ctx context.Context, year, month int) ([]*entity.IndustrialCalendar, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetWorkdaysInMonth(ctx, year, month)
}

func (uc *ManageCalendarUseCase) GetDay(
	ctx context.Context,
	year, month, day int,
) (*entity.IndustrialCalendar, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.GetDay(ctx, year, month, day)
}

func (uc *ManageCalendarUseCase) DeleteDay(
	ctx context.Context,
	year, month, day int,
) error {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return errorsuc.ErrUnauthorized
	}

	return uc.Repo.DeleteDay(ctx, year, month, day)
}
