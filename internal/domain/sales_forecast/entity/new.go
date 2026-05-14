package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidWeek          = errors.New("week must be between 1 and 53")
	ErrInvalidYear          = errors.New("year must be greater than 2000")
	ErrInvalidQuantity      = errors.New("quantity must be greater than zero")
	ErrInvalidItemCode      = errors.New("item code must be greater than zero")
	ErrInvalidBlockDates    = errors.New("start date must be before end date")
	ErrInvalidDescription   = errors.New("description is required")
	ErrPercentageSumTooHigh = errors.New("sum of all day percentages must not exceed 100")
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
