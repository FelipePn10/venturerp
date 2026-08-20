package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Item struct {
	ID           int64
	Code         valueobject.ItemCode // identificador numerico legado; nao expor como codigo de negocio
	BusinessCode valueobject.BusinessCode
	EnterpriseID int64
	Name         string
	Complement   *string

	// Checkbox
	Nature ItemNature
	//---- PDM
	PDM PDM

	Situation types.TypeSituationItem
	Health    types.Health

	// --- Pastas:
	// Almoxarifado
	Warehouse Warehouse
	// Engenharia
	Engineering Engineering
	// Planejamento
	Planning Planning
	// Suprimentos
	Supplies Supplies
	// Comercial
	Commercial Commercial
	// Contábil/Fiscal
	Accounting      Accounting
	FiscalEffective FiscalEffective
	//Status    types.Status

	CreatedBy uuid.UUID
	CreatedAt time.Time
}

type FiscalValueSource string

const (
	FiscalSourceInherited FiscalValueSource = "HERDADO"
	FiscalSourceOverride  FiscalValueSource = "SOBRESCRITO"
)

type EffectiveFiscalContext struct {
	ClassificationID                       int64
	ClassificationCode                     int64
	NCM, CEST, Unit                        *string
	Origin                                 *int
	IPIRate, ICMSRate, PISRate, COFINSRate *decimal.Decimal
	CalculatePISCOFINS                     *bool
	Sources                                map[string]FiscalValueSource
}
type FiscalEffective struct{ Purchase, Sale *EffectiveFiscalContext }

// PDM
type PDM struct {
	GroupCode    int32                   // Grupo de um item, ex: CHAPAS, AÇÕS etc
	ModifierCode int32                   // Compor a descrição do item, ex: Grupo: CHAPAS Modificador: Chapa Aço Retax
	Attributes   []valueobject.Attribute // "nome" para compor, ex: Grupo: CHAPAS Modificador: Chapa Aço Retax Nome: Retax 5MM
	// PDM gera a descrição tecnica:
	DescriptionTechnique string
}

// Pastas
type Warehouse struct {
	WarehouseCode                   int
	UnitOfMeasurement               types.TypeUnitOfMeasurementItem // Qual unidade de medida será armazenada para esse item
	AutomaticLow                    bool                            // Faz baixa autom?
	CyclicalCountConfig             *valueobject.CyclicalCountConfig
	MinimumStock                    int32 // Estoque mínimo para alerta de compra
	AverageMonthlyConsumptionManual *int  // Consumo médio mensal inserido manualmente ou se for nil gera um calculo de consumo médio/mês
}

type Engineering struct {
	ItemBaseCod *int // Somente se ItemBase (checkbox) for false
	Weight      valueobject.Weight

	Dimensions *valueobject.Dimensions

	Type       types.TypeItem
	TypeStruct types.TypeStructItem
	OEM        bool // componentes ou produtos que são fabricados/montados sob a marca de outra empresa e revendidos pela empresa contratante do sistema
}

type Planning struct {
	// Para o MRP calcular e gerar ordem de máteria prima, o nivél deve ser LLC 9 e ser ACTIVE
	TypeMRP      types.TypeMRPItem
	LLC          int // niveis 1 para o produto final, 2 há 8 para estruras e conjuntos e 9 sendo para matérias primas
	ReorderPoint *valueobject.ReorderPoint
	TankCode     *int // Setor onde é feito
	Ghost        bool
	ABCClass     *string
	MinimumLot   int64
	MultipleLot  int64
	SafetyStock  int64
	Critical     bool
	Exclusive    bool
	Active       bool
}

type Supplies struct {
	TypeOfUse          types.TypeOfUseItem
	PurchaseUOM        *types.TypeUnitOfMeasurementItem
	WarehouseCode      *int64
	ReceivingChecklist bool
	Harvest            bool
}

