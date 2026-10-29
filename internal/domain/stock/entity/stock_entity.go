package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Movement type values stored in stock_movements.movement_type.
// These are the canonical strings expected by the stock_balances update logic
// and by the financial reports that filter on movement_type.
const (
	MovementTypeIn                     = "IN"
	MovementTypeOut                    = "OUT"
	MovementTypeTransferIn             = "TRANSFER_IN"
	MovementTypeTransferOut            = "TRANSFER_OUT"
	MovementTypeAdjustment             = "ADJUSTMENT"
	MovementTypeProductionEntry        = "EP"
	MovementTypePlannedProductionEntry = "EPP"
	MovementTypeProductionExcess       = "EPE"
	MovementTypePlannedRequisition     = "REP"
	// Transferência entre endereços do MESMO almoxarifado. O saldo total do
	// almoxarifado não muda — muda de endereço — então SignedQuantity devolve 0
	// de propósito: quem mexe nos dois lados é TransferirEntreEnderecos.
	MovementTypeAddressTransfer = "TRANSF_ENDERECO"
	// MovementTypeCostAdjustment complementa (ou estorna) o custo do saldo sem
	// mexer na quantidade — o frete de compra lançado depois da nota. Não é um
	// tipo de digitação manual: só o sistema o gera.
	MovementTypeCostAdjustment = "AJUSTE_CUSTO"
)

// Reference type values stored in stock_movements.reference_type, identifying the
// document that originated the movement.
const (
	ReferenceTypePurchaseOrder   = "PURCHASE_ORDER"
	ReferenceTypeProductionOrder = "PRODUCTION_ORDER"
	ReferenceTypeSalesOrder      = "SALES_ORDER"
	ReferenceTypeNFEntry         = "NF_ENTRADA"
	ReferenceTypeNFExit          = "NF_SAIDA"
	ReferenceTypeInventory       = "INVENTORY"
	ReferenceTypeCycleCount      = "CONTAGEM_CICLICA"
	ReferenceTypeTransfer        = "TRANSFERENCIA"
	ReferenceTypeReversal        = "ESTORNO"
	ReferenceTypeRequisition     = "REQUISICAO_MATERIAL"
	ReferenceTypeManual          = "MANUAL"
	// Saída para a finalidade de uma demanda independente (VPLA0102).
	ReferenceTypeIndependentDemand = "DEMANDA_INDEPENDENTE"
)

// SignedQuantity returns the delta that a movement of the given type applies to
// the on-hand balance: positive for inbound, negative for outbound. ADJUSTMENT
// is assumed to carry an already-signed quantity. Types that do not change the
// on-hand quantity (e.g. reservations) return 0.
func SignedQuantity(movementType string, quantity decimal.Decimal) decimal.Decimal {
	switch movementType {
	case MovementTypeIn, MovementTypeTransferIn, MovementTypeProductionEntry, MovementTypePlannedProductionEntry, MovementTypeProductionExcess, "ENTRADA":
		return quantity
	case MovementTypeOut, MovementTypeTransferOut, MovementTypePlannedRequisition, "SAIDA":
		return quantity.Neg()
	case MovementTypeAdjustment:
		return quantity
	default:
		return decimal.Zero
	}
}

// TipoMovimentoValido diz se o tipo é um dos que a apuração de saldo reconhece.
//
// Não é preciosismo de vocabulário: SignedQuantity devolve 0 no default, então
// um tipo desconhecido gera um movimento que **não mexe no saldo**. O registro
// aparece no extrato, o saldo não muda, e a diferença só é descoberta no
// inventário. Validar na entrada é o que impede o estoque de mentir.
func TipoMovimentoValido(tipo string) bool {
	switch tipo {
	case MovementTypeIn, MovementTypeOut, MovementTypeTransferIn, MovementTypeTransferOut,
		MovementTypeAdjustment, MovementTypeProductionEntry, MovementTypePlannedProductionEntry,
		MovementTypeProductionExcess, MovementTypePlannedRequisition, MovementTypeAddressTransfer,
		"ENTRADA", "SAIDA":
		return true
	default:
		return false
	}
}

