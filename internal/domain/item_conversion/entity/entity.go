package entity

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"

	"github.com/google/uuid"
)

// Origens possíveis de uma conversão do item.
const (
	OrigemManual       = "MANUAL"
	OrigemNotaFiscal   = "NOTA_FISCAL"
	OrigemPedidoCompra = "PEDIDO_COMPRA"
	OrigemImportacao   = "IMPORTACAO"
)

// SemprePassado é o início de vigência de uma conversão "desde sempre".
var SemprePassado = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// ItemUnitConversion: FromQty FromUOM = ToQty ToUOM, para um item.
//
// A conversão é guardada como FRAÇÃO (as duas quantidades), como no SAP:
// "3 UN = 1 PCT" não vira 0,3333333333 — a inversa e a cadeia saem exatas.
// Factor (= ToQty/FromQty com dez casas) continua gravado para quem lê a
// coluna direto.
type ItemUnitConversion struct {
	ID              int64
	ItemCode        int64
	Mask            string
	FromUOM         string
	ToUOM           string
	FromQty         decimal.Decimal
	ToQty           decimal.Decimal
	Factor          decimal.Decimal
	RoundingPercent decimal.Decimal
	ToleranceValue  decimal.Decimal
	ToleranceType   string
	ValidFrom       time.Time
	ValidTo         *time.Time
	Origin          string
	Notes           *string
	IsActive        bool
	CreatedAt       time.Time
	CreatedBy       uuid.UUID
	UpdatedAt       time.Time
	UpdatedBy       *uuid.UUID
}

// NewItemUnitConversion mantém a assinatura antiga (fator único: 1 de = fator para).
func NewItemUnitConversion(itemCode int64, mask, fromUOM, toUOM string, factor, roundingPercent, toleranceValue decimal.Decimal, toleranceType string, createdBy uuid.UUID) (*ItemUnitConversion, error) {
	return NewItemUnitConversionFraction(itemCode, mask, fromUOM, toUOM, decimal.NewFromInt(1), factor, roundingPercent, toleranceValue, toleranceType, createdBy)
}

// NewItemUnitConversionFraction: fromQty fromUOM = toQty toUOM.
func NewItemUnitConversionFraction(itemCode int64, mask, fromUOM, toUOM string, fromQty, toQty, roundingPercent, toleranceValue decimal.Decimal, toleranceType string, createdBy uuid.UUID) (*ItemUnitConversion, error) {
	fromUOM = strings.ToUpper(strings.TrimSpace(fromUOM))
	toUOM = strings.ToUpper(strings.TrimSpace(toUOM))
	if itemCode == 0 {
		return nil, domainerr.NewValidation("informe o item da conversão")
	}
	if fromUOM == "" || toUOM == "" {
		return nil, domainerr.NewValidation("informe a unidade de origem e a de destino")
	}
	if fromUOM == toUOM {
		return nil, domainerr.NewValidation("a unidade de origem e a de destino devem ser diferentes")
	}
	if !fromQty.IsPositive() || !toQty.IsPositive() {
		return nil, domainerr.NewValidation("o fator de conversão deve ser maior que zero")
	}
	c := &ItemUnitConversion{
		ItemCode:  itemCode,
		Mask:      strings.TrimSpace(mask),
		FromUOM:   fromUOM,
		ToUOM:     toUOM,
		ValidFrom: SemprePassado,
		Origin:    OrigemManual,
		IsActive:  true,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}
	if err := c.SetFraction(fromQty, toQty); err != nil {
		return nil, err
	}
	if err := c.SetPolicy(roundingPercent, toleranceValue, toleranceType); err != nil {
		return nil, err
	}
	return c, nil
}

// SetFraction troca a fração e recalcula o fator.
func (c *ItemUnitConversion) SetFraction(fromQty, toQty decimal.Decimal) error {
	if !fromQty.IsPositive() || !toQty.IsPositive() {
		return domainerr.NewValidation("o fator de conversão deve ser maior que zero")
	}
	f := toQty.DivRound(fromQty, 10)
	if !f.IsPositive() {
		return domainerr.NewValidation("o fator resultante é pequeno demais para dez casas decimais: inverta o sentido da conversão")
	}
	c.FromQty, c.ToQty, c.Factor = fromQty, toQty, f
	return nil
}

// SetPolicy valida e aplica a política de arredondamento/tolerância.
func (c *ItemUnitConversion) SetPolicy(roundingPercent, toleranceValue decimal.Decimal, toleranceType string) error {
	toleranceType = strings.ToUpper(strings.TrimSpace(toleranceType))
	if toleranceType == "" {
		toleranceType = "VALUE"
	}
	if roundingPercent.IsNegative() || roundingPercent.GreaterThan(decimal.NewFromInt(100)) || toleranceValue.IsNegative() || (toleranceType != "VALUE" && toleranceType != "PERCENT") {
		return domainerr.NewValidation("política de arredondamento/tolerância inválida: o arredondamento vai de 0 a 100%, a tolerância não pode ser negativa e o tipo deve ser VALUE (valor) ou PERCENT (percentual)")
	}
	c.RoundingPercent, c.ToleranceValue, c.ToleranceType = roundingPercent, toleranceValue, toleranceType
	return nil
}