type Commercial struct {
	Description                      *string
	SaleType                         *string
	VolumeConversionFactor           *decimal.Decimal
	SaleMultiple                     *decimal.Decimal
	MinimumSaleQuantity              *decimal.Decimal
	EstimatedDeliveryDays            *int
	WarrantyDays                     int
	TransferWarehouseCode            *int64
	TechnicalAssistanceWarehouseCode *int64
	PackagingItemCode                *int64
	AllowBillingDescriptionChange    bool
	IssueLoadingLabels               bool
	AssembleShippingVolumes          bool
	RequiresSpecialPackaging         bool
	WithholdPISCOFINS                bool
	IsPackaging                      bool
	MobileEnabled                    bool
	ExportPackaging                  bool
	ClassificationCode               *string
	Notes                            *string
}

type Accounting struct {
	SaleFiscalClassificationCode     *string
	PurchaseFiscalClassificationCode *string
	Origin                           *int
	SaleIPIType                      *string
	SaleIPIRate                      *decimal.Decimal
	PurchaseIPIType                  *string
	PurchaseIPIRate                  *decimal.Decimal
	ICMSRate                         *decimal.Decimal
	SaleUnitOfMeasurement            *types.TypeUnitOfMeasurementItem
	PurchaseUnitOfMeasurement        *types.TypeUnitOfMeasurementItem
	InventoryGroupCode               *int64
	AccountingClassificationCode     *string
	CEST                             *string
	InputCode                        *string
	CalculatePISCOFINS               *bool
	Notes                            *string
}

type ItemNature int

const (
	ItemGeneric ItemNature = iota
	ItemConfigured
	ItemBase
)

type MaskSummary struct {
	ID        int64
	Mask      string
	MaskHash  string
	CreatedAt time.Time
}

type ItemWithMasks struct {
	Item  *Item
	Masks []MaskSummary
}

