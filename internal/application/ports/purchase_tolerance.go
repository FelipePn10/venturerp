package ports

import (
	"context"

	"github.com/shopspring/decimal"
)

type PurchaseToleranceEvaluator interface {
	EvaluatePurchaseTolerance(ctx context.Context, supplier *int64, toleranceType, appliesTo string, expected, actual decimal.Decimal) (action, message string, exceeded bool, err error)
}
