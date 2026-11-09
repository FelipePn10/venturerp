package item

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func (r *RepositoryItemSQLC) Create(
	ctx context.Context,
	item *entity.Item,
) (*entity.Item, error) {
	if err := r.validateFiscalReferences(ctx, item); err != nil {
		return nil, err
	}

	attributes, err := json.Marshal(item.PDM.Attributes)
	if err != nil {
		return nil, fmt.Errorf("marshal pdm_attributes: %w", err)
	}

	weight, err := json.Marshal(item.Engineering.Weight)
	if err != nil {
		return nil, fmt.Errorf("marshal engineering_weight: %w", err)
	}

	dimensions, err := json.Marshal(item.Engineering.Dimensions)
	if err != nil {
		return nil, fmt.Errorf("marshal engineering_dimensions: %w", err)
	}

	reorderPoint, err := json.Marshal(item.Planning.ReorderPoint)
	if err != nil {
		return nil, fmt.Errorf("marshal planning_reorder_point: %w", err)
	}

	cyclicalCountConfig, err := json.Marshal(item.Warehouse.CyclicalCountConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal cyclical_count_config: %w", err)
	}

	item.AplicarPadroes()
	params := sqlc.CreateItemParams{
		EnterpriseID:  item.EnterpriseID,
		BusinessCode:  string(item.BusinessCode),
		WarehouseCode: int64(item.Warehouse.WarehouseCode),
		Name:          item.Name,

		Complement: pgutil.ToPgTextFromPtr(item.Complement),

		Nature:        int16(item.Nature),
		IsBase:        item.IsBase,
		IsConfigured:  item.IsConfigured,
		IsPrototype:   item.IsPrototype,
		IsTool:        item.IsTool,
		IsProcessItem: item.IsProcessItem,
		Situation:     int16(item.Situation),

		Health: sqlc.HealthEnum(item.Health),

		PdmGroupCode:            int64(item.PDM.GroupCode),
		PdmModifierCode:         int64(item.PDM.ModifierCode),
		PdmAttributes:           attributes,
		PdmDescriptionTechnique: item.PDM.DescriptionTechnique,

		WarehouseUnitOfMeasurement:           string(item.Warehouse.UnitOfMeasurement),
		WarehouseAutomaticLow:                item.Warehouse.AutomaticLow,
		WarehouseCyclicalCountConfig:         cyclicalCountConfig,
		WarehouseMinimumStock:                pgutil.ToPgNumericFromDecimal(item.Warehouse.MinimumStock),
		WarehouseAvgMonthlyConsumptionManual: pgutil.ToPgNumericFromDecimalPtr(item.Warehouse.AverageMonthlyConsumptionManual),

		EngineeringItemBaseCode: intPtrToInt64Ptr(item.Engineering.ItemBaseCod),
		EngineeringWeight:       weight,
		EngineeringDimensions:   dimensions,
		EngineeringType:         int16(item.Engineering.Type),
		EngineeringTypeStruct:   int16(item.Engineering.TypeStruct),
		EngineeringOem:          item.Engineering.OEM,

		PlanningTypeMrp:      int16(item.Planning.TypeMRP),
		PlanningLlc:          int32(item.Planning.LLC),
		PlanningReorderPoint: reorderPoint,
		PlanningTankCode:     intPtrToInt64Ptr(item.Planning.TankCode),
		PlanningGhost:        item.Planning.Ghost,
		PlanningAbcClass:     pgutil.ToPgTextFromPtr(item.Planning.ABCClass),
		PlanningMinimumLot:   pgutil.ToPgNumericFromDecimal(item.Planning.MinimumLot),
		PlanningMultipleLot:  pgutil.ToPgNumericFromDecimal(item.Planning.MultipleLot),
		PlanningSafetyStock:  pgutil.ToPgNumericFromDecimal(item.Planning.SafetyStock),
		PlanningCritical:     item.Planning.Critical,
		PlanningExclusive:    item.Planning.Exclusive,
		PlanningActive:       item.Planning.Active,

		SuppliesTypeOfUse:          int16(item.Supplies.TypeOfUse),
		SuppliesPurchaseUom:        unitOfMeasurementToPgText(item.Supplies.PurchaseUOM),
		SuppliesWarehouseCode:      item.Supplies.WarehouseCode,
		SuppliesReceivingChecklist: item.Supplies.ReceivingChecklist,
		SuppliesHarvest:            item.Supplies.Harvest,
		SuppliesNotes:              pgutil.ToPgTextFromPtr(item.Supplies.Notes),

		CommercialWarrantyDays:       int32(item.Commercial.WarrantyDays),
		AccountingCalculatePisCofins: boolPtrToPgBool(item.Accounting.CalculatePISCOFINS),
		CommercialDescription:        pgutil.ToPgTextFromPtr(item.Commercial.Description), CommercialSaleType: pgutil.ToPgTextFromPtr(item.Commercial.SaleType),
		CommercialVolumeConversionFactor: decimalPtrToNumeric(item.Commercial.VolumeConversionFactor), CommercialSaleMultiple: decimalPtrToNumeric(item.Commercial.SaleMultiple),
		CommercialMinimumSaleQuantity: decimalPtrToNumeric(item.Commercial.MinimumSaleQuantity), CommercialEstimatedDeliveryDays: intPtrToInt32Ptr(item.Commercial.EstimatedDeliveryDays),
		CommercialTransferWarehouseCode: pgutil.ToPgTextFromPtr(item.Commercial.TransferWarehouseCode), CommercialTechnicalAssistanceWarehouseCode: pgutil.ToPgTextFromPtr(item.Commercial.TechnicalAssistanceWarehouseCode),
		CommercialPackagingItemCode: item.Commercial.PackagingItemCode, CommercialAllowBillingDescriptionChange: item.Commercial.AllowBillingDescriptionChange,
		CommercialIssueLoadingLabels: item.Commercial.IssueLoadingLabels, CommercialAssembleShippingVolumes: item.Commercial.AssembleShippingVolumes,
		CommercialRequiresSpecialPackaging: item.Commercial.RequiresSpecialPackaging, CommercialWithholdPisCofins: item.Commercial.WithholdPISCOFINS,
		CommercialIsPackaging: item.Commercial.IsPackaging, CommercialMobileEnabled: item.Commercial.MobileEnabled, CommercialExportPackaging: item.Commercial.ExportPackaging,
		CommercialClassificationCode: pgutil.ToPgTextFromPtr(item.Commercial.ClassificationCode), CommercialNotes: pgutil.ToPgTextFromPtr(item.Commercial.Notes),
		AccountingSaleFiscalClassificationCode: pgutil.ToPgTextFromPtr(item.Accounting.SaleFiscalClassificationCode), AccountingPurchaseFiscalClassificationCode: pgutil.ToPgTextFromPtr(item.Accounting.PurchaseFiscalClassificationCode),
		AccountingOrigin: intPtrToInt2(item.Accounting.Origin), AccountingSaleIpiType: pgutil.ToPgTextFromPtr(item.Accounting.SaleIPIType), AccountingSaleIpiRate: decimalPtrToNumeric(item.Accounting.SaleIPIRate),
		AccountingPurchaseIpiType: pgutil.ToPgTextFromPtr(item.Accounting.PurchaseIPIType), AccountingPurchaseIpiRate: decimalPtrToNumeric(item.Accounting.PurchaseIPIRate), AccountingIcmsRate: decimalPtrToNumeric(item.Accounting.ICMSRate),
		AccountingSaleUnitOfMeasurement: unitOfMeasurementToPgText(item.Accounting.SaleUnitOfMeasurement), AccountingPurchaseUnitOfMeasurement: unitOfMeasurementToPgText(item.Accounting.PurchaseUnitOfMeasurement),
		AccountingInventoryGroupCode: item.Accounting.InventoryGroupCode, AccountingClassificationCode: pgutil.ToPgTextFromPtr(item.Accounting.AccountingClassificationCode),
		AccountingCest: pgutil.ToPgTextFromPtr(item.Accounting.CEST), AccountingInputCode: pgutil.ToPgTextFromPtr(item.Accounting.InputCode), AccountingNotes: pgutil.ToPgTextFromPtr(item.Accounting.Notes),

		LongDescription: pgutil.ToPgTextFromPtr(item.LongDescription), GeneralNotes: pgutil.ToPgTextFromPtr(item.GeneralNotes), Revision: pgutil.ToPgTextFromPtr(item.Revision),
		PreviousCode: pgutil.ToPgTextFromPtr(item.PreviousCode), ManufacturerName: pgutil.ToPgTextFromPtr(item.ManufacturerName), ManufacturerPartNumber: pgutil.ToPgTextFromPtr(item.ManufacturerPartNumber),
		PdmSignature: pgutil.ToPgTextFromPtr(item.PDM.Signature), AcceptsFractionalQuantity: item.Warehouse.AcceptsFractionalQuantity,
		WarehouseShelfLifeDays: intPtrToInt32Ptr(item.Warehouse.ShelfLifeDays), WarehouseMinRemainingLifeDays: intPtrToInt32Ptr(item.Warehouse.MinRemainingLifeDays),
		WarehouseStorageConditions: pgutil.ToPgTextFromPtr(item.Warehouse.StorageConditions), WarehouseHazardous: item.Warehouse.Hazardous,
		WarehouseUnNumber: pgutil.ToPgTextFromPtr(item.Warehouse.UNNumber), WarehouseHazardClass: pgutil.ToPgTextFromPtr(item.Warehouse.HazardClass),
		EngineeringVolume: decimalPtrToNumeric(item.Engineering.Volume), PlannerEmployeeCode: item.Planning.PlannerEmployeeCode, PlanningBuyerEmployeeCode: item.Planning.BuyerEmployeeCode,
		PlanningPurchaseLeadTimeDays: intPtrToInt32Ptr(item.Planning.PurchaseLeadTimeDays), PlanningManufacturingLeadTimeDays: intPtrToInt32Ptr(item.Planning.ManufacturingLeadTimeDays),
		PlanningNotes: pgutil.ToPgTextFromPtr(item.Planning.Notes), MaterialIssueTiming: item.Planning.MaterialIssueTiming, ProductionReportingType: item.Planning.ProductionReportingType,
		SuppliesUnderDeliveryTolerancePct: decimalPtrToNumeric(item.Supplies.UnderDeliveryTolerancePct), SuppliesOverDeliveryTolerancePct: decimalPtrToNumeric(item.Supplies.OverDeliveryTolerancePct),
		SuppliesReceiptRouting: item.Supplies.ReceiptRouting, SuppliesClassificationCode: pgutil.ToPgTextFromPtr(item.Supplies.ClassificationCode),
		AccountingGtin: pgutil.ToPgTextFromPtr(item.Accounting.GTIN), AccountingGtinTax: pgutil.ToPgTextFromPtr(item.Accounting.GTINTax), AccountingEfdItemType: pgutil.ToPgTextFromPtr(item.Accounting.EFDItemType),
		AccountingTaxUnit: pgutil.ToPgTextFromPtr(item.Accounting.TaxUnit), AccountingTaxUnitFactor: decimalPtrToNumeric(item.Accounting.TaxUnitFactor),
		IsScrap: item.IsScrap, EngineeringRequiresDrawing: item.Engineering.RequiresDrawing, EngineeringCuttingGrain: item.Engineering.CuttingGrain, EngineeringCuttingNoRotation: item.Engineering.CuttingNoRotation,
		SuppliesRequiresCertificate: item.Supplies.RequiresCertificate, CommercialExclusiveCustomerCode: item.Commercial.ExclusiveCustomerCode, CommercialVolumesPerUnit: intPtrToInt32Ptr(item.Commercial.VolumesPerUnit),
		DescriptionEn: pgutil.ToPgTextFromPtr(item.DescriptionEN), WarehouseSecondaryUom: pgutil.ToPgTextFromPtr(item.Warehouse.SecondaryUOM), WarehouseSecondaryFactor: decimalPtrToNumeric(item.Warehouse.SecondaryFactor),
		AccountingExTipi: pgutil.ToPgTextFromPtr(item.Accounting.ExTIPI), AccountingRelevantScale: boolPtrToPgBool(item.Accounting.RelevantScale), AccountingManufacturerCnpj: pgutil.ToPgTextFromPtr(item.Accounting.ManufacturerCNPJ), AccountingFciNumber: pgutil.ToPgTextFromPtr(item.Accounting.FCINumber),

		CreatedBy: pgutil.ToPgUUID(item.CreatedBy),
	}

	dbItem, err := r.criarComComplementos(ctx, params, item)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, itemrepo.ErrConflict
		}
		if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514") {
			return nil, referenciaInvalida(pgErr, item)
		}
		return nil, fmt.Errorf("create item: %w", err)
	}

	created, err := mapDBItemToEntity(dbItem)
	if err != nil {
		return nil, err
	}
	created.Planning.Extras = item.Planning.Extras
	if err := r.loadEffectiveFiscal(ctx, created); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *RepositoryItemSQLC) ValidatePDMReferences(ctx context.Context, enterpriseID int64, groupCode, modifierCode int) error {
	valid, err := r.q.ValidateItemPDMReferences(ctx, sqlc.ValidateItemPDMReferencesParams{EnterpriseID: enterpriseID, Column2: int64(groupCode), Column3: int64(modifierCode)})
	if err != nil {
		return fmt.Errorf("validar referências PDM: %w", err)
	}
	if !valid.Valid || !valid.Bool {
		return fmt.Errorf("%w: grupo ou modificador PDM não existe na empresa autenticada", itemrepo.ErrInvalidReference)
	}
	return nil
}