func (i *Item) Validate() error {
	// Compatibilidade para entidades internas legadas enquanto as referencias
	// numericas migram para item_id.
	if i.BusinessCode == "" && i.Code.IsValid() {
		i.BusinessCode = valueobject.BusinessCode(fmt.Sprintf("%d", i.Code))
	}
	if !i.BusinessCode.IsValid() {
		return errors.New("invalid code")
	}
	i.Name = strings.TrimSpace(i.Name)
	if i.Name == "" {
		return errors.New("Informe o nome do item.")
	}
	if i.Engineering.ItemBaseCod != nil && *i.Engineering.ItemBaseCod == 0 {
		i.Engineering.ItemBaseCod = nil
	}
	if !i.Situation.IsValid() || !i.Health.IsValid() || !i.Warehouse.UnitOfMeasurement.IsValid() ||
		!i.Engineering.Type.IsValid() || !i.Engineering.TypeStruct.IsValid() ||
		!i.Planning.TypeMRP.IsValid() || !i.Supplies.TypeOfUse.IsValid() {
		return errors.New("invalid item enum value")
	}
	if i.Supplies.PurchaseUOM != nil && !i.Supplies.PurchaseUOM.IsValid() {
		return errors.New("invalid purchase unit of measurement")
	}
	if i.Planning.MinimumLot < 0 || i.Planning.MultipleLot < 0 || i.Planning.SafetyStock < 0 || i.Commercial.WarrantyDays < 0 {
		return errors.New("planning quantities and warranty days cannot be negative")
	}
	if i.Commercial.SaleType != nil && *i.Commercial.SaleType != "VENDA" && *i.Commercial.SaleType != "REVENDA" {
		return errors.New("commercial.sale_type must be VENDA or REVENDA")
	}
	for name, value := range map[string]*decimal.Decimal{
		"commercial.volume_conversion_factor": i.Commercial.VolumeConversionFactor,
		"commercial.sale_multiple":            i.Commercial.SaleMultiple,
	} {
		if value != nil && !value.IsPositive() {
			return errors.New(name + " must be greater than zero")
		}
	}
	if i.Commercial.MinimumSaleQuantity != nil && i.Commercial.MinimumSaleQuantity.IsNegative() {
		return errors.New("commercial.minimum_sale_quantity cannot be negative")
	}
	if i.Commercial.EstimatedDeliveryDays != nil && *i.Commercial.EstimatedDeliveryDays < 0 {
		return errors.New("commercial.estimated_delivery_days cannot be negative")
	}
	if i.Code.IsValid() && i.Commercial.PackagingItemCode != nil && *i.Commercial.PackagingItemCode == int64(i.Code) {
		return errors.New("commercial.packaging_item_code cannot reference the item itself")
	}
	if i.Accounting.Origin != nil && (*i.Accounting.Origin < 0 || *i.Accounting.Origin > 8) {
		return errors.New("accounting.origin must be between 0 and 8")
	}
	for _, value := range []*string{i.Accounting.SaleIPIType, i.Accounting.PurchaseIPIType} {
		if value != nil && *value != "PERCENTUAL" && *value != "VALOR" {
			return errors.New("accounting IPI type must be PERCENTUAL or VALOR")
		}
	}
	for name, value := range map[string]*decimal.Decimal{"accounting.sale_ipi_rate": i.Accounting.SaleIPIRate, "accounting.purchase_ipi_rate": i.Accounting.PurchaseIPIRate, "accounting.icms_rate": i.Accounting.ICMSRate} {
		if value != nil && value.IsNegative() {
			return errors.New(name + " cannot be negative")
		}
	}
	if i.Accounting.SaleUnitOfMeasurement != nil && !i.Accounting.SaleUnitOfMeasurement.IsValid() {
		return errors.New("invalid accounting.sale_unit_of_measurement")
	}
	if i.Accounting.PurchaseUnitOfMeasurement != nil && !i.Accounting.PurchaseUnitOfMeasurement.IsValid() {
		return errors.New("invalid accounting.purchase_unit_of_measurement")
	}
	if i.Accounting.CEST != nil {
		if len(*i.Accounting.CEST) != 7 {
			return errors.New("accounting.cest must contain exactly 7 digits")
		}
		for _, c := range *i.Accounting.CEST {
			if c < '0' || c > '9' {
				return errors.New("accounting.cest must contain exactly 7 digits")
			}
		}
	}
	for name, field := range map[string]struct {
		value *string
		max   int
	}{
		"commercial.description": {i.Commercial.Description, 255}, "commercial.classification_code": {i.Commercial.ClassificationCode, 40}, "commercial.notes": {i.Commercial.Notes, 1000},
		"accounting.sale_fiscal_classification_code": {i.Accounting.SaleFiscalClassificationCode, 40}, "accounting.purchase_fiscal_classification_code": {i.Accounting.PurchaseFiscalClassificationCode, 40},
		"accounting.accounting_classification_code": {i.Accounting.AccountingClassificationCode, 80}, "accounting.input_code": {i.Accounting.InputCode, 20}, "accounting.notes": {i.Accounting.Notes, 1000},
	} {
		if field.value != nil && len(*field.value) > field.max {
			return errors.New(name + " exceeds maximum length")
		}
	}
	if i.Planning.ABCClass != nil && *i.Planning.ABCClass != "A" && *i.Planning.ABCClass != "B" && *i.Planning.ABCClass != "C" {
		return errors.New("invalid ABC class")
	}

	if i.Engineering.Dimensions != nil && !i.Engineering.Dimensions.IsValid() {
		return errors.New("invalid dimensions")
	}
	if i.Warehouse.CyclicalCountConfig != nil && !i.Warehouse.CyclicalCountConfig.IsValid() {
		return errors.New("invalid cyclical count config")
	}

	if !i.Engineering.Weight.IsValid() {
		return errors.New("invalid weight")
	}

	if i.Planning.ReorderPoint != nil && !i.Planning.ReorderPoint.IsValid() {
		return errors.New("invalid reorder point")
	}

	return nil
}

func (i *Item) ValidateForCreation() error {
	code := i.BusinessCode
	i.BusinessCode = "TEMPORARIO"
	err := i.Validate()
	i.BusinessCode = code
	return err
}
