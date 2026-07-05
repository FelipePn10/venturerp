package sales_forecast_uc

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/forecast_uc"
	calendarrepo "github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
	"github.com/google/uuid"
)

type GenerateSalesForecastUseCase struct {
	Repo     repository.SalesForecastRepository
	Calendar calendarrepo.IndustrialCalendarRepository
	Auth     ports.AuthService
}

type CreateMonthlySalesForecastUseCase struct {
	Repo     repository.SalesForecastRepository
	Calendar calendarrepo.IndustrialCalendarRepository
	Auth     ports.AuthService
}

func (uc *CreateMonthlySalesForecastUseCase) Execute(
	ctx context.Context,
	dto request.CreateMonthlySalesForecastDTO,
) (*response.GenerateSalesForecastResponse, error) {
	if !uc.Auth.CanCreateSalesForecast(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Month < 1 || dto.Month > 12 {
		return nil, fmt.Errorf("month must be between 1 and 12")
	}
	weekly, err := distributeMonthByWorkdays(ctx, uc.Calendar, dto.Year, dto.Month, dto.Quantity, dto.AcceptsFraction, "LAST", nil, nil)
	if err != nil {
		return nil, err
	}
	return upsertWeeklyForecasts(ctx, uc.Repo, userID, dto.ItemCode, dto.Mask, weekly, dto.UpdateExisting, "MONTHLY_MANUAL", 0)
}

func (uc *GenerateSalesForecastUseCase) Execute(
	ctx context.Context,
	dto request.GenerateSalesForecastDTO,
) (*response.GenerateSalesForecastResponse, error) {
	if strings.TrimSpace(dto.HistorySource) != "" {
		return uc.executeFromERPHistory(ctx, dto)
	}
	return uc.executeStatistical(ctx, dto)
}

func (uc *GenerateSalesForecastUseCase) executeFromERPHistory(
	ctx context.Context,
	dto request.GenerateSalesForecastDTO,
) (*response.GenerateSalesForecastResponse, error) {
	if !uc.Auth.CanCreateSalesForecast(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}

	from, err := time.Parse("2006-01-02", dto.HistoryFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid history_from: %w", err)
	}
	to, err := time.Parse("2006-01-02", dto.HistoryTo)
	if err != nil {
		return nil, fmt.Errorf("invalid history_to: %w", err)
	}
	if to.Before(from) {
		return nil, fmt.Errorf("history_to must be greater than or equal to history_from")
	}

	startDate, err := weekToDate(dto.StartYear, dto.StartWeek)
	if err != nil {
		return nil, fmt.Errorf("invalid start week/year combination: %w", err)
	}
	endDate, err := weekToDate(dto.TargetEndYear, dto.TargetEndWeek)
	if err != nil {
		return nil, fmt.Errorf("invalid target end week/year combination: %w", err)
	}
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("target end week must be after start week")
	}

	history, err := uc.Repo.ListHistoricalDemand(ctx, dto.HistorySource, from, to, selectedItemCodes(dto))
	if err != nil {
		return nil, err
	}
	averages := averageHistoryByItemMask(history, calendarMonthCount(from, to), dto.ProjectionPct)
	if len(averages) == 0 {
		return nil, fmt.Errorf("no historical demand found for selected filters")
	}

	var merged *response.GenerateSalesForecastResponse
	months := monthsTouchedByRange(startDate, endDate)
	for _, avg := range averages {
		for _, month := range months {
			monthStart := time.Date(month.Year, time.Month(month.Month), 1, 0, 0, 0, 0, time.UTC)
			monthEnd := monthStart.AddDate(0, 1, -1)
			fromLimit, toLimit := clipRange(monthStart, monthEnd, startDate, endDate.AddDate(0, 0, 6))
			weekly, err := distributeMonthByWorkdays(ctx, uc.Calendar, month.Year, month.Month, avg.Quantity, dto.AcceptsFraction, "FIRST", &fromLimit, &toLimit)
			if err != nil {
				return nil, err
			}
			result, err := upsertWeeklyForecasts(ctx, uc.Repo, userID, avg.ItemCode, avg.Mask, weekly, dto.UpdateExisting, "ERP_HISTORY_AVERAGE", 0)
			if err != nil {
				return nil, err
			}
			merged = mergeGenerateResult(merged, result)
		}
	}
	if merged == nil {
		return nil, fmt.Errorf("no forecast periods generated")
	}
	merged.Model = "ERP_HISTORY_AVERAGE"
	return merged, nil
}

