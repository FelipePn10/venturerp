package entity

import (
	"errors"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
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
	// DescriptionEN: a descrição da nota de exportação (CFOP 7xxx).
	DescriptionEN *string
	// Descrição longa e observações gerais do item; a descrição técnica do PDM
	// continua sendo a que vai para documento.
	LongDescription *string
	GeneralNotes    *string
	// Revisão vigente do item (engenharia).
	Revision *string
	// PreviousCode é o código do item no sistema anterior, para achar o item pela
	// referência antiga. Não vai ao 0200: o Guia da EFD manda não preencher o
	// COD_ANT_ITEM ali.
	PreviousCode *string
	// Fabricante e o número da peça dele (MPN), para achar o item pela referência
	// que vem no catálogo ou na nota do fornecedor.
	ManufacturerName       *string
	ManufacturerPartNumber *string

	// Nature guarda a natureza principal e continua alimentada para quem já a
	// lê. Os marcadores abaixo são independentes: um item pode ser base e
	// configurado ao mesmo tempo, combinação comum em quem usa o configurador.
	Nature ItemNature
	// IsBase indica que o item serve de modelo para criar outros.
	IsBase bool
	// IsConfigured indica que o item tem variações resolvidas pelo configurador.
	IsConfigured bool
	// IsPrototype marca item em desenvolvimento, de controle manual.
	IsPrototype bool
	// IsTool marca ferramenta, que entra no controle de vida útil.
	IsTool bool
	// IsProcessItem marca item de processo em terceiros.
	IsProcessItem bool
	// IsScrap marca sucata e retalho: subproduto na EFD, fora do MRP.
	IsScrap bool
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
	// Signature identifica a combinação de atributos do padrão, para achar o
	// duplicado que a grafia esconderia. Vazia em item sem padrão descritivo.
	Signature *string
}

// Pastas
type Warehouse struct {
	WarehouseCode                   int
	UnitOfMeasurement               types.TypeUnitOfMeasurementItem // Qual unidade de medida será armazenada para esse item
	AutomaticLow                    bool                            // Faz baixa autom?
	CyclicalCountConfig             *valueobject.CyclicalCountConfig
	MinimumStock                    decimal.Decimal  // Estoque mínimo para alerta de compra, na unidade do estoque
	AverageMonthlyConsumptionManual *decimal.Decimal // Consumo médio mensal inserido manualmente ou se for nil gera um calculo de consumo médio/mês
	// ShelfLifeDays é a validade do item em dias: a entrada de lote sem data
	// assume entrada + validade. MinRemainingLifeDays recusa, no recebimento, o
	// lote que já chega com menos vida útil que isso.
	ShelfLifeDays        *int
	MinRemainingLifeDays *int
	StorageConditions    *string
	// Produto perigoso: número ONU e classe de risco vão para o romaneio.
	Hazardous   bool
	UNNumber    *string
	HazardClass *string
	// AcceptsFractionalQuantity: o item admite quantidade fracionada.
	AcceptsFractionalQuantity bool
	// Segunda unidade do estoque: 1 unidade principal = fator nominal na segunda.
	// O movimento pode informar o valor real, como o peso da chapa na balança.
	SecondaryUOM    *string
	SecondaryFactor *decimal.Decimal
}

type Engineering struct {
	ItemBaseCod *int // Somente se ItemBase (checkbox) for false
	// ItemBaseBusinessCode é o código de negócio do item-base, como a tela o
	// envia; o caso de uso o converte em ItemBaseCod.
	ItemBaseBusinessCode string
	Weight               valueobject.Weight

	Dimensions *valueobject.Dimensions

	Type       types.TypeItem
	TypeStruct types.TypeStructItem
	OEM        bool // componentes ou produtos que são fabricados/montados sob a marca de outra empresa e revendidos pela empresa contratante do sistema
	// Volume em metros cúbicos, para frete e ocupação.
	Volume *decimal.Decimal
	// RequiresDrawing: o item só fica pronto para produzir com desenho vinculado.
	RequiresDrawing bool
	// CuttingGrain e CuttingNoRotation vão para o plano de corte: veio do MDF ou
	// sentido de laminação da chapa, peça que não pode girar.
	CuttingGrain      string
	CuttingNoRotation bool
}

