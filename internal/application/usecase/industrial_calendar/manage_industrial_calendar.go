package industrial_calendar_uc

import (
	"context"
	"errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository"
)

var ErrInvalidCalendarDate = errors.New("data do calendario invalida")
var ErrCalendarGenerationUnavailable = errors.New("geracao do calendario indisponivel")

type calendarMonthGenerator interface {
	GenerateMonth(context.Context, int, int) (int, error)
}
type calendarRangeGenerator interface {
	GenerateRange(context.Context, int, *int, []int) (int, int, error)
}

func validateMonth(year, month int) error {
	if year < 2000 || year > 2200 || month < 1 || month > 12 {
		return ErrInvalidCalendarDate
	}
	return nil
}

func validateDay(year, month, day int) error {
	if validateMonth(year, month) != nil || day < 1 || day > 31 {
		return ErrInvalidCalendarDate
	}
	d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if d.Year() != year || int(d.Month()) != month || d.Day() != day {
		return ErrInvalidCalendarDate
	}
	return nil
}

type ManageCalendarUseCase struct {
	Repo repository.IndustrialCalendarRepository
	Auth ports.AuthService
}

func (uc *ManageCalendarUseCase) CreateDay(
	ctx context.Context,
	dto request.CreateCalendarDayDTO,
) (*response.IndustrialCalendarResponse, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := validateDay(dto.Year, dto.Month, dto.Day); err != nil {
		return nil, err
	}
	cal := &entity.IndustrialCalendar{
		Year:        dto.Year,
		Month:       dto.Month,
		Day:         dto.Day,
		IsWorkday:   dto.IsWorkday,
		Description: dto.Description,
	}
	created, err := uc.Repo.CreateDay(ctx, cal)
	if err != nil {
		return nil, err
	}
	return toIndustrialCalendarResponse(created), nil
}

func (uc *ManageCalendarUseCase) GenerateMonth(ctx context.Context, year, month int) (*response.GenerateIndustrialCalendarResponse, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := validateMonth(year, month); err != nil {
		return nil, err
	}
	existing, err := uc.Repo.ListMonth(ctx, year, month)
	if err != nil {
		return nil, err
	}
	generator, ok := uc.Repo.(calendarMonthGenerator)
	if !ok {
		return nil, ErrCalendarGenerationUnavailable
	}
	created, err := generator.GenerateMonth(ctx, year, month)
	if err != nil {
		return nil, err
	}
	return &response.GenerateIndustrialCalendarResponse{Year: year, Month: &month, Created: created, Preserved: len(existing)}, nil
}

func (uc *ManageCalendarUseCase) Generate(ctx context.Context, dto request.GenerateIndustrialCalendarDTO) (*response.GenerateIndustrialCalendarResponse, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Year < 2000 || dto.Year > 2200 {
		return nil, ErrInvalidCalendarDate
	}
	if dto.Month != nil {
		if err := validateMonth(dto.Year, *dto.Month); err != nil {
			return nil, err
		}
	}
	if len(dto.Weekdays) == 0 {
		dto.Weekdays = []int{1, 2, 3, 4, 5}
	}
	seen := map[int]bool{}
	for _, d := range dto.Weekdays {
		if d < 0 || d > 6 || seen[d] {
			return nil, ErrInvalidCalendarDate
		}
		seen[d] = true
	}
	g, ok := uc.Repo.(calendarRangeGenerator)
	if !ok {
		return nil, ErrCalendarGenerationUnavailable
	}
	created, preserved, err := g.GenerateRange(ctx, dto.Year, dto.Month, dto.Weekdays)
	if err != nil {
		return nil, err
	}
	total := 365
	if time.Date(dto.Year+1, 1, 1, 0, 0, 0, 0, time.UTC).Sub(time.Date(dto.Year, 1, 1, 0, 0, 0, 0, time.UTC)).Hours()/24 == 366 {
		total = 366
	}
	if dto.Month != nil {
		total = time.Date(dto.Year, time.Month(*dto.Month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	}
	return &response.GenerateIndustrialCalendarResponse{Year: dto.Year, Month: dto.Month, Created: created, Preserved: preserved, Ignored: total - created - preserved}, nil
}

func (uc *ManageCalendarUseCase) GetMonth(ctx context.Context, year, month int) ([]*response.IndustrialCalendarResponse, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := validateMonth(year, month); err != nil {
		return nil, err
	}
	list, err := uc.Repo.ListMonth(ctx, year, month)
	if err != nil {
		return nil, err
	}
	return toIndustrialCalendarResponses(list), nil
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

func (uc *ManageCalendarUseCase) GetWorkdaysInMonth(ctx context.Context, year, month int) ([]*response.IndustrialCalendarResponse, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.GetWorkdaysInMonth(ctx, year, month)
	if err != nil {
		return nil, err
	}
	return toIndustrialCalendarResponses(list), nil
}

func (uc *ManageCalendarUseCase) GetDay(
	ctx context.Context,
	year, month, day int,
) (*response.IndustrialCalendarResponse, error) {
	if !uc.Auth.CanManageIndustrialCalendar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	c, err := uc.Repo.GetDay(ctx, year, month, day)
	if err != nil {
		return nil, err
	}
	return toIndustrialCalendarResponse(c), nil
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
