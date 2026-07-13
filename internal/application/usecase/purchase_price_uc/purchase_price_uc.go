package purchase_price_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository"
)

type PurchasePriceUseCase struct {
	repo repository.PurchasePriceRepository
	auth ports.AuthService
}

func NewPurchasePriceUseCase(repo repository.PurchasePriceRepository, auth ports.AuthService) *PurchasePriceUseCase {
	return &PurchasePriceUseCase{repo: repo, auth: auth}
}

func parseDatePtr(s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*s))
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: use YYYY-MM-DD", *s)
	}
	return &t, nil
}

func (uc *PurchasePriceUseCase) tenant(ctx context.Context) (int64, error) {
	return uc.auth.EnterpriseID(ctx)
}

func (uc *PurchasePriceUseCase) CreateTable(ctx context.Context, dto request.CreatePurchasePriceTableDTO) (*response.PurchasePriceTableResponse, error) {
	enterpriseID, err := uc.tenant(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := uc.auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	code, err := uc.repo.NextTableCode(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	t, err := entity.NewPurchasePriceTable(enterpriseID, code, dto.SupplierCode, dto.Description, dto.CurrencyCode, actor)
	if err != nil {
		return nil, err
	}
	if t.ValidityStart, err = parseDatePtr(dto.ValidityStart); err != nil {
		return nil, err
	}
	if t.ValidityEnd, err = parseDatePtr(dto.ValidityEnd); err != nil {
		return nil, err
	}
	if err = t.ValidateValidity(); err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateTable(ctx, t)
	if err != nil {
		return nil, err
	}
	return toPriceTableResponse(created), nil
}

func (uc *PurchasePriceUseCase) UpdateTable(ctx context.Context, dto request.UpdatePurchasePriceTableDTO) (*response.PurchasePriceTableResponse, error) {
	enterpriseID, err := uc.tenant(ctx)
	if err != nil {
		return nil, err
	}
	t, err := uc.repo.GetTableByCode(ctx, enterpriseID, dto.Code)
	if err != nil {
		return nil, err
	}
	if dto.SupplierCode <= 0 || strings.TrimSpace(dto.Description) == "" {
		return nil, fmt.Errorf("supplier_code and description are required")
	}
	t.SupplierCode, t.Description, t.IsActive = dto.SupplierCode, strings.TrimSpace(dto.Description), dto.IsActive
	if dto.CurrencyCode != "" {
		t.CurrencyCode = strings.ToUpper(strings.TrimSpace(dto.CurrencyCode))
	}
	if t.ValidityStart, err = parseDatePtr(dto.ValidityStart); err != nil {
		return nil, err
	}
	if t.ValidityEnd, err = parseDatePtr(dto.ValidityEnd); err != nil {
		return nil, err
	}
	if err = t.ValidateValidity(); err != nil {
		return nil, err
	}
	updated, err := uc.repo.UpdateTable(ctx, t)
	if err != nil {
		return nil, err
	}
	return toPriceTableResponse(updated), nil
}

func (uc *PurchasePriceUseCase) GetTable(ctx context.Context, code int64) (*response.PurchasePriceTableResponse, error) {
	e, err := uc.tenant(ctx)
	if err != nil {
		return nil, err
	}
	t, err := uc.repo.GetTableByCode(ctx, e, code)
	if err != nil {
		return nil, err
	}
	if t.Items, err = uc.repo.ListItems(ctx, e, t.ID); err != nil {
		return nil, err
	}
	return toPriceTableResponse(t), nil
}
func (uc *PurchasePriceUseCase) ListTables(ctx context.Context, supplier *int64, active bool) ([]*response.PurchasePriceTableResponse, error) {
	e, err := uc.tenant(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.ListTables(ctx, e, supplier, active)
	if err != nil {
		return nil, err
	}
	return toPriceTableResponses(x), nil
}

func (uc *PurchasePriceUseCase) AddItem(ctx context.Context, dto request.AddPurchasePriceItemDTO) (*response.PurchasePriceTableItemResponse, error) {
	e, err := uc.tenant(ctx)
	if err != nil {
		return nil, err
	}
	t, err := uc.repo.GetTableByCode(ctx, e, dto.TableCode)
	if err != nil {
		return nil, err
	}
	item, err := entity.NewPurchasePriceTableItem(t.ID, dto.ItemCode, dto.Price)
	if err != nil {
		return nil, err
	}
	item.SupplierCode, item.UOM, item.MinQty, item.UpdateReplacementValue = dto.SupplierCode, dto.UOM, dto.MinQty, dto.UpdateReplacementValue
	if item.MinQty.IsNegative() {
		return nil, fmt.Errorf("min_qty must not be negative")
	}
	if item.SupplierCode == nil {
		item.SupplierCode = &t.SupplierCode
	} else if *item.SupplierCode != t.SupplierCode {
		return nil, fmt.Errorf("supplier_code must match the purchase price table supplier")
	}
	if item.UpdateReplacementValue {
		ok, err := uc.repo.IsPreferredSupplier(ctx, e, item.ItemCode, *item.SupplierCode)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("update_replacement_value requires a preferred supplier for the item")
		}
	}
	for _, a := range dto.Adjustments {
		adj, err := entity.NewPriceAdjustment(a.Sequence, a.Kind, a.CalculationType, a.Value)
		if err != nil {
			return nil, err
		}
		item.Adjustments = append(item.Adjustments, adj)
	}
	created, err := uc.repo.AddItem(ctx, e, item)
	if err != nil {
		return nil, err
	}
	return toPriceTableItemResponse(created), nil
}
func (uc *PurchasePriceUseCase) ListItems(ctx context.Context, code int64) ([]*response.PurchasePriceTableItemResponse, error) {
	e, err := uc.tenant(ctx)
	if err != nil {
		return nil, err
	}
	t, err := uc.repo.GetTableByCode(ctx, e, code)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.ListItems(ctx, e, t.ID)
	if err != nil {
		return nil, err
	}
	return toPriceTableItemResponses(x), nil
}
func (uc *PurchasePriceUseCase) DeleteItem(ctx context.Context, id int64) error {
	e, err := uc.tenant(ctx)
	if err != nil {
		return err
	}
	return uc.repo.DeleteItem(ctx, e, id)
}

func (uc *PurchasePriceUseCase) ListCandidates(ctx context.Context, tableCode int64, mode, order string, classificationID *int64) ([]response.PurchasePriceItemCandidateResponse, error) {
	e, err := uc.tenant(ctx)
	if err != nil {
		return nil, err
	}
	mode, order = strings.ToUpper(mode), strings.ToUpper(order)
	if mode != "INTERNAL" && mode != "SUPPLIER" {
		return nil, fmt.Errorf("mode must be INTERNAL or SUPPLIER")
	}
	if order != "NUMERIC" && order != "ALPHANUMERIC" {
		return nil, fmt.Errorf("order must be NUMERIC or ALPHANUMERIC")
	}
	x, err := uc.repo.ListItemCandidates(ctx, e, tableCode, mode, order, classificationID)
	if err != nil {
		return nil, err
	}
	out := make([]response.PurchasePriceItemCandidateResponse, 0, len(x))
	for _, v := range x {
		out = append(out, response.PurchasePriceItemCandidateResponse(v))
	}
	return out, nil
}
func (uc *PurchasePriceUseCase) CopyAdjustments(ctx context.Context, dto request.CopyPriceAdjustmentsDTO) error {
	e, err := uc.tenant(ctx)
	if err != nil {
		return err
	}
	mode := strings.ToUpper(dto.Mode)
	if mode != "REPLACE" && mode != "ADD" {
		return fmt.Errorf("mode must be REPLACE or ADD")
	}
	return uc.repo.CopyAdjustments(ctx, e, dto.SourceItemID, dto.TargetItemID, mode)
}
func (uc *PurchasePriceUseCase) ListSourcePrices(ctx context.Context, f repository.SourceFilter) ([]response.PurchasePriceSourceResponse, error) {
	e, err := uc.tenant(ctx)
	if err != nil {
		return nil, err
	}
	f.EnterpriseID = e
	f.Source = strings.ToUpper(f.Source)
	x, err := uc.repo.ListSourcePrices(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]response.PurchasePriceSourceResponse, 0, len(x))
	for _, v := range x {
		out = append(out, response.PurchasePriceSourceResponse(v))
	}
	return out, nil
}
func (uc *PurchasePriceUseCase) ApplySourcePrices(ctx context.Context, dto request.ApplyPurchasePriceSourcesDTO) (int64, error) {
	e, err := uc.tenant(ctx)
	if err != nil {
		return 0, err
	}
	s := make([]repository.ApplySourceSelection, 0, len(dto.Selections))
	for _, x := range dto.Selections {
		s = append(s, repository.ApplySourceSelection{SourceType: strings.ToUpper(x.SourceType), SourceID: x.SourceID})
	}
	return uc.repo.ApplySourcePrices(ctx, e, dto.TableCode, dto.Overwrite, s)
}

// GetItemPrice implements ports.PurchasePriceProvider.
func (uc *PurchasePriceUseCase) GetItemPrice(ctx context.Context, tableCode, itemCode int64, supplierCode *int64) (float64, string, bool, error) {
	e, err := uc.tenant(ctx)
	if err != nil {
		return 0, "", false, err
	}
	item, err := uc.repo.GetItemPrice(ctx, e, tableCode, itemCode, supplierCode)
	if err != nil || item == nil {
		return 0, "", false, nil
	}
	u := ""
	if item.UOM != nil {
		u = *item.UOM
	}
	p, _ := item.Price.Float64()
	return p, u, true, nil
}
