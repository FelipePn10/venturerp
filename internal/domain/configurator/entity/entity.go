// Package entity holds the Product Configurator domain: Sets + Variables,
// Characteristics (with types) and Item-Characteristics. It is a rich model,
// parallel to the legacy `questions`, bridged to `item_masks` for mask generation.
package entity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Characteristic types (Tipo).
const (
	TypeEscolha     = "ESCOLHA"      // pick one variable from a set
	TypeEscolhaMult = "ESCOLHA_MULT" // pick several variables from a set
	TypeFormula     = "FORMULA"      // answer computed by a formula
	TypeDesenho     = "DESENHO"      // answer is a drawing code#digit
	TypeInfCaracter = "INF_CARACTER" // free text complement (optionally required)
	TypeInfNumerica = "INF_NUMERICA" // numeric within range/multiple
	TypeOpcao       = "OPCAO"        // yes/no
	TypeCampo       = "CAMPO"        // pulled from a sales-order field / sequential
	TypeSequencial  = "SEQUENCIAL"   // sequential number (lot/serial)
)

// ReceivingType (Tipo Recebimento) — only relevant for encarroçadora companies.
const (
	RecebNenhum       = "NENHUM"
	RecebRecebimento  = "RECEBIMENTO"
	RecebVinculo      = "VINCULO"
	RecebRecebVinculo = "RECEBIMENTO_VINCULO"
)

// Field sources for the CAMPO type.
const (
	FieldItemCode     = "ITEM_CODE"
	FieldCustomerCode = "CUSTOMER_CODE"
	FieldOrderCode    = "ORDER_CODE"
	FieldSequential   = "SEQUENTIAL"
)

func ValidCharType(t string) bool {
	switch t {
	case TypeEscolha, TypeEscolhaMult, TypeFormula, TypeDesenho, TypeInfCaracter,
		TypeInfNumerica, TypeOpcao, TypeCampo, TypeSequencial:
		return true
	}
	return false
}

func ValidReceivingType(t string) bool {
	switch t {
	case RecebNenhum, RecebRecebimento, RecebVinculo, RecebRecebVinculo:
		return true
	}
	return false
}

func validFieldSource(t string) bool {
	switch t {
	case "", FieldItemCode, FieldCustomerCode, FieldOrderCode, FieldSequential:
		return true
	}
	return false
}

// usesSet reports whether the type draws its answers from a set of variables.
func usesSet(t string) bool { return t == TypeEscolha || t == TypeEscolhaMult }

// ─── Conjunto ─────────────────────────────────────────────────────────────────

type Set struct {
	ID          int64
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uuid.UUID
	VariableQty int // denormalized for reads
}

func NewSet(description string, createdBy uuid.UUID) (*Set, error) {
	if strings.TrimSpace(description) == "" {
		return nil, errors.New("descrição do conjunto é obrigatória")
	}
	return &Set{Description: description, IsActive: true, CreatedBy: createdBy}, nil
}

// ─── Variável ─────────────────────────────────────────────────────────────────

type Variable struct {
	ID                 int64
	SetID              int64
	Code               string
	Description        string
	MaskComposition    string
	IsActive           bool
	IsSpecial          bool
	IncludeDescription bool
	SpecialData        string
	Marketing          bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CreatedBy          uuid.UUID
	Languages          []VariableLanguage
}

type VariableLanguage struct {
	ID          int64
	VariableID  int64
	Language    string
	Country     string
	Translation string
}

func NewVariable(setID int64, code, description, maskComposition string, createdBy uuid.UUID) (*Variable, error) {
	if setID <= 0 {
		return nil, errors.New("set_id é obrigatório")
	}
	if strings.TrimSpace(code) == "" {
		return nil, errors.New("código da variável é obrigatório")
	}
	if strings.TrimSpace(description) == "" {
		return nil, errors.New("descrição da variável é obrigatória")
	}
	if strings.TrimSpace(maskComposition) == "" {
		// default the mask composition to the code when omitted
		maskComposition = code
	}
	return &Variable{
		SetID:           setID,
		Code:            code,
		Description:     description,
		MaskComposition: maskComposition,
		IsActive:        true,
	}, nil
}

// ─── Característica ────────────────────────────────────────────────────────────

type Characteristic struct {
	ID                int64
	Code              string
	Description       string
	Type              string
	IsActive          bool
	SetID             *int64
	DefaultVariableID *int64
	Mask              string
	IsSpecial         bool
	AffectsPrice      bool
	ControlsGoals     bool
	ReceivingType     string
	FieldSource       string
	Formula           string
	IsRequired        bool
	NumMin            *float64
	NumMax            *float64
	NumMultiple       *float64
	OptionTrue        string
	OptionFalse       string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         uuid.UUID
	Languages         []CharacteristicLanguage

	// denormalized for reads
	SetDescription     string
	DefaultVariableStr string
}

type CharacteristicLanguage struct {
	ID               int64
	CharacteristicID int64
	Language         string
	Description      string
	Mask             string
}