type Planning struct {
	// Para o MRP calcular e gerar ordem de máteria prima, o nivél deve ser LLC 9 e ser ACTIVE
	TypeMRP      types.TypeMRPItem
	LLC          int // niveis 1 para o produto final, 2 há 8 para estruras e conjuntos e 9 sendo para matérias primas
	ReorderPoint *valueobject.ReorderPoint
	TankCode     *int // Setor onde é feito
	Ghost        bool
	ABCClass     *string
	MinimumLot   decimal.Decimal
	MultipleLot  decimal.Decimal
	SafetyStock  decimal.Decimal
	Critical     bool
	Exclusive    bool
	Active       bool

	// Responsáveis pelo item (funcionários da empresa).
	PlannerEmployeeCode *int64
	BuyerEmployeeCode   *int64
	// Prazos do próprio item, em dias úteis: o MRP os usa quando não há prazo de
	// fornecedor ou roteiro.
	PurchaseLeadTimeDays      *int
	ManufacturingLeadTimeDays *int
	Notes                     *string
	// MaterialIssueTiming: quando a OF baixa o material (REGISTRATION_RELEASE ou
	// PRODUCTION). ProductionReportingType: apontamento por ORDER ou OPERATION.
	MaterialIssueTiming     string
	ProductionReportingType string

	// Extras gravados em item_planning_extras e política em
	// politicas_planejamento, na mesma transação do item.
	Extras PlanningExtras
}

// PlanningExtras são os parâmetros de planejamento que moram fora de items.
type PlanningExtras struct {
	MaximumStock   decimal.Decimal
	Coverage       int
	GroupingKey    *string
	SafetyTimeDays *int
	MaximumLot     *decimal.Decimal
}

type Supplies struct {
	TypeOfUse          types.TypeOfUseItem
	PurchaseUOM        *types.TypeUnitOfMeasurementItem
	WarehouseCode      *int64
	ReceivingChecklist bool
	Harvest            bool
	// Notes existia na tela (pasta Suprimentos) sem coluna nem campo aqui: o
	// texto digitado sumia no salvar.
	Notes *string
	// Tolerâncias de recebimento do item, em percentual da quantidade pedida.
	UnderDeliveryTolerancePct *decimal.Decimal
	OverDeliveryTolerancePct  *decimal.Decimal
	// ReceiptRouting: DIRETO não passa por inspeção, PADRAO segue a rota de
	// inspeção cadastrada e INSPECAO exige que exista uma.
	ReceiptRouting     string
	ClassificationCode *string
	// RequiresCertificate: o recebimento exige lote, corrida e certificado.
	RequiresCertificate bool
}

type Commercial struct {
	Description            *string
	SaleType               *string
	VolumeConversionFactor *decimal.Decimal
	SaleMultiple           *decimal.Decimal
	MinimumSaleQuantity    *decimal.Decimal
	EstimatedDeliveryDays  *int
	WarrantyDays           int
	// Código (texto) do almoxarifado — FK para warehouse(code). Não é o id.
	TransferWarehouseCode            *string
	TechnicalAssistanceWarehouseCode *string
	PackagingItemCode                *int64
	// PackagingItemBusinessCode é o código de negócio do item de embalagem, como
	// a tela o envia; o caso de uso o converte em PackagingItemCode.
	PackagingItemBusinessCode     string
	AllowBillingDescriptionChange bool
	IssueLoadingLabels            bool
	AssembleShippingVolumes       bool
	RequiresSpecialPackaging      bool
	WithholdPISCOFINS             bool
	IsPackaging                   bool
	MobileEnabled                 bool
	ExportPackaging               bool
	ClassificationCode            *string
	Notes                         *string
	// ExclusiveCustomerCode: peça de desenho do cliente, que só ele compra.
	ExclusiveCustomerCode *int64
	// VolumesPerUnit: em quantas caixas uma unidade embarca (móvel desmontado).
	VolumesPerUnit *int
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
	// GTIN comercial e tributável: cEAN e cEANTrib da NF-e, COD_BARRA do 0200.
	GTIN    *string
	GTINTax *string
	// EFDItemType é o TIPO_ITEM do 0200. Vazio deixa a EFD deduzir.
	EFDItemType *string
	// TaxUnit é a unidade tributável (uTrib) e TaxUnitFactor quantas dela há em
	// uma unidade comercial: qTrib = qCom × fator.
	TaxUnit       *string
	TaxUnitFactor *decimal.Decimal
	// EXTIPI, escala relevante, CNPJ do fabricante e FCI vão ao item da NF-e.
	ExTIPI           *string
	RelevantScale    *bool
	ManufacturerCNPJ *string
	FCINumber        *string
}

