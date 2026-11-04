package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type NcmTaxTable struct {
	ID          int64
	Ncm         string
	AliqIPI     decimal.Decimal
	AliqPis     decimal.Decimal
	AliqCofins  decimal.Decimal
	CstPis      string
	CstCofins   string
	CstIPI      string
	Description *string
	IsActive    bool
	CreatedAt   time.Time
}
