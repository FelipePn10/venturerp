package entity

import (
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

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
		return nil, errorsuc.NewValidationError("informe o item da conversão")
	}
	if fromUOM == "" || toUOM == "" {
		return nil, errorsuc.NewValidationError("informe a unidade de origem e a de destino")
	}
	if fromUOM == toUOM {
		return nil, errorsuc.NewValidationError("a unidade de origem e a de destino devem ser diferentes")
	}
	if factor <= 0 {
		return nil, errorsuc.NewValidationError("o fator de conversão deve ser maior que zero")
	}
	mask = strings.TrimSpace(mask)
	toleranceType = strings.ToUpper(strings.TrimSpace(toleranceType))
	if toleranceType == "" {
		toleranceType = "VALUE"
	}
	if roundingPercent < 0 || roundingPercent > 100 || toleranceValue < 0 || (toleranceType != "VALUE" && toleranceType != "PERCENT") {
		return nil, errorsuc.NewValidationError("política de arredondamento/tolerância inválida: o arredondamento vai de 0 a 100%, a tolerância não pode ser negativa e o tipo deve ser VALUE (valor) ou PERCENT (percentual)")
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
