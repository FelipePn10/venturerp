package entity

import (
	"fmt"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PurchasePriceTable struct {
	ID            int64
	EnterpriseID  int64
	Code          int64
	SupplierCode  *int64
	Description   string
	CurrencyCode  string
	ValidityStart *time.Time
	ValidityEnd   *time.Time
	IsActive      bool
	CreatedAt     time.Time
	CreatedBy     uuid.UUID
	UpdatedAt     time.Time
	Items         []*PurchasePriceTableItem
}

// NewPurchasePriceTable monta a tabela de preço. A empresa e o código vêm do
// servidor (JWT e sequência); o fornecedor é opcional — sem ele a tabela vale
// para qualquer fornecedor e o preço por item define o seu.
func NewPurchasePriceTable(enterpriseID, code int64, supplierCode *int64, description, currency string, createdBy uuid.UUID) (*PurchasePriceTable, error) {
	if enterpriseID <= 0 {
		return nil, errorsuc.NewValidationError("não foi possível identificar a empresa da sessão")
	}
	if code <= 0 {
		return nil, errorsuc.NewValidationError("não foi possível gerar o código da tabela de preço")
	}
	if supplierCode != nil && *supplierCode <= 0 {
		return nil, errorsuc.NewValidationError("informe um fornecedor válido ou deixe a tabela sem fornecedor")
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return nil, errorsuc.NewValidationError("informe a descrição da tabela de preço")
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = "BRL"
	}
	if len(currency) != 3 {
		return nil, errorsuc.NewValidationError("a moeda deve ter 3 caracteres (por exemplo BRL)")
	}
	now := time.Now()
	return &PurchasePriceTable{EnterpriseID: enterpriseID, Code: code, SupplierCode: supplierCode, Description: description, CurrencyCode: currency, IsActive: true, CreatedAt: now, CreatedBy: createdBy, UpdatedAt: now}, nil
}

func (t *PurchasePriceTable) ValidateValidity() error {
	if t.ValidityStart != nil && t.ValidityEnd != nil && t.ValidityStart.After(*t.ValidityEnd) {
		return errorsuc.NewValidationError("a vigência inicial não pode ser posterior à vigência final")
	}
	return nil
}

type PurchasePriceTableItem struct {
	ID                     int64
	TableID                int64
	ItemCode               int64
	SupplierCode           *int64
	UOM                    *string
	Price                  decimal.Decimal
	MinQty                 decimal.Decimal
	UpdateReplacementValue bool
	IsActive               bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
	Adjustments            []*PriceAdjustment
}

func NewPurchasePriceTableItem(tableID, itemCode int64, price decimal.Decimal) (*PurchasePriceTableItem, error) {
	if tableID <= 0 {
		return nil, errorsuc.NewValidationError("informe a tabela de preço")
	}
	if itemCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o item")
	}
	if !price.IsPositive() {
		return nil, errorsuc.NewValidationError("o preço deve ser maior que zero")
	}
	now := time.Now()
	return &PurchasePriceTableItem{TableID: tableID, ItemCode: itemCode, Price: price, MinQty: decimal.Zero, IsActive: true, CreatedAt: now, UpdatedAt: now}, nil
}

type PriceAdjustment struct {
	ID              int64
	PriceItemID     int64
	Sequence        int32
	Kind            string
	CalculationType string
	Value           decimal.Decimal
}

func NewPriceAdjustment(sequence int32, kind, calculationType string, value decimal.Decimal) (*PriceAdjustment, error) {
	kind = strings.ToUpper(strings.TrimSpace(kind))
	calculationType = strings.ToUpper(strings.TrimSpace(calculationType))
	if sequence <= 0 || (kind != "DISCOUNT" && kind != "SURCHARGE") || (calculationType != "PERCENT" && calculationType != "FIXED") || value.IsNegative() {
		return nil, fmt.Errorf("invalid price adjustment")
	}
	return &PriceAdjustment{Sequence: sequence, Kind: kind, CalculationType: calculationType, Value: value}, nil
}

type ItemCandidate struct {
	ItemCode            int64
	InternalDescription string
	SupplierItemCode    *string
	SupplierDescription *string
	UOM                 *string
}

type SourcePrice struct {
	SourceType   string
	SourceID     int64
	DocumentCode int64
	DocumentDate time.Time
	SupplierCode int64
	ItemCode     int64
	UOM          string
	UnitPrice    decimal.Decimal
}
