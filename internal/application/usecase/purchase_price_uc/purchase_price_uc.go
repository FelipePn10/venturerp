package purchase_price_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository"
)

type PurchasePriceUseCase struct {
	repo repository.PurchasePriceRepository
}

func NewPurchasePriceUseCase(repo repository.PurchasePriceRepository) *PurchasePriceUseCase {
	return &PurchasePriceUseCase{repo: repo}
}

func parseDatePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}

func (uc *PurchasePriceUseCase) CreateTable(ctx context.Context, dto request.CreatePurchasePriceTableDTO) (*response.PurchasePriceTableResponse, error) {
	code, err := uc.repo.NextTableCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	t, err := entity.NewPurchasePriceTable(code, dto.Description, dto.CurrencyCode, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	t.ValidityStart = parseDatePtr(dto.ValidityStart)
	t.ValidityEnd = parseDatePtr(dto.ValidityEnd)
	created, err := uc.repo.CreateTable(ctx, t)
	if err != nil {
		return nil, err
	}
	return toPriceTableResponse(created), nil
}

func (uc *PurchasePriceUseCase) UpdateTable(ctx context.Context, dto request.UpdatePurchasePriceTableDTO) (*response.PurchasePriceTableResponse, error) {
	t, err := uc.repo.GetTableByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	t.Description = dto.Description
	if dto.CurrencyCode != "" {
		t.CurrencyCode = dto.CurrencyCode
	}
	t.ValidityStart = parseDatePtr(dto.ValidityStart)
	t.ValidityEnd = parseDatePtr(dto.ValidityEnd)
	t.IsActive = dto.IsActive
	updated, err := uc.repo.UpdateTable(ctx, t)
	if err != nil {
		return nil, err
	}
	return toPriceTableResponse(updated), nil
}

func (uc *PurchasePriceUseCase) GetTable(ctx context.Context, code int64) (*response.PurchasePriceTableResponse, error) {
	t, err := uc.repo.GetTableByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if t.Items, err = uc.repo.ListItems(ctx, t.ID); err != nil {
		return nil, err
	}
	return toPriceTableResponse(t), nil
}

func (uc *PurchasePriceUseCase) ListTables(ctx context.Context, onlyActive bool) ([]*response.PurchasePriceTableResponse, error) {
	tables, err := uc.repo.ListTables(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toPriceTableResponses(tables), nil
}

func (uc *PurchasePriceUseCase) AddItem(ctx context.Context, dto request.AddPurchasePriceItemDTO) (*response.PurchasePriceTableItemResponse, error) {
	t, err := uc.repo.GetTableByCode(ctx, dto.TableCode)
	if err != nil {
		return nil, err
	}
	item, err := entity.NewPurchasePriceTableItem(t.ID, dto.ItemCode, dto.Price)
	if err != nil {
		return nil, err
	}
	item.SupplierCode = dto.SupplierCode
	item.UOM = dto.UOM
	item.MinQty = dto.MinQty
	created, err := uc.repo.AddItem(ctx, item)
	if err != nil {
		return nil, err
	}
	return toPriceTableItemResponse(created), nil
}

func (uc *PurchasePriceUseCase) ListItems(ctx context.Context, tableCode int64) ([]*response.PurchasePriceTableItemResponse, error) {
	t, err := uc.repo.GetTableByCode(ctx, tableCode)
	if err != nil {
		return nil, err
	}
	items, err := uc.repo.ListItems(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	return toPriceTableItemResponses(items), nil
}

func (uc *PurchasePriceUseCase) DeleteItem(ctx context.Context, id int64) error {
	return uc.repo.DeleteItem(ctx, id)
}

// GetItemPrice implements ports.PurchasePriceProvider.
func (uc *PurchasePriceUseCase) GetItemPrice(ctx context.Context, tableCode, itemCode int64, supplierCode *int64) (float64, string, bool, error) {
	item, err := uc.repo.GetItemPrice(ctx, tableCode, itemCode, supplierCode)
	if err != nil || item == nil {
		return 0, "", false, nil
	}
	uom := ""
	if item.UOM != nil {
		uom = *item.UOM
	}
	return item.Price, uom, true, nil
}