func (r *RepositoryItemSQLC) NextAutomaticBusinessCode(ctx context.Context, enterpriseID int64) (valueobject.BusinessCode, error) {
	code, err := r.q.NextAutomaticItemBusinessCode(ctx, enterpriseID)
	if err != nil {
		return "", fmt.Errorf("gerar codigo automatico do item: %w", err)
	}
	return valueobject.NewBusinessCode(code)
}

func (r *RepositoryItemSQLC) UpdateFolders(ctx context.Context, item *entity.Item) (*entity.Item, error) {
	if err := r.validateFiscalReferences(ctx, item); err != nil {
		return nil, err
	}
	cyclicalCountConfig, err := json.Marshal(item.Warehouse.CyclicalCountConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal cyclical_count_config: %w", err)
	}
	weight, err := json.Marshal(item.Engineering.Weight)
	if err != nil {
		return nil, fmt.Errorf("marshal engineering_weight: %w", err)
	}
	dimensions, err := json.Marshal(item.Engineering.Dimensions)
	if err != nil {
		return nil, fmt.Errorf("marshal engineering_dimensions: %w", err)
	}
	attributes, err := json.Marshal(item.PDM.Attributes)
	if err != nil {
		return nil, fmt.Errorf("marshal pdm_attributes: %w", err)
	}
	// Sem ponto de pedido a coluna fica no vazio que a leitura já entende, e não
	// NULL: a alteração parcial não pode mudar a forma do que não foi tocado.
	reorderPoint := []byte("{}")
	if item.Planning.ReorderPoint != nil {
		if reorderPoint, err = json.Marshal(item.Planning.ReorderPoint); err != nil {
			return nil, fmt.Errorf("marshal planning_reorder_point: %w", err)
		}
	}
	if err := r.conferirGTINeResponsaveis(ctx, item); err != nil {
		return nil, err
	}
	if err := r.conferirConflitoDeCodigos(ctx, item, false); err != nil {
		return nil, err
	}
	if err := r.conferirValidadeComLote(ctx, item); err != nil {
		return nil, err
	}
	item.AplicarPadroes()
	p := sqlc.UpdateItemFoldersParams{BusinessCode: string(item.BusinessCode), EnterpriseID: item.EnterpriseID, CommercialDescription: pgutil.ToPgTextFromPtr(item.Commercial.Description), CommercialSaleType: pgutil.ToPgTextFromPtr(item.Commercial.SaleType),
		CommercialVolumeConversionFactor: decimalPtrToNumeric(item.Commercial.VolumeConversionFactor), CommercialSaleMultiple: decimalPtrToNumeric(item.Commercial.SaleMultiple), CommercialMinimumSaleQuantity: decimalPtrToNumeric(item.Commercial.MinimumSaleQuantity), CommercialEstimatedDeliveryDays: intPtrToInt32Ptr(item.Commercial.EstimatedDeliveryDays), CommercialWarrantyDays: int32(item.Commercial.WarrantyDays),
		CommercialTransferWarehouseCode: pgutil.ToPgTextFromPtr(item.Commercial.TransferWarehouseCode), CommercialTechnicalAssistanceWarehouseCode: pgutil.ToPgTextFromPtr(item.Commercial.TechnicalAssistanceWarehouseCode), CommercialPackagingItemCode: item.Commercial.PackagingItemCode,
		CommercialAllowBillingDescriptionChange: item.Commercial.AllowBillingDescriptionChange, CommercialIssueLoadingLabels: item.Commercial.IssueLoadingLabels, CommercialAssembleShippingVolumes: item.Commercial.AssembleShippingVolumes, CommercialRequiresSpecialPackaging: item.Commercial.RequiresSpecialPackaging, CommercialWithholdPisCofins: item.Commercial.WithholdPISCOFINS, CommercialIsPackaging: item.Commercial.IsPackaging, CommercialMobileEnabled: item.Commercial.MobileEnabled, CommercialExportPackaging: item.Commercial.ExportPackaging, CommercialClassificationCode: pgutil.ToPgTextFromPtr(item.Commercial.ClassificationCode), CommercialNotes: pgutil.ToPgTextFromPtr(item.Commercial.Notes),
		AccountingSaleFiscalClassificationCode: pgutil.ToPgTextFromPtr(item.Accounting.SaleFiscalClassificationCode), AccountingPurchaseFiscalClassificationCode: pgutil.ToPgTextFromPtr(item.Accounting.PurchaseFiscalClassificationCode), AccountingOrigin: intPtrToInt2(item.Accounting.Origin), AccountingSaleIpiType: pgutil.ToPgTextFromPtr(item.Accounting.SaleIPIType), AccountingSaleIpiRate: decimalPtrToNumeric(item.Accounting.SaleIPIRate), AccountingPurchaseIpiType: pgutil.ToPgTextFromPtr(item.Accounting.PurchaseIPIType), AccountingPurchaseIpiRate: decimalPtrToNumeric(item.Accounting.PurchaseIPIRate), AccountingIcmsRate: decimalPtrToNumeric(item.Accounting.ICMSRate), AccountingSaleUnitOfMeasurement: unitOfMeasurementToPgText(item.Accounting.SaleUnitOfMeasurement), AccountingPurchaseUnitOfMeasurement: unitOfMeasurementToPgText(item.Accounting.PurchaseUnitOfMeasurement), AccountingInventoryGroupCode: item.Accounting.InventoryGroupCode, AccountingClassificationCode: pgutil.ToPgTextFromPtr(item.Accounting.AccountingClassificationCode), AccountingCest: pgutil.ToPgTextFromPtr(item.Accounting.CEST), AccountingInputCode: pgutil.ToPgTextFromPtr(item.Accounting.InputCode), AccountingCalculatePisCofins: boolPtrToPgBool(item.Accounting.CalculatePISCOFINS), AccountingNotes: pgutil.ToPgTextFromPtr(item.Accounting.Notes), WarehouseCyclicalCountConfig: cyclicalCountConfig,
		// Identificação, marcadores e demais pastas — o que a alteração do item
		// precisava gravar e antes ficava de fora.
		Name: item.Name, Complement: pgutil.ToPgTextFromPtr(item.Complement),
		Nature: int16(item.Nature), IsBase: item.IsBase, IsConfigured: item.IsConfigured, IsPrototype: item.IsPrototype,
		IsTool: item.IsTool, IsProcessItem: item.IsProcessItem,
		Situation: int16(item.Situation), Health: sqlc.HealthEnum(item.Health),
		PdmDescriptionTechnique: item.PDM.DescriptionTechnique, PdmGroupCode: int64(item.PDM.GroupCode), PdmModifierCode: int64(item.PDM.ModifierCode),
		EngineeringWeight: weight, EngineeringDimensions: dimensions,
		EngineeringType: int16(item.Engineering.Type), EngineeringTypeStruct: int16(item.Engineering.TypeStruct), EngineeringOem: item.Engineering.OEM,
		PlanningTypeMrp: int16(item.Planning.TypeMRP), PlanningLlc: int32(item.Planning.LLC), PlanningGhost: item.Planning.Ghost,
		PlanningAbcClass: pgutil.ToPgTextFromPtr(item.Planning.ABCClass), PlanningMinimumLot: pgutil.ToPgNumericFromDecimal(item.Planning.MinimumLot),
		PlanningMultipleLot: pgutil.ToPgNumericFromDecimal(item.Planning.MultipleLot), PlanningSafetyStock: pgutil.ToPgNumericFromDecimal(item.Planning.SafetyStock),
		PlanningCritical: item.Planning.Critical, PlanningExclusive: item.Planning.Exclusive, PlanningActive: item.Planning.Active,
		SuppliesTypeOfUse: int16(item.Supplies.TypeOfUse), SuppliesPurchaseUom: unitOfMeasurementToPgText(item.Supplies.PurchaseUOM),
		SuppliesReceivingChecklist: item.Supplies.ReceivingChecklist, SuppliesHarvest: item.Supplies.Harvest,
		SuppliesWarehouseCode: item.Supplies.WarehouseCode, SuppliesNotes: pgutil.ToPgTextFromPtr(item.Supplies.Notes),
		WarehouseCode: int64(item.Warehouse.WarehouseCode), WarehouseAvgMonthlyConsumptionManual: pgutil.ToPgNumericFromDecimalPtr(item.Warehouse.AverageMonthlyConsumptionManual),
		WarehouseUnitOfMeasurement: string(item.Warehouse.UnitOfMeasurement),
		WarehouseAutomaticLow:      item.Warehouse.AutomaticLow, WarehouseMinimumStock: pgutil.ToPgNumericFromDecimal(item.Warehouse.MinimumStock),
		LongDescription: pgutil.ToPgTextFromPtr(item.LongDescription), GeneralNotes: pgutil.ToPgTextFromPtr(item.GeneralNotes), Revision: pgutil.ToPgTextFromPtr(item.Revision),
		PreviousCode: pgutil.ToPgTextFromPtr(item.PreviousCode), ManufacturerName: pgutil.ToPgTextFromPtr(item.ManufacturerName), ManufacturerPartNumber: pgutil.ToPgTextFromPtr(item.ManufacturerPartNumber),
		PdmSignature: pgutil.ToPgTextFromPtr(item.PDM.Signature), PdmAttributes: attributes, AcceptsFractionalQuantity: item.Warehouse.AcceptsFractionalQuantity,
		WarehouseShelfLifeDays: intPtrToInt32Ptr(item.Warehouse.ShelfLifeDays), WarehouseMinRemainingLifeDays: intPtrToInt32Ptr(item.Warehouse.MinRemainingLifeDays),
		WarehouseStorageConditions: pgutil.ToPgTextFromPtr(item.Warehouse.StorageConditions), WarehouseHazardous: item.Warehouse.Hazardous,
		WarehouseUnNumber: pgutil.ToPgTextFromPtr(item.Warehouse.UNNumber), WarehouseHazardClass: pgutil.ToPgTextFromPtr(item.Warehouse.HazardClass),
		EngineeringVolume: decimalPtrToNumeric(item.Engineering.Volume), PlannerEmployeeCode: item.Planning.PlannerEmployeeCode, PlanningBuyerEmployeeCode: item.Planning.BuyerEmployeeCode,
		PlanningPurchaseLeadTimeDays: intPtrToInt32Ptr(item.Planning.PurchaseLeadTimeDays), PlanningManufacturingLeadTimeDays: intPtrToInt32Ptr(item.Planning.ManufacturingLeadTimeDays),
		PlanningNotes: pgutil.ToPgTextFromPtr(item.Planning.Notes), MaterialIssueTiming: item.Planning.MaterialIssueTiming, ProductionReportingType: item.Planning.ProductionReportingType,
		SuppliesUnderDeliveryTolerancePct: decimalPtrToNumeric(item.Supplies.UnderDeliveryTolerancePct), SuppliesOverDeliveryTolerancePct: decimalPtrToNumeric(item.Supplies.OverDeliveryTolerancePct),
		SuppliesReceiptRouting: item.Supplies.ReceiptRouting, SuppliesClassificationCode: pgutil.ToPgTextFromPtr(item.Supplies.ClassificationCode),
		AccountingGtin: pgutil.ToPgTextFromPtr(item.Accounting.GTIN), AccountingGtinTax: pgutil.ToPgTextFromPtr(item.Accounting.GTINTax), AccountingEfdItemType: pgutil.ToPgTextFromPtr(item.Accounting.EFDItemType),
		AccountingTaxUnit: pgutil.ToPgTextFromPtr(item.Accounting.TaxUnit), AccountingTaxUnitFactor: decimalPtrToNumeric(item.Accounting.TaxUnitFactor),
		IsScrap: item.IsScrap, EngineeringRequiresDrawing: item.Engineering.RequiresDrawing, EngineeringCuttingGrain: item.Engineering.CuttingGrain, EngineeringCuttingNoRotation: item.Engineering.CuttingNoRotation,
		SuppliesRequiresCertificate: item.Supplies.RequiresCertificate, CommercialExclusiveCustomerCode: item.Commercial.ExclusiveCustomerCode, CommercialVolumesPerUnit: intPtrToInt32Ptr(item.Commercial.VolumesPerUnit),
		DescriptionEn: pgutil.ToPgTextFromPtr(item.DescriptionEN), WarehouseSecondaryUom: pgutil.ToPgTextFromPtr(item.Warehouse.SecondaryUOM), WarehouseSecondaryFactor: decimalPtrToNumeric(item.Warehouse.SecondaryFactor),
		AccountingExTipi: pgutil.ToPgTextFromPtr(item.Accounting.ExTIPI), AccountingRelevantScale: boolPtrToPgBool(item.Accounting.RelevantScale), AccountingManufacturerCnpj: pgutil.ToPgTextFromPtr(item.Accounting.ManufacturerCNPJ), AccountingFciNumber: pgutil.ToPgTextFromPtr(item.Accounting.FCINumber),
		PlanningReorderPoint: reorderPoint, PlanningTankCode: intPtrToInt64Ptr(item.Planning.TankCode), EngineeringItemBaseCode: intPtrToInt64Ptr(item.Engineering.ItemBaseCod)}
	dbItem, err := r.atualizarComAutor(ctx, p, item)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514") {
			return nil, referenciaInvalida(pgErr, item)
		}
		return nil, fmt.Errorf("update item folders: %w", err)
	}
	updated, err := mapDBItemToEntity(dbItem)
	if err != nil {
		return nil, err
	}
	updated.Planning.Extras = item.Planning.Extras
	if err := r.loadEffectiveFiscal(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

// atualizarComAutor grava a alteração; com o pool, numa transação que informa
// o autor ao gatilho do histórico do cadastro.
func (r *RepositoryItemSQLC) atualizarComAutor(ctx context.Context, p sqlc.UpdateItemFoldersParams, entidade *entity.Item) (sqlc.Item, error) {
	if r.pool == nil {
		return r.q.UpdateItemFolders(ctx, p)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlc.Item{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('venture.item_actor', $1, true)`, tenant.Ator(ctx)); err != nil {
		return sqlc.Item{}, fmt.Errorf("registrar o autor da alteração do item: %w", err)
	}
	item, err := r.q.WithTx(tx).UpdateItemFolders(ctx, p)
	if err != nil {
		return sqlc.Item{}, err
	}
	if err := gravarComplementos(ctx, tx, item.EnterpriseID, item.Code, entidade); err != nil {
		return sqlc.Item{}, err
	}
	return item, tx.Commit(ctx)
}

func (r *RepositoryItemSQLC) validateFiscalReferences(ctx context.Context, item *entity.Item) error {
	for _, code := range []*string{item.Accounting.SaleFiscalClassificationCode, item.Accounting.PurchaseFiscalClassificationCode} {
		if code == nil {
			continue
		}
		exists, err := r.q.ItemFiscalClassificationExists(ctx, sqlc.ItemFiscalClassificationExistsParams{EnterpriseID: item.EnterpriseID, ClassificationCode: *code})
		if err != nil {
			return fmt.Errorf("validate fiscal classification: %w", err)
		}
		if !exists {
			return fmt.Errorf("%w: fiscal classification %q", itemrepo.ErrInvalidReference, *code)
		}
	}
	return nil
}

func (r *RepositoryItemSQLC) FindItemByCode(
	ctx context.Context,
	code valueobject.ItemCode,
) (*entity.Item, error) {

	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	dbItem, err := r.q.FindItemByCode(ctx, sqlc.FindItemByCodeParams{Code: int64(code), EnterpriseID: enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, itemrepo.ErrNotFound
		}
		return nil, fmt.Errorf("find item by code: %w", err)
	}

	item, err := mapDBItemToEntity(dbItem)
	if err != nil {
		return nil, err
	}
	if err := r.loadEffectiveFiscal(ctx, item); err != nil {
		return nil, err
	}
	if err := r.carregarComplementos(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *RepositoryItemSQLC) FindItemByBusinessCode(ctx context.Context, code valueobject.BusinessCode) (*entity.Item, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	dbItem, err := r.q.FindItemByBusinessCode(ctx, sqlc.FindItemByBusinessCodeParams{BusinessCode: string(code), EnterpriseID: enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, itemrepo.ErrNotFound
		}
		return nil, fmt.Errorf("find item by business code: %w", err)
	}
	item, err := mapDBItemToEntity(dbItem)
	if err != nil {
		return nil, err
	}
	if err := r.loadEffectiveFiscal(ctx, item); err != nil {
		return nil, err
	}
	if err := r.carregarComplementos(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// loadEffectiveFiscal: o que a nota usa do item além da classificação — CEST,
// origem e unidade. As alíquotas do item (IPI, ICMS, PIS/COFINS) não entram no
// cálculo da nota (o motor usa o NCM, a UF e a regra de ICMS/IPI), então não são
// anunciadas aqui como se valessem.
func (r *RepositoryItemSQLC) loadEffectiveFiscal(ctx context.Context, item *entity.Item) error {
	load := func(code *string, purchase bool) (*entity.EffectiveFiscalContext, error) {
		if code == nil {
			return nil, nil
		}
		row, err := r.q.GetEffectiveItemFiscalDefaults(ctx, sqlc.GetEffectiveItemFiscalDefaultsParams{EnterpriseID: item.EnterpriseID, ClassificationCode: *code})
		if err != nil {
			return nil, fmt.Errorf("carregar padroes fiscais efetivos: %w", err)
		}
		c := &entity.EffectiveFiscalContext{ClassificationID: row.ID, ClassificationCode: row.Code, NCM: pgTextToStringPtr(row.Ncm), CEST: pgTextToStringPtr(row.Cest), Origin: int2ToIntPtr(row.DefaultOrigin), Unit: pgTextToStringPtr(row.UnTributacao), IPIRate: numericToDecimalPtr(row.IpiRate), ICMSRate: numericToDecimalPtr(row.DefaultIcmsRate), PISRate: numericToDecimalPtr(row.PisRate), COFINSRate: numericToDecimalPtr(row.CofinsRate), Sources: map[string]entity.FiscalValueSource{}}
		for _, field := range []string{"ncm", "cest", "origin", "unit", "ipi_rate", "icms_rate", "pis_rate", "cofins_rate", "calculate_pis_cofins"} {
			c.Sources[field] = entity.FiscalSourceInherited
		}
		if row.DefaultCalculatePisCofins.Valid {
			v := row.DefaultCalculatePisCofins.Bool
			c.CalculatePISCOFINS = &v
		}
		if item.Accounting.CEST != nil {
			c.CEST = item.Accounting.CEST
			c.Sources["cest"] = entity.FiscalSourceOverride
		}
		if item.Accounting.Origin != nil {
			c.Origin = item.Accounting.Origin
			c.Sources["origin"] = entity.FiscalSourceOverride
		}
		if purchase {
			// A unidade de compra é uma só: a da aba Estoque e compras. A antiga, da
			// aba Contábil, só vale para o item que ainda não tem a de compra.
			if u := item.Supplies.PurchaseUOM; u != nil && u.String() != "" {
				s := u.String()
				c.Unit = &s
				c.Sources["unit"] = entity.FiscalSourceOverride
			} else if item.Accounting.PurchaseUnitOfMeasurement != nil {
				s := item.Accounting.PurchaseUnitOfMeasurement.String()
				c.Unit = &s
				c.Sources["unit"] = entity.FiscalSourceOverride
			}
		} else {
			if item.Accounting.SaleUnitOfMeasurement != nil {
				s := item.Accounting.SaleUnitOfMeasurement.String()
				c.Unit = &s
				c.Sources["unit"] = entity.FiscalSourceOverride
			}
		}
		return c, nil
	}
	var err error
	item.FiscalEffective.Purchase, err = load(item.Accounting.PurchaseFiscalClassificationCode, true)
	if err != nil {
		return err
	}
	item.FiscalEffective.Sale, err = load(item.Accounting.SaleFiscalClassificationCode, false)
	return err
}

func mapDBItemToEntity(
	dbItem sqlc.Item,
) (*entity.Item, error) {

	var complement *string

	if dbItem.Complement.Valid {
		v := dbItem.Complement.String
		complement = &v
	}

	var pdmAttributes []valueobject.Attribute

	if err := json.Unmarshal(dbItem.PdmAttributes, &pdmAttributes); err != nil {
		return nil, fmt.Errorf("unmarshal pdm_attributes: %w", err)
	}

	var engineeringWeight valueobject.Weight

	if err := json.Unmarshal(dbItem.EngineeringWeight, &engineeringWeight); err != nil {
		return nil, fmt.Errorf("unmarshal engineering_weight: %w", err)
	}

	var engineeringDimensions *valueobject.Dimensions

	if hasOptionalJSONValue(dbItem.EngineeringDimensions) {

		var v valueobject.Dimensions

		if err := json.Unmarshal(dbItem.EngineeringDimensions, &v); err != nil {
			return nil, fmt.Errorf("unmarshal engineering_dimensions: %w", err)
		}

		engineeringDimensions = &v
	}

	var planningReorderPoint *valueobject.ReorderPoint

	if hasOptionalJSONValue(dbItem.PlanningReorderPoint) {

		var v valueobject.ReorderPoint

		if err := json.Unmarshal(dbItem.PlanningReorderPoint, &v); err != nil {
			return nil, fmt.Errorf("unmarshal planning_reorder_point: %w", err)
		}

		planningReorderPoint = &v
	}

	var cyclicalCount *valueobject.CyclicalCountConfig

	if hasOptionalJSONValue(dbItem.WarehouseCyclicalCountConfig) {

		var v valueobject.CyclicalCountConfig

		if err := json.Unmarshal(dbItem.WarehouseCyclicalCountConfig, &v); err != nil {
			return nil, fmt.Errorf("unmarshal cyclical_count_config: %w", err)
		}

		cyclicalCount = &v
	}

	return &entity.Item{
		ID:           dbItem.ID,
		Code:         valueobject.ItemCode(dbItem.Code),
		BusinessCode: valueobject.BusinessCode(dbItem.BusinessCode),
		EnterpriseID: dbItem.EnterpriseID,
		Name:         dbItem.Name,
		Complement:   complement,

		LongDescription: pgTextToStringPtr(dbItem.LongDescription), GeneralNotes: pgTextToStringPtr(dbItem.GeneralNotes), Revision: pgTextToStringPtr(dbItem.Revision),
		PreviousCode: pgTextToStringPtr(dbItem.PreviousCode), ManufacturerName: pgTextToStringPtr(dbItem.ManufacturerName), ManufacturerPartNumber: pgTextToStringPtr(dbItem.ManufacturerPartNumber),

		Nature:        entity.ItemNature(dbItem.Nature),
		IsBase:        dbItem.IsBase,
		IsConfigured:  dbItem.IsConfigured,
		IsPrototype:   dbItem.IsPrototype,
		IsTool:        dbItem.IsTool,
		IsProcessItem: dbItem.IsProcessItem,
		IsScrap:       dbItem.IsScrap,
		DescriptionEN: pgTextToStringPtr(dbItem.DescriptionEn),

		PDM: entity.PDM{
			GroupCode:            int32(dbItem.PdmGroupCode),
			ModifierCode:         int32(dbItem.PdmModifierCode),
			Attributes:           pdmAttributes,
			DescriptionTechnique: dbItem.PdmDescriptionTechnique,
			Signature:            pgTextToStringPtr(dbItem.PdmSignature),
		},

		Situation: types.TypeSituationItem(dbItem.Situation),
		Health:    types.Health(dbItem.Health),

		Warehouse: entity.Warehouse{
			WarehouseCode:                   int(dbItem.WarehouseCode),
			UnitOfMeasurement:               types.TypeUnitOfMeasurementItem(dbItem.WarehouseUnitOfMeasurement),
			AutomaticLow:                    dbItem.WarehouseAutomaticLow,
			CyclicalCountConfig:             cyclicalCount,
			MinimumStock:                    pgutil.FromPgNumericToDecimal(dbItem.WarehouseMinimumStock),
			AverageMonthlyConsumptionManual: pgutil.FromPgNumericToDecimalPtr(dbItem.WarehouseAvgMonthlyConsumptionManual),
			ShelfLifeDays:                   int32PtrToIntPtr(dbItem.WarehouseShelfLifeDays),
			MinRemainingLifeDays:            int32PtrToIntPtr(dbItem.WarehouseMinRemainingLifeDays),
			StorageConditions:               pgTextToStringPtr(dbItem.WarehouseStorageConditions),
			Hazardous:                       dbItem.WarehouseHazardous,
			UNNumber:                        pgTextToStringPtr(dbItem.WarehouseUnNumber),
			HazardClass:                     pgTextToStringPtr(dbItem.WarehouseHazardClass),
			AcceptsFractionalQuantity:       dbItem.AcceptsFractionalQuantity,
			SecondaryUOM:                    pgTextToStringPtr(dbItem.WarehouseSecondaryUom),
			SecondaryFactor:                 numericToDecimalPtr(dbItem.WarehouseSecondaryFactor),
		},

		Engineering: entity.Engineering{
			ItemBaseCod: int64PtrToIntPtr(dbItem.EngineeringItemBaseCode),
			Weight:      engineeringWeight,
			Dimensions:  engineeringDimensions,
			Type:        types.TypeItem(dbItem.EngineeringType),
			TypeStruct:  types.TypeStructItem(dbItem.EngineeringTypeStruct),
			OEM:         dbItem.EngineeringOem,
			Volume:      numericToDecimalPtr(dbItem.EngineeringVolume),

			RequiresDrawing: dbItem.EngineeringRequiresDrawing, CuttingGrain: dbItem.EngineeringCuttingGrain, CuttingNoRotation: dbItem.EngineeringCuttingNoRotation,
		},

		Planning: entity.Planning{
			TypeMRP:      types.TypeMRPItem(dbItem.PlanningTypeMrp),
			LLC:          int(dbItem.PlanningLlc),
			ReorderPoint: planningReorderPoint,
			TankCode:     int64PtrToIntPtr(dbItem.PlanningTankCode),
			Ghost:        dbItem.PlanningGhost,
			ABCClass:     pgTextToStringPtr(dbItem.PlanningAbcClass),
			MinimumLot:   pgutil.FromPgNumericToDecimal(dbItem.PlanningMinimumLot),
			MultipleLot:  pgutil.FromPgNumericToDecimal(dbItem.PlanningMultipleLot),
			SafetyStock:  pgutil.FromPgNumericToDecimal(dbItem.PlanningSafetyStock),
			Critical:     dbItem.PlanningCritical,
			Exclusive:    dbItem.PlanningExclusive,
			Active:       dbItem.PlanningActive,

			PlannerEmployeeCode: dbItem.PlannerEmployeeCode, BuyerEmployeeCode: dbItem.PlanningBuyerEmployeeCode,
			PurchaseLeadTimeDays: int32PtrToIntPtr(dbItem.PlanningPurchaseLeadTimeDays), ManufacturingLeadTimeDays: int32PtrToIntPtr(dbItem.PlanningManufacturingLeadTimeDays),
			Notes: pgTextToStringPtr(dbItem.PlanningNotes), MaterialIssueTiming: dbItem.MaterialIssueTiming, ProductionReportingType: dbItem.ProductionReportingType,
		},

		Supplies: entity.Supplies{
			TypeOfUse:          types.TypeOfUseItem(dbItem.SuppliesTypeOfUse),
			PurchaseUOM:        pgTextToUnitOfMeasurementPtr(dbItem.SuppliesPurchaseUom),
			WarehouseCode:      dbItem.SuppliesWarehouseCode,
			ReceivingChecklist: dbItem.SuppliesReceivingChecklist,
			Harvest:            dbItem.SuppliesHarvest,
			Notes:              pgTextToStringPtr(dbItem.SuppliesNotes),

			UnderDeliveryTolerancePct: numericToDecimalPtr(dbItem.SuppliesUnderDeliveryTolerancePct), OverDeliveryTolerancePct: numericToDecimalPtr(dbItem.SuppliesOverDeliveryTolerancePct),
			ReceiptRouting: dbItem.SuppliesReceiptRouting, ClassificationCode: pgTextToStringPtr(dbItem.SuppliesClassificationCode),
			RequiresCertificate: dbItem.SuppliesRequiresCertificate,
		},
		Commercial: entity.Commercial{
			Description: pgTextToStringPtr(dbItem.CommercialDescription), SaleType: pgTextToStringPtr(dbItem.CommercialSaleType),
			VolumeConversionFactor: numericToDecimalPtr(dbItem.CommercialVolumeConversionFactor), SaleMultiple: numericToDecimalPtr(dbItem.CommercialSaleMultiple),
			MinimumSaleQuantity: numericToDecimalPtr(dbItem.CommercialMinimumSaleQuantity), EstimatedDeliveryDays: int32PtrToIntPtr(dbItem.CommercialEstimatedDeliveryDays),
			WarrantyDays: int(dbItem.CommercialWarrantyDays), TransferWarehouseCode: pgTextToStringPtr(dbItem.CommercialTransferWarehouseCode),
			TechnicalAssistanceWarehouseCode: pgTextToStringPtr(dbItem.CommercialTechnicalAssistanceWarehouseCode), PackagingItemCode: dbItem.CommercialPackagingItemCode,
			AllowBillingDescriptionChange: dbItem.CommercialAllowBillingDescriptionChange, IssueLoadingLabels: dbItem.CommercialIssueLoadingLabels,
			AssembleShippingVolumes: dbItem.CommercialAssembleShippingVolumes, RequiresSpecialPackaging: dbItem.CommercialRequiresSpecialPackaging,
			WithholdPISCOFINS: dbItem.CommercialWithholdPisCofins, IsPackaging: dbItem.CommercialIsPackaging,
			MobileEnabled: dbItem.CommercialMobileEnabled, ExportPackaging: dbItem.CommercialExportPackaging,
			ClassificationCode: pgTextToStringPtr(dbItem.CommercialClassificationCode), Notes: pgTextToStringPtr(dbItem.CommercialNotes),
			ExclusiveCustomerCode: dbItem.CommercialExclusiveCustomerCode, VolumesPerUnit: int32PtrToIntPtr(dbItem.CommercialVolumesPerUnit),
		},
		Accounting: entity.Accounting{
			SaleFiscalClassificationCode: pgTextToStringPtr(dbItem.AccountingSaleFiscalClassificationCode), PurchaseFiscalClassificationCode: pgTextToStringPtr(dbItem.AccountingPurchaseFiscalClassificationCode),
			Origin: int2ToIntPtr(dbItem.AccountingOrigin), SaleIPIType: pgTextToStringPtr(dbItem.AccountingSaleIpiType), SaleIPIRate: numericToDecimalPtr(dbItem.AccountingSaleIpiRate),
			PurchaseIPIType: pgTextToStringPtr(dbItem.AccountingPurchaseIpiType), PurchaseIPIRate: numericToDecimalPtr(dbItem.AccountingPurchaseIpiRate), ICMSRate: numericToDecimalPtr(dbItem.AccountingIcmsRate),
			SaleUnitOfMeasurement: pgTextToUnitOfMeasurementPtr(dbItem.AccountingSaleUnitOfMeasurement), PurchaseUnitOfMeasurement: pgTextToUnitOfMeasurementPtr(dbItem.AccountingPurchaseUnitOfMeasurement),
			InventoryGroupCode: dbItem.AccountingInventoryGroupCode, AccountingClassificationCode: pgTextToStringPtr(dbItem.AccountingClassificationCode),
			CEST: pgTextToStringPtr(dbItem.AccountingCest), InputCode: pgTextToStringPtr(dbItem.AccountingInputCode), CalculatePISCOFINS: pgBoolToBoolPtr(dbItem.AccountingCalculatePisCofins), Notes: pgTextToStringPtr(dbItem.AccountingNotes),
			GTIN: pgTextToStringPtr(dbItem.AccountingGtin), GTINTax: pgTextToStringPtr(dbItem.AccountingGtinTax), EFDItemType: pgTextToStringPtr(dbItem.AccountingEfdItemType),
			TaxUnit: pgTextToStringPtr(dbItem.AccountingTaxUnit), TaxUnitFactor: numericToDecimalPtr(dbItem.AccountingTaxUnitFactor),
			ExTIPI: pgTextToStringPtr(dbItem.AccountingExTipi), RelevantScale: pgBoolToBoolPtr(dbItem.AccountingRelevantScale),
			ManufacturerCNPJ: pgTextToStringPtr(dbItem.AccountingManufacturerCnpj), FCINumber: pgTextToStringPtr(dbItem.AccountingFciNumber),
		},

		CreatedBy: pgutil.FromPgUUID(dbItem.CreatedBy),
		CreatedAt: pgutil.FromPgTimestamp(dbItem.CreatedAt),
	}, nil
}

// hasOptionalJSONValue normalizes the absence markers found in optional JSONB
// value objects. Historical rows may contain SQL NULL, JSON null or an empty
// object instead of NULL. Non-empty values remain present so their domain
// validation cannot be bypassed by the persistence mapper.
func hasOptionalJSONValue(raw []byte) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return false
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &object); err == nil && len(object) == 0 {
		return false
	}

	return true
}

func unitOfMeasurementToPgText(value *types.TypeUnitOfMeasurementItem) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value.String(), Valid: true}
}

func boolPtrToPgBool(value *bool) pgtype.Bool {
	if value == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *value, Valid: true}
}

func pgBoolToBoolPtr(value pgtype.Bool) *bool {
	if !value.Valid {
		return nil
	}
	v := value.Bool
	return &v
}

func pgTextToUnitOfMeasurementPtr(value pgtype.Text) *types.TypeUnitOfMeasurementItem {
	if !value.Valid {
		return nil
	}
	unit := types.TypeUnitOfMeasurementItem(value.String)
	return &unit
}

func pgTextToStringPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func int64PtrToPgText(v *int64) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: strconv.FormatInt(*v, 10), Valid: true}
}
func pgTextToInt64Ptr(v pgtype.Text) *int64 {
	if !v.Valid {
		return nil
	}
	n, err := strconv.ParseInt(v.String, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func decimalPtrToNumeric(v *decimal.Decimal) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}
	return pgutil.ToPgNumericFromString(v.String())
}
func numericToDecimalPtr(v pgtype.Numeric) *decimal.Decimal {
	if !v.Valid {
		return nil
	}
	d, err := decimal.NewFromString(pgutil.FromPgNumericToString(v))
	if err != nil {
		return nil
	}
	return &d
}
func intPtrToInt2(v *int) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: int16(*v), Valid: true}
}
func int2ToIntPtr(v pgtype.Int2) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int16)
	return &n
}

func intPtrToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}

	value := int32(*v)

	return &value
}

func int32PtrToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}

	value := int(*v)

	return &value
}

func intPtrToInt64Ptr(v *int) *int64 {
	if v == nil {
		return nil
	}

	value := int64(*v)

	return &value
}

func int64PtrToIntPtr(v *int64) *int {
	if v == nil {
		return nil
	}

	value := int(*v)

	return &value
}

func (r *RepositoryItemSQLC) ListAll(ctx context.Context) ([]*entity.Item, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	dbItems, err := r.q.ListItems(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}

	result := make([]*entity.Item, 0, len(dbItems))
	for _, dbItem := range dbItems {
		item, err := mapDBItemToEntity(sqlc.Item(dbItem))
		if err != nil {
			return nil, err
		}
		if err := r.loadEffectiveFiscal(ctx, item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *RepositoryItemSQLC) ListAllWithMasks(ctx context.Context) ([]entity.ItemWithMasks, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	dbItems, err := r.q.ListItems(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}

	dbMasks, err := r.q.ListAllItemMasks(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("list masks: %w", err)
	}

	// index masks by item_code (items.code)
	masksByCode := make(map[int64][]entity.MaskSummary, len(dbMasks))
	for _, m := range dbMasks {
		var createdAt time.Time
		if m.CreatedAt.Valid {
			createdAt = m.CreatedAt.Time
		}
		masksByCode[m.ItemCode] = append(masksByCode[m.ItemCode], entity.MaskSummary{
			ID:        m.ID,
			Mask:      m.Mask,
			MaskHash:  m.MaskHash,
			CreatedAt: createdAt,
		})
	}

	result := make([]entity.ItemWithMasks, 0, len(dbItems))
	for _, dbItem := range dbItems {
		item, err := mapDBItemToEntity(sqlc.Item(dbItem))
		if err != nil {
			return nil, err
		}
		result = append(result, entity.ItemWithMasks{
			Item:  item,
			Masks: masksByCode[dbItem.Code],
		})
	}
	return result, nil
}

// referenciaInvalida explica a referência recusada pelo banco. A unidade de
// estoque tem de existir no catálogo de unidades da empresa (migração 474).
func referenciaInvalida(pgErr *pgconn.PgError, item *entity.Item) error {
	if pgErr.ConstraintName == "fk_items_warehouse_uom" {
		return fmt.Errorf("%w: a unidade de estoque %s não está cadastrada no catálogo de unidades — cadastre-a em VSUP0110, aba Unidades",
			itemrepo.ErrInvalidReference, item.Warehouse.UnitOfMeasurement)
	}
	return fmt.Errorf("%w: %s", itemrepo.ErrInvalidReference, pgErr.ConstraintName)
}
