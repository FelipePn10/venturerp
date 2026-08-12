package item

import (
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

	params := sqlc.CreateItemParams{
		EnterpriseID:  item.EnterpriseID,
		BusinessCode:  string(item.BusinessCode),
		WarehouseCode: int64(item.Warehouse.WarehouseCode),
		Name:          item.Name,

		Complement: pgutil.ToPgTextFromPtr(item.Complement),

		Nature:    int16(item.Nature),
		Situation: int16(item.Situation),

		Health: sqlc.HealthEnum(item.Health),

		PdmGroupCode:            int64(item.PDM.GroupCode),
		PdmModifierCode:         int64(item.PDM.ModifierCode),
		PdmAttributes:           attributes,
		PdmDescriptionTechnique: item.PDM.DescriptionTechnique,

		WarehouseUnitOfMeasurement:           sqlc.UnitOfMeasurementEnum(item.Warehouse.UnitOfMeasurement),
		WarehouseAutomaticLow:                item.Warehouse.AutomaticLow,
		WarehouseCyclicalCountConfig:         cyclicalCountConfig,
		WarehouseMinimumStock:                item.Warehouse.MinimumStock,
		WarehouseAvgMonthlyConsumptionManual: intPtrToInt32Ptr(item.Warehouse.AverageMonthlyConsumptionManual),

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
		PlanningMinimumLot:   item.Planning.MinimumLot,
		PlanningMultipleLot:  item.Planning.MultipleLot,
		PlanningSafetyStock:  item.Planning.SafetyStock,
		PlanningCritical:     item.Planning.Critical,
		PlanningExclusive:    item.Planning.Exclusive,
		PlanningActive:       item.Planning.Active,

		SuppliesTypeOfUse:          int16(item.Supplies.TypeOfUse),
		SuppliesPurchaseUom:        unitOfMeasurementToPgText(item.Supplies.PurchaseUOM),
		SuppliesWarehouseCode:      item.Supplies.WarehouseCode,
		SuppliesReceivingChecklist: item.Supplies.ReceivingChecklist,
		SuppliesHarvest:            item.Supplies.Harvest,

		CommercialWarrantyDays:       int32(item.Commercial.WarrantyDays),
		AccountingCalculatePisCofins: item.Accounting.CalculatePISCOFINS,
		CommercialDescription:        pgutil.ToPgTextFromPtr(item.Commercial.Description), CommercialSaleType: pgutil.ToPgTextFromPtr(item.Commercial.SaleType),
		CommercialVolumeConversionFactor: decimalPtrToNumeric(item.Commercial.VolumeConversionFactor), CommercialSaleMultiple: decimalPtrToNumeric(item.Commercial.SaleMultiple),
		CommercialMinimumSaleQuantity: decimalPtrToNumeric(item.Commercial.MinimumSaleQuantity), CommercialEstimatedDeliveryDays: intPtrToInt32Ptr(item.Commercial.EstimatedDeliveryDays),
		CommercialTransferWarehouseCode: int64PtrToPgText(item.Commercial.TransferWarehouseCode), CommercialTechnicalAssistanceWarehouseCode: int64PtrToPgText(item.Commercial.TechnicalAssistanceWarehouseCode),
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

		CreatedBy: pgutil.ToPgUUID(item.CreatedBy),
	}

	dbItem, err := r.q.CreateItem(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, itemrepo.ErrConflict
		}
		if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514") {
			return nil, fmt.Errorf("%w: %s", itemrepo.ErrInvalidReference, pgErr.ConstraintName)
		}
		return nil, fmt.Errorf("create item: %w", err)
	}

	return mapDBItemToEntity(dbItem)
}

