package item_supplier_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository"
)

type ItemSupplierUseCase struct {
	repo repository.ItemSupplierRepository
}

func NewItemSupplierUseCase(repo repository.ItemSupplierRepository) *ItemSupplierUseCase {
	return &ItemSupplierUseCase{repo: repo}
}

func (uc *ItemSupplierUseCase) Upsert(ctx context.Context, dto request.UpsertItemPreferredSupplierDTO) (*entity.ItemPreferredSupplier, error) {
	s, err := entity.NewItemPreferredSupplier(dto.ItemCode, dto.SupplierCode, dto.Ranking, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	s.SupplierItemCode = dto.SupplierItemCode
	s.SupplierDescription = dto.SupplierDescription
	s.UOM = dto.UOM
	s.LeadTimeDays = dto.LeadTimeDays
	return uc.repo.Upsert(ctx, s)
}

func (uc *ItemSupplierUseCase) ListByItem(ctx context.Context, itemCode int64) ([]*entity.ItemPreferredSupplier, error) {
	return uc.repo.ListByItem(ctx, itemCode)
}

func (uc *ItemSupplierUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// GetPreferredSupplier implements ports.PreferredSupplierProvider.
func (uc *ItemSupplierUseCase) GetPreferredSupplier(ctx context.Context, itemCode int64) (int64, bool, error) {
	s, err := uc.repo.GetPreferred(ctx, itemCode)
	if err != nil || s == nil {
		return 0, false, nil
	}
	return s.SupplierCode, true, nil
}