type StockMovement struct {
	ID          int64
	ItemCode    int64
	Mask        string
	WarehouseID int64
	// Endereço de origem e, numa transferência interna, o de destino. Vazio
	// significa almoxarifado sem endereçamento — é como fica toda a base que
	// existia antes da migração 344.
	Address      *string
	AddressTo    *string
	MovementType string
	// Quantidade, preço e total em decimal: o saldo e o custo médio são somas
	// de milhares de movimentos, e float acumulava erro em cada um.
	Quantity       decimal.Decimal
	UnitPrice      decimal.Decimal
	TotalPrice     decimal.Decimal
	ReferenceType  *string
	ReferenceCode  *int64
	Lot            *string
	SerialNumber   *string
	Batch          *string
	ExpirationDate *time.Time
	Notes          *string
	CreatedAt      time.Time
	CreatedBy      uuid.UUID
	// PermiteSaldoNegativo libera ESTE movimento do bloqueio de saldo negativo
	// (não é gravado). É para o ajuste de inventário: a diferença é calculada
	// contra a foto da abertura, e uma saída lançada depois dela pode levar o
	// saldo abaixo de zero — o que o ajuste precisa registrar, não esconder.
	PermiteSaldoNegativo bool
	// MovementTypeCode: sigla do tipo de movimento do cadastro (VEST0200).
	MovementTypeCode *string
	// CostCenterCode: centro de custo da saída (requisição, consumo interno).
	CostCenterCode *int32
	// Estorno: este movimento estorna outro / foi estornado por outro.
	ReversesMovementID   *int64
	ReversedByMovementID *int64
	// ValorizarPelaMedia: entrada que NÃO atualiza o custo médio (tipo de
	// movimento com "atualiza custo" desligado) entra pelo custo médio atual.
	ValorizarPelaMedia bool
}

type StockReservation struct {
	ID                int64
	ItemCode          int64
	Mask              string
	WarehouseID       int64
	Quantity          decimal.Decimal
	ReferenceType     string
	ReferenceCode     int64
	ReferenceItemCode *int64
	ReservationDate   time.Time
	ExpirationDate    *time.Time
	Status            string
	Notes             *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         uuid.UUID
}

type StockBalance struct {
	ID             int64
	ItemCode       int64
	Mask           string
	WarehouseID    int64
	Quantity       decimal.Decimal
	ReservedQty    decimal.Decimal
	AvailableQty   decimal.Decimal
	MinimumStock   decimal.Decimal
	MaximumStock   decimal.Decimal
	SafetyStock    decimal.Decimal
	AvgCost        decimal.Decimal
	LastCost       decimal.Decimal
	TotalCost      decimal.Decimal
	LastMovementAt *time.Time
	UpdatedAt      time.Time
}

type PhysicalInventory struct {
	ID           int64
	Code         int64
	Description  string
	WarehouseID  int64
	StartDate    time.Time
	EndDate      *time.Time
	Status       string
	TotalItems   int
	CountedItems int
	Notes        *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    uuid.UUID
	// ItemCodes restringe a abertura a esses itens (inventário parcial). Vazio =
	// todo o almoxarifado. Não é gravado.
	ItemCodes       []int64
	ClosedBy        *uuid.UUID
	AdjustmentValue *decimal.Decimal
}

// ResultadoFechamentoInventario é o que o fechamento lançou.
type ResultadoFechamentoInventario struct {
	Ajustes            int
	Sobras             decimal.Decimal // quantidade somada das diferenças positivas
	Faltas             decimal.Decimal // quantidade somada das diferenças negativas (em módulo)
	ValorAjustado      decimal.Decimal // valor líquido dos ajustes (sobra − falta, ao custo médio)
	NaoContadosZerados int
}

type PhysicalInventoryItem struct {
	ID               int64
	InventoryID      int64
	ItemCode         int64
	Mask             string
	WarehouseID      int64
	SystemQty        decimal.Decimal
	CountedQty       *decimal.Decimal
	DifferenceQty    *decimal.Decimal
	UnitCost         *decimal.Decimal
	AdjustmentType   *string
	AdjustmentReason *string
	CountedBy        *uuid.UUID
	CountedAt        *time.Time
	IsAdjusted       bool
	CreatedAt        time.Time
	// CountNumber: quantas vezes o item foi contado (recontagem sobrescreve a
	// anterior, que fica em PreviousCountedQty).
	CountNumber          int
	PreviousCountedQty   *decimal.Decimal
	AdjustmentMovementID *int64
}
