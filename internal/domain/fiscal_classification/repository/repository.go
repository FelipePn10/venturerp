package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity"
)

type FiscalClassificationRepository interface {
	Create(ctx context.Context, c *entity.FiscalClassification) (*entity.FiscalClassification, error)
	Update(ctx context.Context, c *entity.FiscalClassification) (*entity.FiscalClassification, error)
	GetByCode(ctx context.Context, code int64) (*entity.FiscalClassification, error)
	List(ctx context.Context, onlyActive bool) ([]*entity.FiscalClassification, error)
	NextCode(ctx context.Context) (int64, error)

	// Languages
	AddLanguage(ctx context.Context, l *entity.FiscalClassificationLanguage) (*entity.FiscalClassificationLanguage, error)
	ListLanguages(ctx context.Context, classificationID int64) ([]*entity.FiscalClassificationLanguage, error)
	DeleteLanguage(ctx context.Context, id int64) error

	// Export attributes
	AddExportAttribute(ctx context.Context, a *entity.FiscalClassificationExportAttribute) (*entity.FiscalClassificationExportAttribute, error)
	ListExportAttributes(ctx context.Context, classificationID int64) ([]*entity.FiscalClassificationExportAttribute, error)
	DeleteExportAttribute(ctx context.Context, id int64) error
}
