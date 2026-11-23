package third_party_service

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrNotFound = domainerr.NewNotFound("registro de serviço terceirizado não encontrado")

type Price struct {
	ID, EnterpriseID, ItemCode, SupplierCode, OperationID int64
	Mask, UOM, FreightType, Formula                       string
	ItemDescription, SupplierName, OperationName          string
	ReferenceDate                                         time.Time
	Preferred, IsActive                                   bool
	UnitPrice, FreightValue, TaxPercent                   decimal.Decimal
	ConversionFactor                                      *decimal.Decimal
	CreatedBy                                             uuid.UUID
	CreatedAt, UpdatedAt                                  time.Time
	Rules                                                 []PriceRule
}
type PriceRule struct {
	ID             int64
	Characteristic string
	Answer         *string
}
type PriceFilter struct {
	ItemFrom, ItemTo, SupplierFrom, SupplierTo, OperationID *int64
	Mask                                                    *string
	ReferenceDate                                           *time.Time
	Preferred                                               *bool
	PriceType, OrderBy, ItemSearch, SupplierSearch          string
	ClassificationMaskCode                                  *int64
	ClassificationCodes                                     []string
	Limit, Offset                                           int
}
type History struct {
	ID, PriceID    int64
	Action, Reason string
	Snapshot       []byte
	ChangedBy      uuid.UUID
	ChangedAt      time.Time
}

type ServiceOrder struct {
	ID, Code, EnterpriseID, ProductionOrderID, RouteOperationID, OperationID, ItemCode int64
	PlannedSuggestionCode, PlanCode                                                    *int64
	Mask, UOM, Status, RemittanceType, Notes                                           string
	ItemDescription, SupplierName, OperationName                                       string
	SupplierCode, ServiceItemCode, PurchaseRequisitionCode, PurchaseOrderCode          *int64
	Quantity, FulfilledQuantity                                                        decimal.Decimal
	StartDate, DueDate                                                                 time.Time
	Kanban                                                                             bool
	CreatedBy                                                                          uuid.UUID
	CreatedAt, UpdatedAt                                                               time.Time
}
type OrderFilter struct {
	PlanCode                                                                                            *int64
	ItemFrom, ItemTo, ProductionOrderID, ServiceOrderCode, OperationID, SupplierCode, PurchaseOrderCode *int64
	ProductionOrderIDs, ServiceOrderCodes, OperationIDs, SupplierCodes, PurchaseOrderCodes              []int64
	From, To                                                                                            *time.Time
	EmittedFrom, EmittedTo, DeliveryFrom, DeliveryTo                                                    *time.Time
	Statuses                                                                                            []string
	Position, ItemSearch, SupplierSearch, OrderBy                                                       string
	ClassificationMaskCode                                                                              *int64
	ClassificationCodes                                                                                 []string
	OnlyKanban                                                                                          bool
	Limit, Offset                                                                                       int
}
type Movement struct {
	ID, ServiceOrderID                  int64
	MovementType                        string
	Quantity                            decimal.Decimal
	OccurredAt                          time.Time
	ReferenceType, ReferenceCode, Notes string
	IdempotencyKey                      string
	WarehouseID                         *int64
	Lot                                 string
	CreatedBy                           uuid.UUID
}
type GlobalConversion struct {
	ID, EnterpriseID     int64
	FromUOM, ToUOM       string
	Factor               decimal.Decimal
	IsActive             bool
	CreatedBy            uuid.UUID
	CreatedAt, UpdatedAt time.Time
}
type OrderHistory struct {
	ID, ServiceOrderID           int64
	EventType                    string
	PreviousStatus, NewStatus    *string
	Quantity                     *decimal.Decimal
	ReferenceType, ReferenceCode string
	ActorID                      uuid.UUID
	OccurredAt                   time.Time
}

