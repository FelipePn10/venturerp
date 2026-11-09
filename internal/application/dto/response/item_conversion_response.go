package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ItemUnitConversionResponse is the API representation of an item UOM conversion.
//
// `factor` (número) fica por compatibilidade com os desktops antigos; os
// campos exatos são from_quantity/to_quantity/factor_exact/inverse_factor.
type ItemUnitConversionResponse struct {
	ID int64 `json:"id"`
	// ItemCode é o código de negócio do item (texto), como a tela o conhece.
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name,omitempty"`
	// LegacyCode é a chave numérica interna, mantida para integrações antigas.
	LegacyCode      int64           `json:"legacy_item_code"`
	Mask            string          `json:"mask,omitempty"`
	FromUOM         string          `json:"from_uom"`
	ToUOM           string          `json:"to_uom"`
	Factor          float64         `json:"factor"`
	FromQuantity    decimal.Decimal `json:"from_quantity"`
	ToQuantity      decimal.Decimal `json:"to_quantity"`
	FactorExact     decimal.Decimal `json:"factor_exact"`
	InverseFactor   decimal.Decimal `json:"inverse_factor"`
	RoundingPercent float64         `json:"rounding_percent"`
	ToleranceValue  float64         `json:"tolerance_value"`
	ToleranceType   string          `json:"tolerance_type"`
	ValidFrom       string          `json:"valid_from"`
	ValidTo         *string         `json:"valid_to,omitempty"`
	// Situation: VIGENTE, FUTURA ou ENCERRADA (na data de hoje).
	Situation string     `json:"situation"`
	Origin    string     `json:"origin"`
	Notes     *string    `json:"notes,omitempty"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	CreatedBy uuid.UUID  `json:"created_by"`
	UpdatedAt time.Time  `json:"updated_at"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`
	// Warnings: avisos da gravação (SPED, estruturas com fator antigo…).
	Warnings []string `json:"warnings,omitempty"`
}

// ConversionStepResponse é um passo do caminho de conversão.
type ConversionStepResponse struct {
	FromUOM  string          `json:"from_uom"`
	ToUOM    string          `json:"to_uom"`
	Factor   decimal.Decimal `json:"factor"`
	Source   string          `json:"source"` // LOTE, ITEM_MASCARA, ITEM, PADRAO
	Inverted bool            `json:"inverted"`
	RecordID int64           `json:"record_id"`
}

// ConversionSimulationResponse: resultado do simulador da VSUP0110.
type ConversionSimulationResponse struct {
	ItemCode     string                   `json:"item_code"`
	FromUOM      string                   `json:"from_uom"`
	ToUOM        string                   `json:"to_uom"`
	Quantity     decimal.Decimal          `json:"quantity"`
	Converted    decimal.Decimal          `json:"converted"`
	Rounded      *decimal.Decimal         `json:"rounded,omitempty"`
	Factor       decimal.Decimal          `json:"factor"`
	Explanation  string                   `json:"explanation"`
	Source       string                   `json:"source"`
	Steps        []ConversionStepResponse `json:"steps"`
	Date         string                   `json:"date"`
	Lot          string                   `json:"lot,omitempty"`
	Mask         string                   `json:"mask,omitempty"`
	RoundingNote string                   `json:"rounding_note,omitempty"`
}

// UnitCoverageRow: um papel de unidade do item e se ele chega à de estoque.
type UnitCoverageRow struct {
	Role        string           `json:"role"` // ESTOQUE, COMPRA, COMPRA_CONTABIL, VENDA, SECUNDARIA, ESTRUTURA, FORNECEDOR, PEDIDO, NOTA
	Unit        string           `json:"unit"`
	Status      string           `json:"status"` // BASE, OK, SEM_CONVERSAO, FORA_DO_CATALOGO
	Factor      *decimal.Decimal `json:"factor,omitempty"`
	Explanation string           `json:"explanation,omitempty"`
	Source      string           `json:"source,omitempty"`
	Occurrences int64            `json:"occurrences,omitempty"`
}

// UnitCoverageResponse: matriz de cobertura do item.
type UnitCoverageResponse struct {
	ItemCode          string            `json:"item_code"`
	ItemName          string            `json:"item_name,omitempty"`
	StockUnit         string            `json:"stock_unit"`
	AcceptsFractional bool              `json:"accepts_fractional"`
	Rows              []UnitCoverageRow `json:"rows"`
	Missing           int               `json:"missing"`
}

// ConversionImpactResponse: o que depende da conversão (antes de alterar/excluir).
type ConversionImpactResponse struct {
	ConversionID           int64    `json:"conversion_id"`
	StructureLines         int64    `json:"structure_lines"`
	StructureLinesStale    int64    `json:"structure_lines_stale"`
	OpenPurchaseLines      int64    `json:"open_purchase_lines"`
	PendingEntryLines      int64    `json:"pending_entry_lines"`
	SupplierLinks          int64    `json:"supplier_links"`
	ApprovedEntriesInMonth int64    `json:"approved_entries_in_month"`
	Messages               []string `json:"messages"`
}

// ConversionEventResponse: linha do histórico.
type ConversionEventResponse struct {
	ID              int64          `json:"id"`
	ConversionID    *int64         `json:"conversion_id,omitempty"`
	LotConversionID *int64         `json:"lot_conversion_id,omitempty"`
	Action          string         `json:"action"`
	Before          map[string]any `json:"before,omitempty"`
	After           map[string]any `json:"after,omitempty"`
	Reason          *string        `json:"reason,omitempty"`
	CreatedBy       uuid.UUID      `json:"created_by"`
	CreatedByName   string         `json:"created_by_name,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

// LotConversionResponse: fator medido de um lote.
type LotConversionResponse struct {
	ID           int64           `json:"id"`
	ItemCode     string          `json:"item_code"`
	Lot          string          `json:"lot"`
	FromUOM      string          `json:"from_uom"`
	ToUOM        string          `json:"to_uom"`
	FromQuantity decimal.Decimal `json:"from_quantity"`
	ToQuantity   decimal.Decimal `json:"to_quantity"`
	Factor       decimal.Decimal `json:"factor"`
	Notes        *string         `json:"notes,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Warnings     []string        `json:"warnings,omitempty"`
}

// UnitOfMeasureResponse: unidade do catálogo.
type UnitOfMeasureResponse struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Dimension   string    `json:"dimension"`
	FiscalCode  string    `json:"fiscal_code"`
	Decimals    int       `json:"decimals"`
	IsSystem    bool      `json:"is_system"`
	IsActive    bool      `json:"is_active"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UnitAliasResponse: sinônimo.
type UnitAliasResponse struct {
	ID           int64     `json:"id"`
	Alias        string    `json:"alias"`
	UnitCode     string    `json:"unit_code"`
	SupplierCode *int64    `json:"supplier_code,omitempty"`
	SupplierName string    `json:"supplier_name,omitempty"`
	IsSystem     bool      `json:"is_system"`
	CreatedAt    time.Time `json:"created_at"`
}

// StandardConversionResponse: conversão padrão da empresa.
type StandardConversionResponse struct {
	ID            int64           `json:"id"`
	FromUOM       string          `json:"from_uom"`
	ToUOM         string          `json:"to_uom"`
	FromQuantity  decimal.Decimal `json:"from_quantity"`
	ToQuantity    decimal.Decimal `json:"to_quantity"`
	Factor        decimal.Decimal `json:"factor"`
	InverseFactor decimal.Decimal `json:"inverse_factor"`
	Dimension     string          `json:"dimension"`
	IsSystem      bool            `json:"is_system"`
}

// UnitResolutionResponse: texto recebido traduzido para o catálogo.
type UnitResolutionResponse struct {
	Text string `json:"text"`
	Unit string `json:"unit,omitempty"`
	Via  string `json:"via,omitempty"`
}