func (r *RepositoryItemSQLC) UpdateCommercialAccounting(ctx context.Context, item *entity.Item) (*entity.Item, error) {
	if err := r.validateFiscalReferences(ctx, item); err != nil {
		return nil, err
	}
	p := sqlc.UpdateItemCommercialAccountingParams{BusinessCode: string(item.BusinessCode), EnterpriseID: item.EnterpriseID, CommercialDescription: pgutil.ToPgTextFromPtr(item.Commercial.Description), CommercialSaleType: pgutil.ToPgTextFromPtr(item.Commercial.SaleType),
		CommercialVolumeConversionFactor: decimalPtrToNumeric(item.Commercial.VolumeConversionFactor), CommercialSaleMultiple: decimalPtrToNumeric(item.Commercial.SaleMultiple), CommercialMinimumSaleQuantity: decimalPtrToNumeric(item.Commercial.MinimumSaleQuantity), CommercialEstimatedDeliveryDays: intPtrToInt32Ptr(item.Commercial.EstimatedDeliveryDays), CommercialWarrantyDays: int32(item.Commercial.WarrantyDays),
		CommercialTransferWarehouseCode: int64PtrToPgText(item.Commercial.TransferWarehouseCode), CommercialTechnicalAssistanceWarehouseCode: int64PtrToPgText(item.Commercial.TechnicalAssistanceWarehouseCode), CommercialPackagingItemCode: item.Commercial.PackagingItemCode,
		CommercialAllowBillingDescriptionChange: item.Commercial.AllowBillingDescriptionChange, CommercialIssueLoadingLabels: item.Commercial.IssueLoadingLabels, CommercialAssembleShippingVolumes: item.Commercial.AssembleShippingVolumes, CommercialRequiresSpecialPackaging: item.Commercial.RequiresSpecialPackaging, CommercialWithholdPisCofins: item.Commercial.WithholdPISCOFINS, CommercialIsPackaging: item.Commercial.IsPackaging, CommercialMobileEnabled: item.Commercial.MobileEnabled, CommercialExportPackaging: item.Commercial.ExportPackaging, CommercialClassificationCode: pgutil.ToPgTextFromPtr(item.Commercial.ClassificationCode), CommercialNotes: pgutil.ToPgTextFromPtr(item.Commercial.Notes),
		AccountingSaleFiscalClassificationCode: pgutil.ToPgTextFromPtr(item.Accounting.SaleFiscalClassificationCode), AccountingPurchaseFiscalClassificationCode: pgutil.ToPgTextFromPtr(item.Accounting.PurchaseFiscalClassificationCode), AccountingOrigin: intPtrToInt2(item.Accounting.Origin), AccountingSaleIpiType: pgutil.ToPgTextFromPtr(item.Accounting.SaleIPIType), AccountingSaleIpiRate: decimalPtrToNumeric(item.Accounting.SaleIPIRate), AccountingPurchaseIpiType: pgutil.ToPgTextFromPtr(item.Accounting.PurchaseIPIType), AccountingPurchaseIpiRate: decimalPtrToNumeric(item.Accounting.PurchaseIPIRate), AccountingIcmsRate: decimalPtrToNumeric(item.Accounting.ICMSRate), AccountingSaleUnitOfMeasurement: unitOfMeasurementToPgText(item.Accounting.SaleUnitOfMeasurement), AccountingPurchaseUnitOfMeasurement: unitOfMeasurementToPgText(item.Accounting.PurchaseUnitOfMeasurement), AccountingInventoryGroupCode: item.Accounting.InventoryGroupCode, AccountingClassificationCode: pgutil.ToPgTextFromPtr(item.Accounting.AccountingClassificationCode), AccountingCest: pgutil.ToPgTextFromPtr(item.Accounting.CEST), AccountingInputCode: pgutil.ToPgTextFromPtr(item.Accounting.InputCode), AccountingCalculatePisCofins: item.Accounting.CalculatePISCOFINS, AccountingNotes: pgutil.ToPgTextFromPtr(item.Accounting.Notes)}
	dbItem, err := r.q.UpdateItemCommercialAccounting(ctx, p)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514") {
			return nil, fmt.Errorf("%w: %s", itemrepo.ErrInvalidReference, pgErr.ConstraintName)
		}
		return nil, fmt.Errorf("update item folders: %w", err)
	}
	return mapDBItemToEntity(dbItem)
}

