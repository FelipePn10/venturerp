package ports

import (
	"context"

	"github.com/shopspring/decimal"
)

// FiscalClassificationProvider exposes fiscal-classification-derived defaults
// consumed by the Purchase Order item (IPI %, ICMS BC modalities). Implemented by
// fiscal_classification_uc.
type FiscalClassificationProvider interface {
	// GetIPIRate returns the IPI rate (%) for a classification code, and whether
	// the classification exists.
	GetIPIRate(ctx context.Context, classificationCode int64) (rate decimal.Decimal, found bool, err error)
}
