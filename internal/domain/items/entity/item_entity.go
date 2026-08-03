package entity

import (
	"errors"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

type Item struct {
	ID         int64
	Code       valueobject.ItemCode
	Name       string
	Complement *string

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
	AccountingFiscal AccountingFiscal
	//Status    types.Status

	CreatedBy uuid.UUID
	CreatedAt time.Time
}

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
	WarrantyDays int
}

type AccountingFiscal struct {
	Active             bool
	CalculatePISCOFINS bool
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
	if !i.Code.IsValid() {
		return errors.New("invalid code")
	}
	if strings.TrimSpace(i.Name) == "" {
		return errors.New("item name is required")
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
	if i.Planning.ABCClass != nil && *i.Planning.ABCClass != "A" && *i.Planning.ABCClass != "B" && *i.Planning.ABCClass != "C" {
		return errors.New("invalid ABC class")
	}

	if i.Engineering.Dimensions != nil && !i.Engineering.Dimensions.IsValid() {
		return errors.New("invalid dimensions")
	}

	if !i.Engineering.Weight.IsValid() {
		return errors.New("invalid weight")
	}

	if i.Planning.ReorderPoint != nil && !i.Planning.ReorderPoint.IsValid() {
		return errors.New("invalid reorder point")
	}

	if i.Nature != ItemBase && i.Engineering.ItemBaseCod == nil {
		return errors.New("item base code required")
	}

	if i.Nature == ItemBase && i.Engineering.ItemBaseCod != nil {
		return errors.New("item base cannot have base code")
	}

	return nil
}
