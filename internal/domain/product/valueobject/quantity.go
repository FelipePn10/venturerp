package valueobject

import (
	"errors"

	"github.com/shopspring/decimal"
)

type Quantity struct {
	value decimal.Decimal
}

func (q Quantity) Value() decimal.Decimal {
	return q.value
}

func (q Quantity) Add(other Quantity) Quantity {
	return Quantity{
		value: q.value.Add(other.value),
	}
}

func NewQuantity(value decimal.Decimal) (Quantity, error) {
	if value.LessThanOrEqual(decimal.Zero) {
		return Quantity{}, errors.New("a quantidade deve ser maior que zero")
	}

	return Quantity{value: value}, nil
}
