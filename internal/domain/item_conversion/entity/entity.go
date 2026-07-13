package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ItemUnitConversion: 1 FromUOM = Factor × ToUOM, for a given item.
type ItemUnitConversion struct {
	ID              int64
	ItemCode        int64
	Mask            string
	FromUOM         string
	ToUOM           string
	Factor          float64
	RoundingPercent float64
	ToleranceValue  float64
	ToleranceType   string
	IsActive        bool
	CreatedAt       time.Time
	CreatedBy       uuid.UUID
}

func NewItemUnitConversion(itemCode int64, mask, fromUOM, toUOM string, factor, roundingPercent, toleranceValue float64, toleranceType string, createdBy uuid.UUID) (*ItemUnitConversion, error) {
	fromUOM = strings.ToUpper(strings.TrimSpace(fromUOM))
	toUOM = strings.ToUpper(strings.TrimSpace(toUOM))
	if itemCode == 0 {
		return nil, fmt.Errorf("item_code is required")
	}
	if fromUOM == "" || toUOM == "" {
		return nil, fmt.Errorf("from_uom and to_uom are required")
	}
	if fromUOM == toUOM {
		return nil, fmt.Errorf("from_uom and to_uom must differ")
	}
	if factor <= 0 {
		return nil, fmt.Errorf("factor must be greater than zero")
	}
	mask = strings.TrimSpace(mask)
	toleranceType = strings.ToUpper(strings.TrimSpace(toleranceType))
	if toleranceType == "" {
		toleranceType = "VALUE"
	}
	if roundingPercent < 0 || roundingPercent > 100 || toleranceValue < 0 || (toleranceType != "VALUE" && toleranceType != "PERCENT") {
		return nil, fmt.Errorf("invalid rounding/tolerance policy")
	}
	return &ItemUnitConversion{
		ItemCode:        itemCode,
		Mask:            mask,
		FromUOM:         fromUOM,
		ToUOM:           toUOM,
		Factor:          factor,
		RoundingPercent: roundingPercent, ToleranceValue: toleranceValue, ToleranceType: toleranceType,
		IsActive:  true,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}, nil
}