func (uc *GenerateSalesForecastUseCase) executeStatistical(
	ctx context.Context,
	dto request.GenerateSalesForecastDTO,
) (*response.GenerateSalesForecastResponse, error) {
	if !uc.Auth.CanCreateSalesForecast(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}

	startDate, err := weekToDate(dto.StartYear, dto.StartWeek)
	if err != nil {
		return nil, fmt.Errorf("invalid start week/year combination: %w", err)
	}

	statistical, err := forecast_uc.Execute(forecast_uc.StatisticalForecastDTO{
		ItemCode:  dto.ItemCode,
		History:   toForecastDataPoints(dto.History),
		Periods:   dto.Periods,
		Model:     normalizeForecastModel(dto.Model),
		MAWindow:  dto.MAWindow,
		Alpha:     dto.Alpha,
		Beta:      dto.Beta,
		Gamma:     dto.Gamma,
		SeasonLen: dto.SeasonLen,
	})
	if err != nil {
		return nil, err
	}

	weekly := map[forecastWeek]float64{}
	for i, quantity := range statistical.Result.Forecasts {
		periodDate := startDate.AddDate(0, 0, i*7)
		year, week := periodDate.ISOWeek()
		weekly[forecastWeek{Year: year, Week: week}] += quantity
	}
	return upsertWeeklyForecasts(ctx, uc.Repo, userID, dto.ItemCode, dto.Mask, weekly, dto.UpdateExisting, statistical.Result.Model, statistical.Result.MAPE)
}

func upsertWeeklyForecasts(
	ctx context.Context,
	repo repository.SalesForecastRepository,
	userID uuid.UUID,
	itemCode int64,
	mask *string,
	weekly map[forecastWeek]float64,
	updateExisting bool,
	model string,
	mape float64,
) (*response.GenerateSalesForecastResponse, error) {
	existing, err := repo.GetForecastByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	existingByPeriod := indexForecasts(existing, mask)

	result := &response.GenerateSalesForecastResponse{
		ItemCode: itemCode,
		Mask:     mask,
		Model:    model,
		MAPE:     mape,
		Created:  []*response.SalesForecastResponse{},
		Updated:  []*response.SalesForecastResponse{},
		Skipped:  []response.SkippedForecastPeriod{},
	}

	weeks := make([]forecastWeek, 0, len(weekly))
	for week := range weekly {
		weeks = append(weeks, week)
	}
	sort.Slice(weeks, func(i, j int) bool {
		if weeks[i].Year == weeks[j].Year {
			return weeks[i].Week < weeks[j].Week
		}
		return weeks[i].Year < weeks[j].Year
	})

	for _, period := range weeks {
		checkDate, _ := weekToDate(period.Year, period.Week)
		blocked, err := repo.IsBlocked(ctx, checkDate)
		if err != nil {
			return nil, fmt.Errorf("checking forecast period %d/%d: %w", period.Week, period.Year, err)
		}
		if blocked {
			result.Skipped = append(result.Skipped, response.SkippedForecastPeriod{
				Year:   period.Year,
				Week:   period.Week,
				Reason: "forecast period is blocked",
			})
			continue
		}

		key := forecastPeriodKey(period.Year, period.Week)
		if current, ok := existingByPeriod[key]; ok {
			if !updateExisting {
				result.Skipped = append(result.Skipped, response.SkippedForecastPeriod{
					Year:   period.Year,
					Week:   period.Week,
					Reason: "forecast already exists",
				})
				continue
			}
			current.Quantity = weekly[period]
			updated, err := repo.UpdateForecast(ctx, current)
			if err != nil {
				return nil, err
			}
			result.Updated = append(result.Updated, toSalesForecastResponse(updated))
			continue
		}

		forecast, err := entity.NewSalesForecast(itemCode, mask, period.Week, period.Year, weekly[period], userID)
		if err != nil {
			return nil, err
		}
		created, err := repo.CreateForecast(ctx, forecast)
		if err != nil {
			return nil, err
		}
		result.Created = append(result.Created, toSalesForecastResponse(created))
	}
	result.GeneratedCount = len(result.Created) + len(result.Updated)
	return result, nil
}

