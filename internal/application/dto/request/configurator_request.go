package request

import "github.com/google/uuid"

// ─── Conjuntos ────────────────────────────────────────────────────────────────

type CreateCfgSetDTO struct {
	Description string    `json:"description"`
	CreatedBy   uuid.UUID `json:"-"`
}

type UpdateCfgSetDTO struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// ─── Variáveis ────────────────────────────────────────────────────────────────

type CreateCfgVariableDTO struct {
	SetID              int64     `json:"set_id"`
	Code               string    `json:"code"`
	Description        string    `json:"description"`
	MaskComposition    string    `json:"mask_composition"`
	IsSpecial          bool      `json:"is_special"`
	IncludeDescription bool      `json:"include_description"`
	SpecialData        string    `json:"special_data"`
	Marketing          bool      `json:"marketing"`
	CreatedBy          uuid.UUID `json:"-"`
}

type UpdateCfgVariableDTO struct {
	ID                 int64  `json:"id"`
	Code               string `json:"code"`
	Description        string `json:"description"`
	MaskComposition    string `json:"mask_composition"`
	IsActive           bool   `json:"is_active"`
	IsSpecial          bool   `json:"is_special"`
	IncludeDescription bool   `json:"include_description"`
	SpecialData        string `json:"special_data"`
	Marketing          bool   `json:"marketing"`
}

type CfgVariableLanguageDTO struct {
	Language    string `json:"language"`
	Country     string `json:"country"`
	Translation string `json:"translation"`
}

// ─── Características ───────────────────────────────────────────────────────────

type CreateCfgCharacteristicDTO struct {
	Code              string    `json:"code"`
	Description       string    `json:"description"`
	Type              string    `json:"type"`
	SetID             *int64    `json:"set_id"`
	DefaultVariableID *int64    `json:"default_variable_id"`
	Mask              string    `json:"mask"`
	IsSpecial         bool      `json:"is_special"`
	AffectsPrice      bool      `json:"affects_price"`
	ControlsGoals     bool      `json:"controls_goals"`
	ReceivingType     string    `json:"receiving_type"`
	FieldSource       string    `json:"field_source"`
	Formula           string    `json:"formula"`
	IsRequired        bool      `json:"is_required"`
	NumMin            *float64  `json:"num_min"`
	NumMax            *float64  `json:"num_max"`
	NumMultiple       *float64  `json:"num_multiple"`
	OptionTrue        string    `json:"option_true"`
	OptionFalse       string    `json:"option_false"`
	CreatedBy         uuid.UUID `json:"-"`
}

type UpdateCfgCharacteristicDTO struct {
	ID                int64    `json:"id"`
	Code              string   `json:"code"`
	Description       string   `json:"description"`
	Type              string   `json:"type"`
	IsActive          bool     `json:"is_active"`
	SetID             *int64   `json:"set_id"`
	DefaultVariableID *int64   `json:"default_variable_id"`
	Mask              string   `json:"mask"`
	IsSpecial         bool     `json:"is_special"`
	AffectsPrice      bool     `json:"affects_price"`
	ControlsGoals     bool     `json:"controls_goals"`
	ReceivingType     string   `json:"receiving_type"`
	FieldSource       string   `json:"field_source"`
	Formula           string   `json:"formula"`
	IsRequired        bool     `json:"is_required"`
	NumMin            *float64 `json:"num_min"`
	NumMax            *float64 `json:"num_max"`
	NumMultiple       *float64 `json:"num_multiple"`
	OptionTrue        string   `json:"option_true"`
	OptionFalse       string   `json:"option_false"`
}

type CfgCharacteristicLanguageDTO struct {
	Language    string `json:"language"`
	Description string `json:"description"`
	Mask        string `json:"mask"`
}

// ─── Características do Item ───────────────────────────────────────────────────

type AddCfgItemCharacteristicDTO struct {
	ItemCode          int64   `json:"item_code"`
	CharacteristicID  int64   `json:"characteristic_id"`
	Sequence          int     `json:"sequence"`
	DefaultVariableID *int64  `json:"default_variable_id"`
	ParentID          *int64  `json:"parent_id"`
	IsSpecial         bool    `json:"is_special"`
	IsDrawing         bool    `json:"is_drawing"`
	IsLoad            bool    `json:"is_load"`
	Formula           string  `json:"formula"`
	DefaultAnswers    []int64 `json:"default_answers"`
}

type UpdateCfgItemCharacteristicDTO struct {
	ID                int64   `json:"id"`
	Sequence          int     `json:"sequence"`
	DefaultVariableID *int64  `json:"default_variable_id"`
	ParentID          *int64  `json:"parent_id"`
	IsSpecial         bool    `json:"is_special"`
	IsDrawing         bool    `json:"is_drawing"`
	IsLoad            bool    `json:"is_load"`
	Formula           string  `json:"formula"`
	DefaultAnswers    []int64 `json:"default_answers"`
}

// ─── Geração de máscara ───────────────────────────────────────────────────────

