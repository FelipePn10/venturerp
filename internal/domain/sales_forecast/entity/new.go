package entity

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

var (
	ErrInvalidWeek          = domainerr.NewValidation("a semana deve estar entre 1 e 53")
	ErrInvalidYear          = domainerr.NewValidation("o ano deve ser maior que 2000")
	ErrInvalidQuantity      = domainerr.NewValidation("a quantidade deve ser maior que zero")
	ErrInvalidItemCode      = domainerr.NewValidation("o código do item deve ser maior que zero")
	ErrInvalidBlockDates    = domainerr.NewValidation("a data inicial deve ser anterior à final")
	ErrInvalidDescription   = domainerr.NewValidation("informe a descrição")
	ErrPercentageSumTooHigh = domainerr.NewValidation("a soma dos percentuais dos dias não pode passar de 100")
)

func NewSalesForecast(
	itemCode int64,
	mask *string,
	week int,
	year int,
	quantity decimal.Decimal,
	createdBy uuid.UUID,
) (*SalesForecast, error) {
	if itemCode <= 0 {
		return nil, ErrInvalidItemCode
	}
	if week < 1 || week > 53 {
		return nil, ErrInvalidWeek
	}
	if year <= 2000 {
		return nil, ErrInvalidYear
	}
	if !quantity.IsPositive() {
		return nil, ErrInvalidQuantity
	}

	now := time.Now()
	return &SalesForecast{
		ItemCode:  itemCode,
		Mask:      mask,
		Week:      week,
		Year:      year,
		Quantity:  quantity,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func NewSalesForecastBlock(
	startDate time.Time,
	endDate time.Time,
	reason *string,
	createdBy uuid.UUID,
) (*SalesForecastBlock, error) {
	if !startDate.Before(endDate) {
		return nil, ErrInvalidBlockDates
	}

	return &SalesForecastBlock{
		StartDate: startDate,
		EndDate:   endDate,
		Reason:    reason,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}, nil
}

func NewAppropriationTable(
	description string,
	mondayPct decimal.Decimal,
	tuesdayPct decimal.Decimal,
	wednesdayPct decimal.Decimal,
	thursdayPct decimal.Decimal,
	fridayPct decimal.Decimal,
	saturdayPct decimal.Decimal,
	sundayPct decimal.Decimal,
	isDefault bool,
	createdBy uuid.UUID,
) (*AppropriationTable, error) {
	if description == "" {
		return nil, ErrInvalidDescription
	}

	total := mondayPct.Add(tuesdayPct).Add(wednesdayPct).Add(thursdayPct).Add(fridayPct).Add(saturdayPct).Add(sundayPct)
	if total.GreaterThan(decimal.NewFromInt(100)) {
		return nil, ErrPercentageSumTooHigh
	}

	now := time.Now()
	return &AppropriationTable{
		Description:  description,
		MondayPct:    mondayPct,
		TuesdayPct:   tuesdayPct,
		WednesdayPct: wednesdayPct,
		ThursdayPct:  thursdayPct,
		FridayPct:    fridayPct,
		SaturdayPct:  saturdayPct,
		SundayPct:    sundayPct,
		IsDefault:    isDefault,
		CreatedAt:    now,
		UpdatedAt:    now,
		CreatedBy:    createdBy,
	}, nil
}