type forecastWeek struct {
	Year int
	Week int
}

type forecastMonth struct {
	Year  int
	Month int
}

type averageDemand struct {
	ItemCode int64
	Mask     *string
	Quantity float64
}

func distributeMonthByWorkdays(
	ctx context.Context,
	calendar calendarrepo.IndustrialCalendarRepository,
	year int,
	month int,
	quantity float64,
	acceptsFraction bool,
	residualWeek string,
	fromLimit *time.Time,
	toLimit *time.Time,
) (map[forecastWeek]float64, error) {
	if quantity <= 0 {
		return nil, entity.ErrInvalidQuantity
	}
	workdays, err := workdaysForMonth(ctx, calendar, year, month)
	if err != nil {
		return nil, err
	}
	filtered := make([]time.Time, 0, len(workdays))
	for _, day := range workdays {
		if fromLimit != nil && day.Before(*fromLimit) {
			continue
		}
		if toLimit != nil && day.After(*toLimit) {
			continue
		}
		filtered = append(filtered, day)
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no industrial workdays found for %04d-%02d in target range", year, month)
	}

	workdaysByWeek := map[forecastWeek]int{}
	for _, day := range filtered {
		isoYear, isoWeek := day.ISOWeek()
		workdaysByWeek[forecastWeek{Year: isoYear, Week: isoWeek}]++
	}
	weeks := make([]forecastWeek, 0, len(workdaysByWeek))
	for week := range workdaysByWeek {
		weeks = append(weeks, week)
	}
	sort.Slice(weeks, func(i, j int) bool {
		if weeks[i].Year == weeks[j].Year {
			return weeks[i].Week < weeks[j].Week
		}
		return weeks[i].Year < weeks[j].Year
	})

	totalWorkdays := len(filtered)
	weekly := map[forecastWeek]float64{}
	allocated := 0.0
	for _, week := range weeks {
		raw := quantity * float64(workdaysByWeek[week]) / float64(totalWorkdays)
		if acceptsFraction {
			weekly[week] = round4(raw)
		} else {
			weekly[week] = math.Floor(raw)
		}
		allocated += weekly[week]
	}
	residual := round4(quantity - allocated)
	if residual != 0 {
		target := weeks[len(weeks)-1]
		if strings.EqualFold(residualWeek, "FIRST") {
			target = weeks[0]
		}
		weekly[target] = round4(weekly[target] + residual)
	}
	return weekly, nil
}

func workdaysForMonth(ctx context.Context, calendar calendarrepo.IndustrialCalendarRepository, year, month int) ([]time.Time, error) {
	if calendar != nil {
		days, err := calendar.GetWorkdaysInMonth(ctx, year, month)
		if err != nil {
			return nil, err
		}
		if len(days) > 0 {
			out := make([]time.Time, 0, len(days))
			for _, day := range days {
				out = append(out, time.Date(day.Year, time.Month(day.Month), day.Day, 0, 0, 0, 0, time.UTC))
			}
			return out, nil
		}
	}

	var out []time.Time
	for day := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC); day.Month() == time.Month(month); day = day.AddDate(0, 0, 1) {
		if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
			out = append(out, day)
		}
	}
	return out, nil
}

