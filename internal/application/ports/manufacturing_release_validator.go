package ports

import "context"

type ManufacturingReleaseValidator interface {
	ValidateProductionRelease(ctx context.Context, itemCode int64) error
}