type CfgMaskAnswerInput struct {
	CharacteristicID int64  `json:"characteristic_id"`
	VariableID       *int64 `json:"variable_id"` // for ESCOLHA/ESCOLHA_MULT
	Value            string `json:"value"`       // for free/numeric/option/drawing types
}

type CfgGenerateMaskDTO struct {
	ItemCode  int64                `json:"item_code"`
	Answers   []CfgMaskAnswerInput `json:"answers"`
	Persist   bool                 `json:"persist"`
	CreatedBy uuid.UUID            `json:"-"`
}

// ─── Geração de máscara em lote (produto cartesiano) ──────────────────────────

// CfgMaskRestrictInput fixa uma característica a um subconjunto de variáveis para
// reduzir o volume do produto cartesiano (pelo menos uma é obrigatória).
type CfgMaskRestrictInput struct {
	CharacteristicID int64   `json:"characteristic_id"`
	VariableIDs      []int64 `json:"variable_ids"`
}

type CfgGenerateMasksDTO struct {
	ItemCode     int64                  `json:"item_code"`
	CustomerCode *int64                 `json:"customer_code"`
	DivisionID   *int64                 `json:"division_id"`
	Restrict     []CfgMaskRestrictInput `json:"restrict"`
	Persist      bool                   `json:"persist"`
	CreatedBy    uuid.UUID              `json:"-"`
}

// ─── Tipos de Descrição + Descrição de Itens Configurados (Fase 4) ────────────

type CfgDescriptionTypeDTO struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Kind        string    `json:"kind"` // PROGRAMA | RELATORIO | LOV | GERAL
	IsActive    bool      `json:"is_active"`
	CreatedBy   uuid.UUID `json:"-"`
}

type CreateCfgItemDescriptionDTO struct {
	ItemCode          int64     `json:"item_code"`
	DescriptionTypeID int64     `json:"description_type_id"`
	CreatedBy         uuid.UUID `json:"-"`
}

// CfgItemDescriptionLineDTO is one grid row for bulk update.
type CfgItemDescriptionLineDTO struct {
	ID                 int64  `json:"id"`
	OrderIndex         int    `json:"order_index"`
	ShowCharacteristic bool   `json:"show_characteristic"`
	ShowMask           bool   `json:"show_mask"`
	DescType           string `json:"desc_type"` // DESCRICAO | COMP_MASCARA
	Text               string `json:"text"`
	LineBreak          bool   `json:"line_break"`
}

type UpdateCfgItemDescriptionLinesDTO struct {
	Lines []CfgItemDescriptionLineDTO `json:"lines"`
}

// CfgRenderDescriptionDTO renders the configured mask description for a set of answers.
type CfgRenderDescriptionDTO struct {
	Answers []CfgMaskAnswerInput `json:"answers"`
}

// ─── Regras de Variáveis Equivalentes + Regras de Itens Configurados (Fase 5) ─

type CfgEquivalentRuleDTO struct {
	ID                     int64     `json:"id"`
	ParentItemCode         int64     `json:"parent_item_code"`
	ParentUOM              string    `json:"parent_uom"`
	ChildItemCode          int64     `json:"child_item_code"`
	ChildSeq               *int      `json:"child_seq"`
	ParentCharacteristicID int64     `json:"parent_characteristic_id"`
	ParentOperator         string    `json:"parent_operator"`
	ParentVariableID       *int64    `json:"parent_variable_id"`
	ChildCharacteristicID  int64     `json:"child_characteristic_id"`
	ChildOperator          string    `json:"child_operator"`
	ChildVariableID        *int64    `json:"child_variable_id"`
	Formula                string    `json:"formula"`
	CreatedBy              uuid.UUID `json:"-"`
}

type CfgApplyEquivalentDTO struct {
	ParentItemCode int64                `json:"parent_item_code"`
	Answers        []CfgMaskAnswerInput `json:"answers"`
}

type CfgItemRuleConditionDTO struct {
	CharacteristicID int64  `json:"characteristic_id"`
	Operator         string `json:"operator"`
	VariableID       *int64 `json:"variable_id"`
}

type CfgItemRuleDTO struct {
	ID          int64                     `json:"id"`
	ItemCode    int64                     `json:"item_code"`
	TargetTable string                    `json:"target_table"`
	TargetField string                    `json:"target_field"`
	Content     string                    `json:"content"`
	Formula     string                    `json:"formula"`
	Description string                    `json:"description"`
	Situation   string                    `json:"situation"`
	Conditions  []CfgItemRuleConditionDTO `json:"conditions"`
	CreatedBy   uuid.UUID                 `json:"-"`
}

type CfgEvaluateItemRulesDTO struct {
	ItemCode int64                `json:"item_code"`
	Answers  []CfgMaskAnswerInput `json:"answers"`
}

// CfgReceivingItemDTO — Botão Itens do Tipo Recebimento (encarroçadora).
type CfgReceivingItemDTO struct {
	VariableID         *int64 `json:"variable_id"`    // resposta (nulo = toda a característica)
	ReceivingType      string `json:"receiving_type"` // RECEBIMENTO | VINCULO
	ItemCode           *int64 `json:"item_code"`
	ClassificationCode *int64 `json:"classification_code"`
}
