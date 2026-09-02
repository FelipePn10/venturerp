package item_calendar_promise_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/repository"
)

func validateDate(itemCode int64, year, month, day int, requireDay bool) error {
	if itemCode <= 0 || year <= 0 || month < 1 || month > 12 {
		return errorsuc.NewValidationError("item, ano positivo e mês entre 1 e 12 são obrigatórios")
	}
	if requireDay {
		date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if day < 1 || date.Year() != year || int(date.Month()) != month || date.Day() != day {
			return errorsuc.NewValidationError("dia inválido para o ano e mês informados")
		}
	}
	return nil
}

type ManageItemCalendarPromiseUseCase struct {
	Repo repository.ItemCalendarPromiseRepository
	Auth ports.AuthService
}

func (uc *ManageItemCalendarPromiseUseCase) UpsertDay(
	ctx context.Context,
	dto request.CreateItemCalendarDayDTO,
) (*response.ItemCalendarPromiseResponse, error) {
	if !uc.Auth.CanManageItemCalendarPromise(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := validateDate(dto.ItemCode, dto.Year, dto.Month, dto.Day, true); err != nil {
		return nil, err
	}

	cal := &entity.ItemCalendarPromise{
		ItemCode:    dto.ItemCode,
		Mask:        dto.Mask,
		Year:        dto.Year,
		Month:       dto.Month,
		Day:         dto.Day,
		IsWorkday:   dto.IsWorkday,
		Description: dto.Description,
	}

	saved, err := uc.Repo.UpsertDay(ctx, cal)
	if err != nil {
		return nil, err
	}
	return toItemCalendarPromiseResponse(saved), nil
}

func (uc *ManageItemCalendarPromiseUseCase) GetDay(
	ctx context.Context,
	itemCode int64,
	mask string,
	year, month, day int,
) (*response.ItemCalendarPromiseResponse, error) {
	if !uc.Auth.CanManageItemCalendarPromise(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := validateDate(itemCode, year, month, day, true); err != nil {
		return nil, err
	}
	c, err := uc.Repo.GetDay(ctx, itemCode, mask, year, month, day)
	if err != nil {
		return nil, err
	}
	return toItemCalendarPromiseResponse(c), nil
}

func (uc *ManageItemCalendarPromiseUseCase) ListMonth(
	ctx context.Context,
	itemCode int64,
	mask string,
	year, month int,
) ([]*response.ItemCalendarPromiseResponse, error) {
	if !uc.Auth.CanManageItemCalendarPromise(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := validateDate(itemCode, year, month, 0, false); err != nil {
		return nil, err
	}
	list, err := uc.Repo.ListMonth(ctx, itemCode, mask, year, month)
	if err != nil {
		return nil, err
	}
	return toItemCalendarPromiseResponses(list), nil
}

func (uc *ManageItemCalendarPromiseUseCase) GetWorkdaysInMonth(
	ctx context.Context,
	itemCode int64,
	mask string,
	year, month int,
) ([]*response.ItemCalendarPromiseResponse, error) {
	if !uc.Auth.CanManageItemCalendarPromise(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := validateDate(itemCode, year, month, 0, false); err != nil {
		return nil, err
	}
	list, err := uc.Repo.GetWorkdaysInMonth(ctx, itemCode, mask, year, month)
	if err != nil {
		return nil, err
	}
	return toItemCalendarPromiseResponses(list), nil
}

func (uc *ManageItemCalendarPromiseUseCase) DeleteDay(
	ctx context.Context,
	itemCode int64,
	mask string,
	year, month, day int,
) error {

	if !uc.Auth.CanManageItemCalendarPromise(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if err := validateDate(itemCode, year, month, day, true); err != nil {
		return err
	}

	return uc.Repo.DeleteDay(ctx, itemCode, mask, year, month, day)
}
