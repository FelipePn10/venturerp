package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreatePurchasePriceTableDTO: a empresa e o código da tabela são resolvidos no
// servidor (JWT + sequência). O fornecedor é opcional — uma tabela sem fornecedor
// vale para qualquer um, e o preço por item pode fixar o seu próprio.
type CreatePurchasePriceTableDTO struct {
	SupplierCode *int64 `json:"supplier_code,omitempty"`
	Description  string `json:"description"`
	CurrencyCode string `json:"currency_code,omitempty"`
	// Currency é o nome usado pela tela (VSUP0120).
	Currency      string  `json:"currency,omitempty"`
	ValidityStart *string `json:"validity_start,omitempty"`
	ValidityEnd   *string `json:"validity_end,omitempty"`
	// ValidFrom/ValidTo são os nomes usados pela tela (VSUP0120).
	ValidFrom *string `json:"valid_from,omitempty"`
	ValidTo   *string `json:"valid_to,omitempty"`
	// CreatedBy vem do JWT; nunca do corpo da requisição.
	CreatedBy uuid.UUID `json:"-"`
}

// Currency resolve o apelido de moeda usado pela tela.
func (d CreatePurchasePriceTableDTO) ResolvedCurrency() string {
	if d.CurrencyCode != "" {
		return d.CurrencyCode
	}
	return d.Currency
}

// ResolvedValidity resolve os apelidos de vigência usados pela tela.
func (d CreatePurchasePriceTableDTO) ResolvedValidity() (*string, *string) {
	start, end := d.ValidityStart, d.ValidityEnd
	if start == nil {
		start = d.ValidFrom
	}
	if end == nil {
		end = d.ValidTo
	}
	return start, end
}

type UpdatePurchasePriceTableDTO struct {
	Code          int64   `json:"code"`
	SupplierCode  *int64  `json:"supplier_code,omitempty"`
	Description   string  `json:"description"`
	CurrencyCode  string  `json:"currency_code,omitempty"`
	Currency      string  `json:"currency,omitempty"`
	ValidityStart *string `json:"validity_start,omitempty"`
	ValidityEnd   *string `json:"validity_end,omitempty"`
	ValidFrom     *string `json:"valid_from,omitempty"`
	ValidTo       *string `json:"valid_to,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

// ResolvedCurrency resolve o apelido de moeda usado pela tela.
func (d UpdatePurchasePriceTableDTO) ResolvedCurrency() string {
	if d.CurrencyCode != "" {
		return d.CurrencyCode
	}
	return d.Currency
}

// ResolvedValidity resolve os apelidos de vigência usados pela tela.
func (d UpdatePurchasePriceTableDTO) ResolvedValidity() (*string, *string) {
	start, end := d.ValidityStart, d.ValidityEnd
	if start == nil {
		start = d.ValidFrom
	}
	if end == nil {
		end = d.ValidTo
	}
	return start, end
}

type PriceAdjustmentDTO struct {
	Sequence        int32           `json:"sequence"`
	Kind            string          `json:"kind"`
	CalculationType string          `json:"calculation_type"`
	Value           decimal.Decimal `json:"value"`
}

type AddPurchasePriceItemDTO struct {
	TableCode              int64                `json:"table_code"`
	ItemCode               TextCode             `json:"item_code"`
	SupplierCode           *int64               `json:"supplier_code,omitempty"`
	UOM                    *string              `json:"uom,omitempty"`
	Price                  decimal.Decimal      `json:"price"`
	MinQty                 decimal.Decimal      `json:"min_qty"`
	UpdateReplacementValue bool                 `json:"update_replacement_value"`
	Adjustments            []PriceAdjustmentDTO `json:"adjustments,omitempty"`
}

type CopyPriceAdjustmentsDTO struct {
	SourceItemID int64  `json:"source_item_id"`
	TargetItemID int64  `json:"target_item_id"`
	Mode         string `json:"mode"`
}

type ApplyPurchasePriceSourcesDTO struct {
	TableCode  int64 `json:"table_code"`
	Overwrite  bool  `json:"overwrite"`
	Selections []struct {
		SourceType string `json:"source_type"`
		SourceID   int64  `json:"source_id"`
	} `json:"selections"`
}
