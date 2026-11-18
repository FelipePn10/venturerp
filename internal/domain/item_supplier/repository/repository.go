package repository

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
)

type ItemSupplierRepository interface {
	Upsert(context.Context, *entity.ItemPreferredSupplier) (*entity.ItemPreferredSupplier, error)
	ListByItem(context.Context, int64, int64) ([]*entity.ItemPreferredSupplier, error)
	ListBySupplier(context.Context, int64, int64) ([]*entity.ItemPreferredSupplier, error)
	SearchExternal(context.Context, int64, int64, string) ([]*entity.ItemPreferredSupplier, error)
	GetPreferred(context.Context, int64, int64) (*entity.ItemPreferredSupplier, error)
	Delete(context.Context, int64, int64) error
	ItemAllowsConversionFactor(context.Context, int64) (bool, error)
	CreateQualityReport(context.Context, *entity.QualityReport) (*entity.QualityReport, error)
	ListQualityReports(context.Context, int64, int64) ([]*entity.QualityReport, error)
	GetQualityReport(context.Context, int64, int64) (*entity.QualityReport, error)
	SetBlocked(context.Context, *entity.BlockEvent) (*entity.ItemPreferredSupplier, error)
	ListBlockHistory(context.Context, int64, int64) ([]*entity.BlockEvent, error)
	ClearLearned(context.Context, int64, int64) (*entity.ItemPreferredSupplier, error)
	// FonteDoItem: o fornecedor de quem comprar o item nesta máscara hoje.
	// Com fornecedor informado, confere que ele é fonte; sem, devolve o
	// preferencial/melhor ranking. ok=false quando não há fonte válida.
	FonteDoItem(ctx context.Context, enterpriseID, item int64, mask string, informado *int64) (supplier int64, ok bool, err error)
}