func NewCharacteristic(code, description, charType string, createdBy uuid.UUID) (*Characteristic, error) {
	if strings.TrimSpace(code) == "" {
		return nil, errors.New("código da característica é obrigatório")
	}
	if strings.TrimSpace(description) == "" {
		return nil, errors.New("descrição da característica é obrigatória")
	}
	if !ValidCharType(charType) {
		return nil, errors.New("tipo de característica inválido")
	}
	return &Characteristic{
		Code:          code,
		Description:   description,
		Type:          charType,
		IsActive:      true,
		ReceivingType: RecebNenhum,
	}, nil
}

// Validate checks the type-specific invariants of a characteristic.
func (c *Characteristic) Validate() error {
	if !ValidCharType(c.Type) {
		return errors.New("tipo de característica inválido")
	}
	if c.ReceivingType == "" {
		c.ReceivingType = RecebNenhum
	}
	if !ValidReceivingType(c.ReceivingType) {
		return errors.New("tipo de recebimento inválido")
	}
	if !validFieldSource(c.FieldSource) {
		return errors.New("campo de origem inválido")
	}
	if usesSet(c.Type) && (c.SetID == nil || *c.SetID <= 0) {
		return errors.New("características do tipo Escolha exigem um conjunto")
	}
	if c.Type == TypeOpcao {
		if strings.TrimSpace(c.OptionTrue) == "" {
			c.OptionTrue = "SIM"
		}
		if strings.TrimSpace(c.OptionFalse) == "" {
			c.OptionFalse = "NAO"
		}
	}
	if c.Type == TypeInfNumerica && c.NumMin != nil && c.NumMax != nil && *c.NumMax < *c.NumMin {
		return errors.New("num_max não pode ser menor que num_min")
	}
	if c.Type == TypeCampo && c.FieldSource == "" {
		return errors.New("características do tipo Campo exigem um campo de origem")
	}
	return nil
}

// ─── Característica do Item ────────────────────────────────────────────────────

type ItemCharacteristic struct {
	ID                int64
	ItemCode          int64
	CharacteristicID  int64
	Sequence          int
	DefaultVariableID *int64
	ParentID          *int64
	IsSpecial         bool
	IsDrawing         bool
	IsLoad            bool
	Formula           string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DefaultAnswers    []int64 // variable ids (ESCOLHA_MULT)

	// denormalized for reads
	CharacteristicCode string
	CharacteristicName string
	CharacteristicType string
	CharacteristicMask string
	ParentName         string
}

func NewItemCharacteristic(itemCode, characteristicID int64, sequence int) (*ItemCharacteristic, error) {
	if itemCode <= 0 {
		return nil, errors.New("item_code é obrigatório")
	}
	if characteristicID <= 0 {
		return nil, errors.New("characteristic_id é obrigatório")
	}
	if sequence <= 0 {
		return nil, errors.New("sequência deve ser positiva")
	}
	return &ItemCharacteristic{
		ItemCode:         itemCode,
		CharacteristicID: characteristicID,
		Sequence:         sequence,
	}, nil
}

// ─── Operadores de regra ──────────────────────────────────────────────────────

// Rule operators shared by restrictions, equivalent rules and item rules.
const (
	OpEqual      = "EQUAL"
	OpDifferent  = "DIFFERENT"
	OpGreater    = "GREATER"
	OpLess       = "LESS"
	OpBelongs    = "BELONGS"
	OpNotBelongs = "NOT_BELONGS"
)

// MatchOperator evaluates `actual <op> expected` on string answers (expected may
// be a comma-separated list for BELONGS/NOT_BELONGS). Case-insensitive.
func MatchOperator(op, actual, expected string) bool {
	a := strings.TrimSpace(actual)
	e := strings.TrimSpace(expected)
	switch op {
	case OpEqual:
		return strings.EqualFold(a, e)
	case OpDifferent:
		return !strings.EqualFold(a, e)
	case OpGreater:
		return a > e
	case OpLess:
		return a < e
	case OpBelongs:
		return inCSV(e, a)
	case OpNotBelongs:
		return !inCSV(e, a)
	}
	return false
}

func inCSV(csv, needle string) bool {
	for _, v := range strings.Split(csv, ",") {
		if strings.EqualFold(strings.TrimSpace(v), needle) {
			return true
		}
	}
	return false
}

// ─── Montagem da máscara ──────────────────────────────────────────────────────

// MaskSegment is one resolved answer contributing to the mask, keyed by the
// item-characteristic sequence (position).
type MaskSegment struct {
	Position int
	Value    string
}

// BuildMask joins the non-empty segment values (ordered by position) with '#'
// and returns the mask and its 8-char sha256 hash — bit-for-bit compatible with
// the legacy item-mask value object so downstream (structure/sales/MRP) is
// unaffected.
func BuildMask(segments []MaskSegment) (string, string) {
	cp := make([]MaskSegment, len(segments))
	copy(cp, segments)
	sort.SliceStable(cp, func(i, j int) bool { return cp[i].Position < cp[j].Position })
	vals := make([]string, 0, len(cp))
	for _, s := range cp {
		if strings.TrimSpace(s.Value) == "" {
			continue
		}
		vals = append(vals, s.Value)
	}
	mask := strings.Join(vals, "#")
	h := sha256.Sum256([]byte(mask))
	return mask, hex.EncodeToString(h[:])[:8]
}
