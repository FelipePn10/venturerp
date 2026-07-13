package repository

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
)

type ItemSupplierRepository interface {
	Upsert(context.Context, *entity.ItemPreferredSupplier) (*entity.ItemPreferredSupplier, error)
	ListByItem(context.Context, int64, int64) ([]*entity.ItemPreferredSupplier, error)
	ListBySupplier(context.Context, int64, int64) ([]*entity.ItemPreferredSupplier, error)
	GetPreferred(context.Context, int64, int64) (*entity.ItemPreferredSupplier, error)
	Delete(context.Context, int64, int64) error
	ItemAllowsConversionFactor(context.Context, int64) (bool, error)
	CreateQualityReport(context.Context, *entity.QualityReport) (*entity.QualityReport, error)
	ListQualityReports(context.Context, int64, int64) ([]*entity.QualityReport, error)
}