func (r *RepositoryItemSQLC) validateFiscalReferences(ctx context.Context, item *entity.Item) error {
	for _, code := range []*string{item.Accounting.SaleFiscalClassificationCode, item.Accounting.PurchaseFiscalClassificationCode} {
		if code == nil {
			continue
		}
		exists, err := r.q.ItemFiscalClassificationExists(ctx, *code)
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

	return mapDBItemToEntity(dbItem)
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
	return mapDBItemToEntity(dbItem)
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

	if len(dbItem.EngineeringDimensions) > 0 {

		var v valueobject.Dimensions

		if err := json.Unmarshal(dbItem.EngineeringDimensions, &v); err != nil {
			return nil, fmt.Errorf("unmarshal engineering_dimensions: %w", err)
		}

		engineeringDimensions = &v
	}

	var planningReorderPoint *valueobject.ReorderPoint

	if len(dbItem.PlanningReorderPoint) > 0 {

		var v valueobject.ReorderPoint

		if err := json.Unmarshal(dbItem.PlanningReorderPoint, &v); err != nil {
			return nil, fmt.Errorf("unmarshal planning_reorder_point: %w", err)
		}

		planningReorderPoint = &v
	}

	var cyclicalCount *valueobject.CyclicalCountConfig

	if len(dbItem.WarehouseCyclicalCountConfig) > 0 {

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

		Nature: entity.ItemNature(dbItem.Nature),

		PDM: entity.PDM{
			GroupCode:            int32(dbItem.PdmGroupCode),
			ModifierCode:         int32(dbItem.PdmModifierCode),
			Attributes:           pdmAttributes,
			DescriptionTechnique: dbItem.PdmDescriptionTechnique,
		},

		Situation: types.TypeSituationItem(dbItem.Situation),
		Health:    types.Health(dbItem.Health),

		Warehouse: entity.Warehouse{
			WarehouseCode:                   int(dbItem.WarehouseCode),
			UnitOfMeasurement:               types.TypeUnitOfMeasurementItem(dbItem.WarehouseUnitOfMeasurement),
			AutomaticLow:                    dbItem.WarehouseAutomaticLow,
			CyclicalCountConfig:             cyclicalCount,
			MinimumStock:                    dbItem.WarehouseMinimumStock,
			AverageMonthlyConsumptionManual: int32PtrToIntPtr(dbItem.WarehouseAvgMonthlyConsumptionManual),
		},

		Engineering: entity.Engineering{
			ItemBaseCod: int64PtrToIntPtr(dbItem.EngineeringItemBaseCode),
			Weight:      engineeringWeight,
			Dimensions:  engineeringDimensions,
			Type:        types.TypeItem(dbItem.EngineeringType),
			TypeStruct:  types.TypeStructItem(dbItem.EngineeringTypeStruct),
			OEM:         dbItem.EngineeringOem,
		},

		Planning: entity.Planning{
			TypeMRP:      types.TypeMRPItem(dbItem.PlanningTypeMrp),
			LLC:          int(dbItem.PlanningLlc),
			ReorderPoint: planningReorderPoint,
			TankCode:     int64PtrToIntPtr(dbItem.PlanningTankCode),
			Ghost:        dbItem.PlanningGhost,
			ABCClass:     pgTextToStringPtr(dbItem.PlanningAbcClass),
			MinimumLot:   dbItem.PlanningMinimumLot,
			MultipleLot:  dbItem.PlanningMultipleLot,
			SafetyStock:  dbItem.PlanningSafetyStock,
			Critical:     dbItem.PlanningCritical,
			Exclusive:    dbItem.PlanningExclusive,
			Active:       dbItem.PlanningActive,
		},

		Supplies: entity.Supplies{
			TypeOfUse:          types.TypeOfUseItem(dbItem.SuppliesTypeOfUse),
			PurchaseUOM:        pgTextToUnitOfMeasurementPtr(dbItem.SuppliesPurchaseUom),
			WarehouseCode:      dbItem.SuppliesWarehouseCode,
			ReceivingChecklist: dbItem.SuppliesReceivingChecklist,
			Harvest:            dbItem.SuppliesHarvest,
		},
		Commercial: entity.Commercial{
			Description: pgTextToStringPtr(dbItem.CommercialDescription), SaleType: pgTextToStringPtr(dbItem.CommercialSaleType),
			VolumeConversionFactor: numericToDecimalPtr(dbItem.CommercialVolumeConversionFactor), SaleMultiple: numericToDecimalPtr(dbItem.CommercialSaleMultiple),
			MinimumSaleQuantity: numericToDecimalPtr(dbItem.CommercialMinimumSaleQuantity), EstimatedDeliveryDays: int32PtrToIntPtr(dbItem.CommercialEstimatedDeliveryDays),
			WarrantyDays: int(dbItem.CommercialWarrantyDays), TransferWarehouseCode: pgTextToInt64Ptr(dbItem.CommercialTransferWarehouseCode),
			TechnicalAssistanceWarehouseCode: pgTextToInt64Ptr(dbItem.CommercialTechnicalAssistanceWarehouseCode), PackagingItemCode: dbItem.CommercialPackagingItemCode,
			AllowBillingDescriptionChange: dbItem.CommercialAllowBillingDescriptionChange, IssueLoadingLabels: dbItem.CommercialIssueLoadingLabels,
			AssembleShippingVolumes: dbItem.CommercialAssembleShippingVolumes, RequiresSpecialPackaging: dbItem.CommercialRequiresSpecialPackaging,
			WithholdPISCOFINS: dbItem.CommercialWithholdPisCofins, IsPackaging: dbItem.CommercialIsPackaging,
			MobileEnabled: dbItem.CommercialMobileEnabled, ExportPackaging: dbItem.CommercialExportPackaging,
			ClassificationCode: pgTextToStringPtr(dbItem.CommercialClassificationCode), Notes: pgTextToStringPtr(dbItem.CommercialNotes),
		},
		Accounting: entity.Accounting{
			SaleFiscalClassificationCode: pgTextToStringPtr(dbItem.AccountingSaleFiscalClassificationCode), PurchaseFiscalClassificationCode: pgTextToStringPtr(dbItem.AccountingPurchaseFiscalClassificationCode),
			Origin: int2ToIntPtr(dbItem.AccountingOrigin), SaleIPIType: pgTextToStringPtr(dbItem.AccountingSaleIpiType), SaleIPIRate: numericToDecimalPtr(dbItem.AccountingSaleIpiRate),
			PurchaseIPIType: pgTextToStringPtr(dbItem.AccountingPurchaseIpiType), PurchaseIPIRate: numericToDecimalPtr(dbItem.AccountingPurchaseIpiRate), ICMSRate: numericToDecimalPtr(dbItem.AccountingIcmsRate),
			SaleUnitOfMeasurement: pgTextToUnitOfMeasurementPtr(dbItem.AccountingSaleUnitOfMeasurement), PurchaseUnitOfMeasurement: pgTextToUnitOfMeasurementPtr(dbItem.AccountingPurchaseUnitOfMeasurement),
			InventoryGroupCode: dbItem.AccountingInventoryGroupCode, AccountingClassificationCode: pgTextToStringPtr(dbItem.AccountingClassificationCode),
			CEST: pgTextToStringPtr(dbItem.AccountingCest), InputCode: pgTextToStringPtr(dbItem.AccountingInputCode), CalculatePISCOFINS: dbItem.AccountingCalculatePisCofins, Notes: pgTextToStringPtr(dbItem.AccountingNotes),
		},

		CreatedBy: pgutil.FromPgUUID(dbItem.CreatedBy),
		CreatedAt: pgutil.FromPgTimestamp(dbItem.CreatedAt),
	}, nil
}

func unitOfMeasurementToPgText(value *types.TypeUnitOfMeasurementItem) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value.String(), Valid: true}
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

	dbMasks, err := r.q.ListAllItemMasks(ctx)
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
