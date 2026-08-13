package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
)

type ProductionScannerRepository interface {
	CreateScanToken(context.Context, *entity.ScanToken) error
	ExecuteScan(context.Context, entity.ScanCommand) (*entity.ScanResult, error)
	RevokeScanToken(context.Context, int64, []byte, time.Time) error
}
