package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ItemUnitConversion: 1 FromUOM = Factor × ToUOM, for a given item.
type ItemUnitConversion struct {
	ID        int64
	ItemCode  int64
	FromUOM   string
	ToUOM     string
	Factor    float64
	IsActive  bool
	CreatedAt time.Time
	CreatedBy uuid.UUID
}

func NewItemUnitConversion(itemCode int64, fromUOM, toUOM string, factor float64, createdBy uuid.UUID) (*ItemUnitConversion, error) {
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
	return &ItemUnitConversion{
		ItemCode:  itemCode,
		FromUOM:   fromUOM,
		ToUOM:     toUOM,
		Factor:    factor,
		IsActive:  true,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}, nil
}