func (p *Price) Validate() error {
	p.Mask = strings.TrimSpace(p.Mask)
	p.UOM = strings.ToUpper(strings.TrimSpace(p.UOM))
	p.FreightType = strings.ToUpper(strings.TrimSpace(p.FreightType))
	if p.ItemCode <= 0 || p.SupplierCode <= 0 || p.OperationID <= 0 || p.UOM == "" || p.ReferenceDate.IsZero() {
		return domainerr.NewValidation("item, fornecedor, operação, unidade e data de referência são obrigatórios")
	}
	if p.UnitPrice.IsNegative() || p.FreightValue.IsNegative() || p.TaxPercent.IsNegative() || p.TaxPercent.GreaterThan(decimal.NewFromInt(100)) {
		return domainerr.NewValidation("valores de preço, frete ou imposto são inválidos")
	}
	if p.FreightType != "FIXED" && p.FreightType != "PERCENT" {
		return domainerr.NewValidation("frete deve ser FIXO ou PERCENTUAL")
	}
	if p.ConversionFactor != nil && !p.ConversionFactor.IsPositive() {
		return domainerr.NewValidation("fator de conversão deve ser positivo")
	}
	return nil
}
func (o *ServiceOrder) Pending() decimal.Decimal {
	v := o.Quantity.Sub(o.FulfilledQuantity)
	if v.IsNegative() {
		return decimal.Zero
	}
	return v
}

// PrazoLegalDeRetorno: a suspensão do ICMS na remessa para industrialização vale
// por 180 dias (prorrogáveis pela SEFAZ); passado isso, a remessa vira saída tributada.
const PrazoLegalDeRetorno = 180

// EmPoder é o que uma OS de terceiro deixou com o fornecedor.
type EmPoder struct {
	SupplierCode      int64  `json:"supplier_code"`
	SupplierName      string `json:"supplier_name"`
	ServiceOrderID    int64  `json:"service_order_id"`
	ServiceOrderCode  int64  `json:"service_order_code"`
	ProductionOrderID int64  `json:"production_order_id"`
	// ItemCode é a peça que foi ao terceiro (o componente da etapa), não
	// necessariamente o item da OF.
	ItemCode         int64           `json:"item_code"`
	ItemBusinessCode string          `json:"item_business_code"`
	ItemName         string          `json:"item_name"`
	Remitted         decimal.Decimal `json:"remitted"`
	Returned         decimal.Decimal `json:"returned"`
	Received         decimal.Decimal `json:"received"`
	InHands          decimal.Decimal `json:"in_hands"`
	// Valor do que está com o terceiro, pelo custo médio do item.
	Value               decimal.Decimal `json:"value"`
	FirstRemittance     time.Time       `json:"first_remittance"`
	DueDate             time.Time       `json:"due_date"`
	DaysInHands         int             `json:"days_in_hands"`
	Overdue             bool            `json:"overdue"` // passou da data prevista da OS
	LegalDeadline       time.Time       `json:"legal_deadline"`
	LegalDeadlineNear   bool            `json:"legal_deadline_near"` // faltam 30 dias ou menos para o prazo legal
	LegalDeadlinePassed bool            `json:"legal_deadline_passed"`
}

// CorrecaoDeRemessa é uma OS cuja remessa levou o item da OF no lugar da peça
// da etapa (antes da correção do bug Q) e o que o reparo fez com ela.
type CorrecaoDeRemessa struct {
	ServiceOrderCode int64           `json:"service_order_code"`
	ItemLevado       int64           `json:"item_moved"`
	ItemCerto        int64           `json:"item_expected"`
	Quantidade       decimal.Decimal `json:"quantity"`
	// Situacao: CORRIGIDA, AMBIGUA (a OF tem vários materiais e nenhum na
	// etapa: corrija a estrutura e rode de novo) ou SIMULADA (sem gravar).
	Situacao string `json:"status"`
}

// FiltroEmPoder filtra o saldo em poder de terceiros.
type FiltroEmPoder struct {
	SupplierCode *int64
	OnlyOverdue  bool
}