func averageHistoryByItemMask(history []*entity.HistoricalDemand, divisor int, projectionPct float64) []averageDemand {
	if divisor <= 0 {
		divisor = 1
	}
	type aggregate struct {
		itemCode int64
		mask     *string
		total    float64
	}
	aggregates := map[string]*aggregate{}
	for _, row := range history {
		key := fmt.Sprintf("%d|%s", row.ItemCode, maskKey(row.Mask))
		if aggregates[key] == nil {
			aggregates[key] = &aggregate{itemCode: row.ItemCode, mask: row.Mask}
		}
		aggregates[key].total += row.Quantity
	}
	out := make([]averageDemand, 0, len(aggregates))
	factor := 1 + projectionPct/100
	for _, aggregate := range aggregates {
		out = append(out, averageDemand{
			ItemCode: aggregate.itemCode,
			Mask:     aggregate.mask,
			Quantity: round4((aggregate.total / float64(divisor)) * factor),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ItemCode == out[j].ItemCode {
			return maskKey(out[i].Mask) < maskKey(out[j].Mask)
		}
		return out[i].ItemCode < out[j].ItemCode
	})
	return out
}

func selectedItemCodes(dto request.GenerateSalesForecastDTO) []int64 {
	if len(dto.ItemCodes) > 0 {
		return dto.ItemCodes
	}
	if dto.ItemCode > 0 {
		return []int64{dto.ItemCode}
	}
	return []int64{}
}

func calendarMonthCount(from, to time.Time) int {
	from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	to = time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
	return (to.Year()-from.Year())*12 + int(to.Month()-from.Month()) + 1
}

func monthsTouchedByRange(from, to time.Time) []forecastMonth {
	cursor := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
	var out []forecastMonth
	for !cursor.After(end) {
		out = append(out, forecastMonth{Year: cursor.Year(), Month: int(cursor.Month())})
		cursor = cursor.AddDate(0, 1, 0)
	}
	return out
}

func clipRange(monthStart, monthEnd, from, to time.Time) (time.Time, time.Time) {
	start := monthStart
	if from.After(start) {
		start = from
	}
	end := monthEnd
	if to.Before(end) {
		end = to
	}
	return start, end
}

func mergeGenerateResult(base *response.GenerateSalesForecastResponse, next *response.GenerateSalesForecastResponse) *response.GenerateSalesForecastResponse {
	if base == nil {
		return next
	}
	base.Created = append(base.Created, next.Created...)
	base.Updated = append(base.Updated, next.Updated...)
	base.Skipped = append(base.Skipped, next.Skipped...)
	base.GeneratedCount += next.GeneratedCount
	return base
}

func maskKey(mask *string) string {
	if mask == nil {
		return ""
	}
	return *mask
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func toForecastDataPoints(history []request.DataPoint) []forecast_uc.DataPoint {
	points := make([]forecast_uc.DataPoint, 0, len(history))
	for _, point := range history {
		points = append(points, forecast_uc.DataPoint{
			Period:   point.Period,
			Quantity: point.Quantity,
		})
	}
	return points
}

func normalizeForecastModel(model string) string {
	model = strings.TrimSpace(strings.ToUpper(model))
	if model == "" {
		return forecast_uc.ModelAuto
	}
	return model
}

func indexForecasts(forecasts []*entity.SalesForecast, mask *string) map[string]*entity.SalesForecast {
	index := make(map[string]*entity.SalesForecast, len(forecasts))
	for _, forecast := range forecasts {
		if sameMask(forecast.Mask, mask) {
			index[forecastPeriodKey(forecast.Year, forecast.Week)] = forecast
		}
	}
	return index
}

func forecastPeriodKey(year, week int) string {
	return fmt.Sprintf("%04d-%02d", year, week)
}

func sameMask(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
