package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidWeek          = errors.New("a semana deve estar entre 1 e 53")
	ErrInvalidYear          = errors.New("o ano deve ser maior que 2000")
	ErrInvalidQuantity      = errors.New("a quantidade deve ser maior que zero")
	ErrInvalidItemCode      = errors.New("o código do item deve ser maior que zero")
	ErrInvalidBlockDates    = errors.New("a data inicial deve ser anterior à final")
	ErrInvalidDescription   = errors.New("informe a descrição")
	ErrPercentageSumTooHigh = errors.New("a soma dos percentuais dos dias não pode passar de 100")
)

func NewSalesForecast(
	itemCode int64,
	mask *string,
	week int,
	year int,
	quantity float64,
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
	if quantity <= 0 {
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
	mondayPct float64,
	tuesdayPct float64,
	wednesdayPct float64,
	thursdayPct float64,
	fridayPct float64,
	saturdayPct float64,
	sundayPct float64,
	isDefault bool,
	createdBy uuid.UUID,
) (*AppropriationTable, error) {
	if description == "" {
		return nil, ErrInvalidDescription
	}

	total := mondayPct + tuesdayPct + wednesdayPct + thursdayPct + fridayPct + saturdayPct + sundayPct
	if total > 100.0 {
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
