package item_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type UpdateItemUseCase struct {
	Repo repository.ItemRepository
	Auth ports.AuthService
	// FiscalCatalog valida as classificações fiscais contra o cadastro canônico
	// (/api/fiscal-classifications). Opcional: sem ele os códigos passam livres.
	FiscalCatalog fiscalClassificationCatalog
}

type updateItemBusinessRepository interface {
	FindItemByBusinessCode(context.Context, valueobject.BusinessCode) (*entity.Item, error)
}

func NewUpdateItemUseCase(repo repository.ItemRepository, auth ports.AuthService, fiscalCatalog ...fiscalClassificationCatalog) *UpdateItemUseCase {
	uc := &UpdateItemUseCase{Repo: repo, Auth: auth}
	if len(fiscalCatalog) > 0 {
		uc.FiscalCatalog = fiscalCatalog[0]
	}
	return uc
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
	return uc.update(ctx, item, dto)
}

func (uc *UpdateItemUseCase) ExecuteBusinessCode(ctx context.Context, rawCode string, dto request.UpdateItemDTO) (*response.ItemResponse, error) {
	if !uc.Auth.CanCreateItem(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	code, err := valueobject.NewBusinessCode(rawCode)
	if err != nil {
		return nil, err
	}
	repo, ok := uc.Repo.(updateItemBusinessRepository)
	if !ok {
		return nil, repository.ErrNotFound
	}
	item, err := repo.FindItemByBusinessCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return uc.update(ctx, item, dto)
}

func (uc *UpdateItemUseCase) update(ctx context.Context, item *entity.Item, dto request.UpdateItemDTO) (*response.ItemResponse, error) {
	applyIdentity(item, dto)
	applyEngineeringFolder(item, dto)
	applyPlanningFolder(item, dto)
	applySuppliesFolder(item, dto)
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
			item.Commercial.TransferWarehouseCode = c.TransferWarehouseCode.Ptr()
		}
		if c.TechnicalAssistanceWarehouseCode != nil {
			item.Commercial.TechnicalAssistanceWarehouseCode = c.TechnicalAssistanceWarehouseCode.Ptr()
		}
		if c.PackagingItemCode != nil {
			// Código de negócio (texto) da tela; nil limpa a embalagem.
			item.Commercial.PackagingItemCode = nil
			item.Commercial.PackagingItemBusinessCode = ""
			if *c.PackagingItemCode != nil {
				item.Commercial.PackagingItemBusinessCode = strings.TrimSpace((*c.PackagingItemCode).String())
			}
			if err := resolveReferenceCodes(ctx, uc.Repo, item); err != nil {
				return nil, err
			}
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
			item.Accounting.CalculatePISCOFINS = a.CalculatePISCOFINS
		}
		if a.Notes != nil {
			item.Accounting.Notes = cleanUpdate(*a.Notes)
		}
	}
	if w := dto.Warehouse; w != nil {
		if w.CyclicalCountConfig.Set {
			item.Warehouse.CyclicalCountConfig = w.CyclicalCountConfig.Value
		}
		if w.UnitOfMeasurement != nil {
			item.Warehouse.UnitOfMeasurement = *w.UnitOfMeasurement
		}
		if w.AutomaticLow != nil {
			item.Warehouse.AutomaticLow = *w.AutomaticLow
		}
		if w.MinimumStock != nil {
			item.Warehouse.MinimumStock = *w.MinimumStock
		}
		if w.WarehouseCode != nil {
			item.Warehouse.WarehouseCode = int(*w.WarehouseCode)
		}
		if w.AverageMonthlyConsumptionManual != nil {
			v := *w.AverageMonthlyConsumptionManual
			item.Warehouse.AverageMonthlyConsumptionManual = &v
		}
	}
	if err := item.Validate(); err != nil {
		return nil, err
	}
	if uc.FiscalCatalog != nil {
		enterpriseID, err := uc.Auth.EnterpriseID(ctx)
		if err != nil {
			return nil, errorsuc.ErrUnauthorized
		}
		if err = validateFiscalClassifications(ctx, uc.FiscalCatalog, enterpriseID, item); err != nil {
			return nil, err
		}
	}
	updated, err := uc.Repo.UpdateFolders(ctx, item)
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