// SameFraction: mesma razão (2 = 4/2), não necessariamente as mesmas quantidades.
func (c *ItemUnitConversion) SameFraction(fromQty, toQty decimal.Decimal) bool {
	return c.FromQty.Mul(toQty).Equal(c.ToQty.Mul(fromQty))
}

// Situacao da versão numa data.
const (
	SituacaoVigente   = "VIGENTE"
	SituacaoFutura    = "FUTURA"
	SituacaoEncerrada = "ENCERRADA"
)

func (c *ItemUnitConversion) Situacao(hoje time.Time) string {
	d := hoje.Format("2006-01-02")
	if c.ValidFrom.Format("2006-01-02") > d {
		return SituacaoFutura
	}
	if c.ValidTo != nil && c.ValidTo.Format("2006-01-02") < d {
		return SituacaoEncerrada
	}
	return SituacaoVigente
}

// LotConversion: fator medido de um lote (peso real da chapa daquela corrida).
type LotConversion struct {
	ID        int64
	ItemCode  int64
	Lot       string
	FromUOM   string
	ToUOM     string
	FromQty   decimal.Decimal
	ToQty     decimal.Decimal
	Factor    decimal.Decimal
	Notes     *string
	IsActive  bool
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewLotConversion(itemCode int64, lot, fromUOM, toUOM string, fromQty, toQty decimal.Decimal, notes *string, createdBy uuid.UUID) (*LotConversion, error) {
	lot = strings.TrimSpace(lot)
	fromUOM, toUOM = strings.ToUpper(strings.TrimSpace(fromUOM)), strings.ToUpper(strings.TrimSpace(toUOM))
	if itemCode == 0 {
		return nil, domainerr.NewValidation("informe o item da conversão")
	}
	if lot == "" || len(lot) > 50 {
		return nil, domainerr.NewValidation("informe o lote (até 50 caracteres)")
	}
	if fromUOM == "" || toUOM == "" || fromUOM == toUOM {
		return nil, domainerr.NewValidation("informe duas unidades diferentes")
	}
	if !fromQty.IsPositive() || !toQty.IsPositive() {
		return nil, domainerr.NewValidation("as quantidades devem ser maiores que zero")
	}
	f := toQty.DivRound(fromQty, 10)
	if !f.IsPositive() {
		return nil, domainerr.NewValidation("o fator resultante é pequeno demais: inverta o sentido da conversão")
	}
	return &LotConversion{ItemCode: itemCode, Lot: lot, FromUOM: fromUOM, ToUOM: toUOM, FromQty: fromQty, ToQty: toQty,
		Factor: f, Notes: notes, IsActive: true, CreatedBy: createdBy, CreatedAt: time.Now()}, nil
}

// Ações do histórico.
const (
	EventoCriada       = "CRIADA"
	EventoAlterada     = "ALTERADA"
	EventoNovaVigencia = "NOVA_VIGENCIA"
	EventoExcluida     = "EXCLUIDA"
)

// Event é uma linha do histórico de conversões do item.
type Event struct {
	ID              int64
	ItemCode        int64
	ConversionID    *int64
	LotConversionID *int64
	Action          string
	Before          map[string]any
	After           map[string]any
	Reason          *string
	CreatedBy       uuid.UUID
	CreatedByName   string
	CreatedAt       time.Time
}

// Snapshot é a foto da conversão gravada no histórico.
func (c *ItemUnitConversion) Snapshot() map[string]any {
	m := map[string]any{
		"mask": c.Mask, "from_uom": c.FromUOM, "to_uom": c.ToUOM,
		"from_quantity": c.FromQty.String(), "to_quantity": c.ToQty.String(), "factor": c.Factor.String(),
		"rounding_percent": c.RoundingPercent.String(), "tolerance_value": c.ToleranceValue.String(), "tolerance_type": c.ToleranceType,
		"valid_from": c.ValidFrom.Format("2006-01-02"), "origin": c.Origin,
	}
	if c.ValidTo != nil {
		m["valid_to"] = c.ValidTo.Format("2006-01-02")
	}
	if c.Notes != nil {
		m["notes"] = *c.Notes
	}
	return m
}

func (l *LotConversion) Snapshot() map[string]any {
	m := map[string]any{"lot": l.Lot, "from_uom": l.FromUOM, "to_uom": l.ToUOM,
		"from_quantity": l.FromQty.String(), "to_quantity": l.ToQty.String(), "factor": l.Factor.String()}
	if l.Notes != nil {
		m["notes"] = *l.Notes
	}
	return m
}
