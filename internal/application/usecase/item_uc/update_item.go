package item_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type UpdateItemUseCase struct {
	Repo repository.ItemRepository
	Auth ports.AuthService
}

func NewUpdateItemUseCase(repo repository.ItemRepository, auth ports.AuthService) *UpdateItemUseCase {
	return &UpdateItemUseCase{Repo: repo, Auth: auth}
}

func (uc *UpdateItemUseCase) Execute(ctx context.Context, code int64, dto request.UpdateItemDTO) (*response.ItemResponse, error) {
	if !uc.Auth.CanCreateItem(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	itemCode, err := valueobject.NewItemCode(code)
	if err != nil {
		return nil, err
	}
	item, err := uc.Repo.FindItemByCode(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	if c := dto.Commercial; c != nil {
		if c.Description != nil {
			item.Commercial.Description = cleanUpdate(*c.Description)
		}
		if c.SaleType != nil {
			item.Commercial.SaleType = cleanUpdate(*c.SaleType)
		}
		if c.VolumeConversionFactor != nil {
			item.Commercial.VolumeConversionFactor = *c.VolumeConversionFactor
		}
		if c.SaleMultiple != nil {
			item.Commercial.SaleMultiple = *c.SaleMultiple
		}
		if c.MinimumSaleQuantity != nil {
			item.Commercial.MinimumSaleQuantity = *c.MinimumSaleQuantity
		}
		if c.EstimatedDeliveryDays != nil {
			item.Commercial.EstimatedDeliveryDays = *c.EstimatedDeliveryDays
		}
		if c.WarrantyDays != nil {
			item.Commercial.WarrantyDays = *c.WarrantyDays
		}
		if c.TransferWarehouseCode != nil {
			item.Commercial.TransferWarehouseCode = *c.TransferWarehouseCode
		}
		if c.TechnicalAssistanceWarehouseCode != nil {
			item.Commercial.TechnicalAssistanceWarehouseCode = *c.TechnicalAssistanceWarehouseCode
		}
		if c.PackagingItemCode != nil {
			item.Commercial.PackagingItemCode = *c.PackagingItemCode
		}
		if c.AllowBillingDescriptionChange != nil {
			item.Commercial.AllowBillingDescriptionChange = *c.AllowBillingDescriptionChange
		}
		if c.IssueLoadingLabels != nil {
			item.Commercial.IssueLoadingLabels = *c.IssueLoadingLabels
		}
		if c.AssembleShippingVolumes != nil {
			item.Commercial.AssembleShippingVolumes = *c.AssembleShippingVolumes
		}
		if c.RequiresSpecialPackaging != nil {
			item.Commercial.RequiresSpecialPackaging = *c.RequiresSpecialPackaging
		}
		if c.WithholdPISCOFINS != nil {
			item.Commercial.WithholdPISCOFINS = *c.WithholdPISCOFINS
		}
		if c.IsPackaging != nil {
			item.Commercial.IsPackaging = *c.IsPackaging
		}
		if c.MobileEnabled != nil {
			item.Commercial.MobileEnabled = *c.MobileEnabled
		}
		if c.ExportPackaging != nil {
			item.Commercial.ExportPackaging = *c.ExportPackaging
		}
		if c.ClassificationCode != nil {
			item.Commercial.ClassificationCode = cleanUpdate(*c.ClassificationCode)
		}
		if c.Notes != nil {
			item.Commercial.Notes = cleanUpdate(*c.Notes)
		}
	}
	if a := dto.Accounting; a != nil {
		if a.SaleFiscalClassificationCode != nil {
			item.Accounting.SaleFiscalClassificationCode = cleanUpdate(*a.SaleFiscalClassificationCode)
		}
		if a.PurchaseFiscalClassificationCode != nil {
			item.Accounting.PurchaseFiscalClassificationCode = cleanUpdate(*a.PurchaseFiscalClassificationCode)
		}
		if a.Origin != nil {
			item.Accounting.Origin = *a.Origin
		}
		if a.SaleIPIType != nil {
			item.Accounting.SaleIPIType = cleanUpdate(*a.SaleIPIType)
		}
		if a.SaleIPIRate != nil {
			item.Accounting.SaleIPIRate = *a.SaleIPIRate
		}
		if a.PurchaseIPIType != nil {
			item.Accounting.PurchaseIPIType = cleanUpdate(*a.PurchaseIPIType)
		}
		if a.PurchaseIPIRate != nil {
			item.Accounting.PurchaseIPIRate = *a.PurchaseIPIRate
		}
		if a.ICMSRate != nil {
			item.Accounting.ICMSRate = *a.ICMSRate
		}
		if a.SaleUnitOfMeasurement != nil {
			item.Accounting.SaleUnitOfMeasurement = *a.SaleUnitOfMeasurement
		}
		if a.PurchaseUnitOfMeasurement != nil {
			item.Accounting.PurchaseUnitOfMeasurement = *a.PurchaseUnitOfMeasurement
		}
		if a.InventoryGroupCode != nil {
			item.Accounting.InventoryGroupCode = *a.InventoryGroupCode
		}
		if a.AccountingClassificationCode != nil {
			item.Accounting.AccountingClassificationCode = cleanUpdate(*a.AccountingClassificationCode)
		}
		if a.CEST != nil {
			item.Accounting.CEST = cleanUpdate(*a.CEST)
		}
		if a.InputCode != nil {
			item.Accounting.InputCode = cleanUpdate(*a.InputCode)
		}
		if a.CalculatePISCOFINS != nil {
			item.Accounting.CalculatePISCOFINS = *a.CalculatePISCOFINS
		}
		if a.Notes != nil {
			item.Accounting.Notes = cleanUpdate(*a.Notes)
		}
	}
	if err := item.Validate(); err != nil {
		return nil, err
	}
	updated, err := uc.Repo.UpdateCommercialAccounting(ctx, item)
	if err != nil {
		return nil, err
	}
	return toItemResponse(updated), nil
}

func cleanUpdate(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}