// SyncNature alinha a natureza principal aos marcadores, do mais específico
// para o mais genérico. Existe porque `nature` ainda é lida em consultas e
// relatórios: sem isso um item marcado como configurado continuaria gravado
// como genérico.
func (i *Item) SyncNature() {
	switch {
	case i.IsConfigured:
		i.Nature = ItemConfigured
	case i.IsBase:
		i.Nature = ItemBase
	default:
		i.Nature = ItemGeneric
	}
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
		return domainerr.NewValidation("código inválido")
	}
	i.Name = strings.TrimSpace(i.Name)
	if i.Name == "" {
		return domainerr.NewValidation("Informe o nome do item.")
	}
	if i.Engineering.ItemBaseCod != nil && *i.Engineering.ItemBaseCod == 0 {
		i.Engineering.ItemBaseCod = nil
	}
	if !i.Situation.IsValid() || !i.Health.IsValid() || !i.Warehouse.UnitOfMeasurement.IsValid() ||
		!i.Engineering.Type.IsValid() || !i.Engineering.TypeStruct.IsValid() ||
		!i.Planning.TypeMRP.IsValid() || !i.Supplies.TypeOfUse.IsValid() {
		return domainerr.NewValidation("valor inválido para um campo de lista do item")
	}
	if i.Supplies.PurchaseUOM != nil && !i.Supplies.PurchaseUOM.IsValid() {
		return domainerr.NewValidation("unidade de medida de compra inválida")
	}
	if i.Planning.MinimumLot.IsNegative() || i.Planning.MultipleLot.IsNegative() || i.Planning.SafetyStock.IsNegative() || i.Warehouse.MinimumStock.IsNegative() || i.Commercial.WarrantyDays < 0 {
		return domainerr.NewValidation("as quantidades de planejamento e os dias de garantia não podem ser negativos")
	}
	if i.Commercial.SaleType != nil && *i.Commercial.SaleType != "VENDA" && *i.Commercial.SaleType != "REVENDA" {
		return domainerr.NewValidation("o tipo de venda deve ser venda ou revenda")
	}
	for name, value := range map[string]*decimal.Decimal{
		"commercial.volume_conversion_factor": i.Commercial.VolumeConversionFactor,
		"commercial.sale_multiple":            i.Commercial.SaleMultiple,
	} {
		if value != nil && !value.IsPositive() {
			return errors.New(name + " deve ser maior que zero")
		}
	}
	if i.Commercial.MinimumSaleQuantity != nil && i.Commercial.MinimumSaleQuantity.IsNegative() {
		return domainerr.NewValidation("a quantidade mínima de venda não pode ser negativa")
	}
	if i.Commercial.EstimatedDeliveryDays != nil && *i.Commercial.EstimatedDeliveryDays < 0 {
		return domainerr.NewValidation("o prazo estimado de entrega não pode ser negativo")
	}
	if i.Code.IsValid() && i.Commercial.PackagingItemCode != nil && *i.Commercial.PackagingItemCode == int64(i.Code) {
		return domainerr.NewValidation("o item de embalagem não pode ser o próprio item")
	}
	if i.Accounting.Origin != nil && (*i.Accounting.Origin < 0 || *i.Accounting.Origin > 8) {
		return domainerr.NewValidation("a origem da mercadoria deve estar entre 0 e 8")
	}
	for _, value := range []*string{i.Accounting.SaleIPIType, i.Accounting.PurchaseIPIType} {
		if value != nil && *value != "PERCENTUAL" && *value != "VALOR" {
			return domainerr.NewValidation("o tipo de IPI contábil deve ser percentual ou valor")
		}
	}
	for name, value := range map[string]*decimal.Decimal{"accounting.sale_ipi_rate": i.Accounting.SaleIPIRate, "accounting.purchase_ipi_rate": i.Accounting.PurchaseIPIRate, "accounting.icms_rate": i.Accounting.ICMSRate} {
		if value != nil && value.IsNegative() {
			return errors.New(name + " não pode ser negativo")
		}
	}
	if i.Accounting.SaleUnitOfMeasurement != nil && !i.Accounting.SaleUnitOfMeasurement.IsValid() {
		return domainerr.NewValidation("unidade de medida de venda inválida na pasta Contábil")
	}
	if i.Accounting.PurchaseUnitOfMeasurement != nil && !i.Accounting.PurchaseUnitOfMeasurement.IsValid() {
		return domainerr.NewValidation("unidade de medida de compra inválida na pasta Contábil")
	}
	if i.Accounting.CEST != nil {
		if len(*i.Accounting.CEST) != 7 {
			return domainerr.NewValidation("o CEST deve ter exatamente 7 dígitos")
		}
		for _, c := range *i.Accounting.CEST {
			if c < '0' || c > '9' {
				return domainerr.NewValidation("o CEST deve ter exatamente 7 dígitos")
			}
		}
	}
	for name, field := range map[string]struct {
		value *string
		max   int
	}{
		"commercial.description": {i.Commercial.Description, 255}, "commercial.classification_code": {i.Commercial.ClassificationCode, 40}, "commercial.notes": {i.Commercial.Notes, 1000}, "supplies.notes": {i.Supplies.Notes, 1000},
		"accounting.sale_fiscal_classification_code": {i.Accounting.SaleFiscalClassificationCode, 40}, "accounting.purchase_fiscal_classification_code": {i.Accounting.PurchaseFiscalClassificationCode, 40},
		"accounting.accounting_classification_code": {i.Accounting.AccountingClassificationCode, 80}, "accounting.input_code": {i.Accounting.InputCode, 20}, "accounting.notes": {i.Accounting.Notes, 1000},
	} {
		if field.value != nil && len(*field.value) > field.max {
			return domainerr.NewValidation(fmt.Sprintf("o campo %s passou do tamanho máximo de %d caracteres", name, field.max))
		}
	}
	if err := i.validarCadastroAmpliado(); err != nil {
		return err
	}
	if i.Planning.ABCClass != nil && *i.Planning.ABCClass != "A" && *i.Planning.ABCClass != "B" && *i.Planning.ABCClass != "C" {
		return domainerr.NewValidation("a curva ABC deve ser A, B ou C")
	}

	if i.Engineering.Dimensions != nil && !i.Engineering.Dimensions.IsValid() {
		return domainerr.NewValidation("as dimensões do item devem ser todas maiores que zero")
	}
	if i.Warehouse.CyclicalCountConfig != nil && !i.Warehouse.CyclicalCountConfig.IsValid() {
		return domainerr.NewValidation("configuração de contagem cíclica inválida")
	}

	if !i.Engineering.Weight.IsValid() {
		return domainerr.NewValidation("o peso do item não pode ser negativo")
	}

	if i.Planning.ReorderPoint != nil && !i.Planning.ReorderPoint.IsValid() {
		return domainerr.NewValidation("ponto de reposição inválido")
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
