package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateItemConversionDTO: o item chega pelo código de negócio (texto) — é assim
// que a tela de Conversões por Item (VSUP0110) o conhece.
//
// A conversão pode vir como fator único ("1 de = factor para", contrato
// antigo) ou como fração ("from_quantity de = to_quantity para"). Com a
// fração, "3 UN = 1 PCT" é guardado exato, sem a dízima de 0,333….
type CreateItemConversionDTO struct {
	ItemCode        TextCode         `json:"item_code"`
	Mask            string           `json:"mask,omitempty"`
	FromUOM         string           `json:"from_uom"`
	ToUOM           string           `json:"to_uom"`
	Factor          decimal.Decimal  `json:"factor"` // 1 from_uom = factor × to_uom
	FromQuantity    *decimal.Decimal `json:"from_quantity,omitempty"`
	ToQuantity      *decimal.Decimal `json:"to_quantity,omitempty"`
	RoundingPercent decimal.Decimal  `json:"rounding_percent"`
	ToleranceValue  decimal.Decimal  `json:"tolerance_value"`
	ToleranceType   string           `json:"tolerance_type"`
	// ValidFrom (AAAA-MM-DD): início da vigência; vazio = desde sempre.
	ValidFrom string    `json:"valid_from,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedBy uuid.UUID `json:"-"`
	// Origin é definido pelo sistema (ex.: memorizada a partir da nota fiscal).
	Origin string `json:"-"`
}

// UpdateItemConversionDTO altera uma conversão existente (PUT /{id}).
//
// Trocar o fator exige motivo. Sem valid_from o fator é CORRIGIDO na própria
// versão (cadastro errado); com valid_from abre uma NOVA VIGÊNCIA a partir da
// data e encerra a anterior na véspera (mudou o material, o fornecedor…).
type UpdateItemConversionDTO struct {
	Factor          *decimal.Decimal `json:"factor,omitempty"`
	FromQuantity    *decimal.Decimal `json:"from_quantity,omitempty"`
	ToQuantity      *decimal.Decimal `json:"to_quantity,omitempty"`
	RoundingPercent *decimal.Decimal `json:"rounding_percent,omitempty"`
	ToleranceValue  *decimal.Decimal `json:"tolerance_value,omitempty"`
	ToleranceType   *string          `json:"tolerance_type,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	ValidFrom       string           `json:"valid_from,omitempty"`
	Reason          string           `json:"reason,omitempty"`
}

// LotConversionDTO: fator medido de um lote.
type LotConversionDTO struct {
	ItemCode     TextCode        `json:"item_code"`
	Lot          string          `json:"lot"`
	FromUOM      string          `json:"from_uom"`
	ToUOM        string          `json:"to_uom"`
	FromQuantity decimal.Decimal `json:"from_quantity"`
	ToQuantity   decimal.Decimal `json:"to_quantity"`
	Notes        *string         `json:"notes,omitempty"`
}

// UnitOfMeasureDTO: unidade do catálogo da empresa.
type UnitOfMeasureDTO struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Dimension   string `json:"dimension"`
	FiscalCode  string `json:"fiscal_code,omitempty"`
	Decimals    int    `json:"decimals"`
}

// UnitAliasDTO: sinônimo (texto recebido → unidade do catálogo).
type UnitAliasDTO struct {
	Alias        string `json:"alias"`
	UnitCode     string `json:"unit_code"`
	SupplierCode *int64 `json:"supplier_code,omitempty"`
}

// StandardConversionDTO: conversão padrão da empresa (mesma dimensão).
type StandardConversionDTO struct {
	FromUOM      string          `json:"from_uom"`
	ToUOM        string          `json:"to_uom"`
	FromQuantity decimal.Decimal `json:"from_quantity"`
	ToQuantity   decimal.Decimal `json:"to_quantity"`
}