// applyIdentity aplica identificação, marcadores de natureza e PDM. Cada campo é
// ponteiro: ausente preserva o que já está gravado.
func applyIdentity(item *entity.Item, dto request.UpdateItemDTO) {
	if dto.Name != nil {
		item.Name = strings.TrimSpace(*dto.Name)
	}
	if dto.Complement != nil {
		item.Complement = cleanUpdate(*dto.Complement)
	}
	if dto.Nature != nil {
		item.Nature = *dto.Nature
	}
	if dto.IsBase != nil {
		item.IsBase = *dto.IsBase
	}
	if dto.IsConfigured != nil {
		item.IsConfigured = *dto.IsConfigured
	}
	if dto.IsPrototype != nil {
		item.IsPrototype = *dto.IsPrototype
	}
	if dto.IsTool != nil {
		item.IsTool = *dto.IsTool
	}
	if dto.IsProcessItem != nil {
		item.IsProcessItem = *dto.IsProcessItem
	}
	if dto.Situation != nil {
		item.Situation = *dto.Situation
	}
	if dto.Health != nil {
		item.Health = *dto.Health
	}
	if p := dto.PDM; p != nil {
		if p.GroupCode != nil {
			item.PDM.GroupCode = *p.GroupCode
		}
		if p.ModifierCode != nil {
			item.PDM.ModifierCode = *p.ModifierCode
		}
		if p.DescriptionTechnique != nil {
			item.PDM.DescriptionTechnique = strings.TrimSpace(*p.DescriptionTechnique)
		}
	}
}

func applyEngineeringFolder(item *entity.Item, dto request.UpdateItemDTO) {
	e := dto.Engineering
	if e == nil {
		return
	}
	if e.Weight != nil {
		item.Engineering.Weight = *e.Weight
	}
	if e.Dimensions != nil {
		item.Engineering.Dimensions = *e.Dimensions
	}
	if e.Type != nil {
		item.Engineering.Type = *e.Type
	}
	if e.TypeStruct != nil {
		item.Engineering.TypeStruct = *e.TypeStruct
	}
	if e.OEM != nil {
		item.Engineering.OEM = *e.OEM
	}
}

func applyPlanningFolder(item *entity.Item, dto request.UpdateItemDTO) {
	p := dto.Planning
	if p == nil {
		return
	}
	if p.TypeMRP != nil {
		item.Planning.TypeMRP = *p.TypeMRP
	}
	if p.LLC != nil {
		item.Planning.LLC = *p.LLC
	}
	if p.Ghost != nil {
		item.Planning.Ghost = *p.Ghost
	}
	if p.ABCClass != nil {
		item.Planning.ABCClass = cleanUpdate(*p.ABCClass)
	}
	if p.MinimumLot != nil {
		item.Planning.MinimumLot = *p.MinimumLot
	}
	if p.MultipleLot != nil {
		item.Planning.MultipleLot = *p.MultipleLot
	}
	if p.SafetyStock != nil {
		item.Planning.SafetyStock = *p.SafetyStock
	}
	if p.Critical != nil {
		item.Planning.Critical = *p.Critical
	}
	if p.Exclusive != nil {
		item.Planning.Exclusive = *p.Exclusive
	}
	if p.Active != nil {
		item.Planning.Active = *p.Active
	}
}

func applySuppliesFolder(item *entity.Item, dto request.UpdateItemDTO) {
	s := dto.Supplies
	if s == nil {
		return
	}
	if s.TypeOfUse != nil {
		item.Supplies.TypeOfUse = *s.TypeOfUse
	}
	if s.PurchaseUOM != nil {
		item.Supplies.PurchaseUOM = *s.PurchaseUOM
	}
	if s.ReceivingChecklist != nil {
		item.Supplies.ReceivingChecklist = *s.ReceivingChecklist
	}
	if s.Harvest != nil {
		item.Supplies.Harvest = *s.Harvest
	}
	if s.WarehouseCode != nil {
		v := *s.WarehouseCode
		item.Supplies.WarehouseCode = &v
	}
	if s.Notes != nil {
		item.Supplies.Notes = cleanUpdate(s.Notes)
	}
}
