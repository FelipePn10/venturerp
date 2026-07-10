package response

import "time"

// ─── Conjuntos ────────────────────────────────────────────────────────────────

type CfgSetResponse struct {
	ID          int64     `json:"id"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	VariableQty int64     `json:"variable_qty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─── Variáveis ────────────────────────────────────────────────────────────────

type CfgVariableResponse struct {
	ID                 int64                         `json:"id"`
	SetID              int64                         `json:"set_id"`
	Code               string                        `json:"code"`
	Description        string                        `json:"description"`
	MaskComposition    string                        `json:"mask_composition"`
	IsActive           bool                          `json:"is_active"`
	IsSpecial          bool                          `json:"is_special"`
	IncludeDescription bool                          `json:"include_description"`
	SpecialData        string                        `json:"special_data,omitempty"`
	Marketing          bool                          `json:"marketing"`
	Languages          []CfgVariableLanguageResponse `json:"languages,omitempty"`
}

type CfgVariableLanguageResponse struct {
	ID          int64  `json:"id"`
	VariableID  int64  `json:"variable_id"`
	Language    string `json:"language"`
	Country     string `json:"country,omitempty"`
	Translation string `json:"translation"`
}

// ─── Características ───────────────────────────────────────────────────────────

type CfgCharacteristicResponse struct {
	ID                  int64                               `json:"id"`
	Code                string                              `json:"code"`
	Description         string                              `json:"description"`
	Type                string                              `json:"type"`
	IsActive            bool                                `json:"is_active"`
	SetID               *int64                              `json:"set_id,omitempty"`
	SetDescription      string                              `json:"set_description,omitempty"`
	DefaultVariableID   *int64                              `json:"default_variable_id,omitempty"`
	DefaultVariableCode string                              `json:"default_variable_code,omitempty"`
	Mask                string                              `json:"mask,omitempty"`
	IsSpecial           bool                                `json:"is_special"`
	AffectsPrice        bool                                `json:"affects_price"`
	ControlsGoals       bool                                `json:"controls_goals"`
	ReceivingType       string                              `json:"receiving_type"`
	FieldSource         string                              `json:"field_source,omitempty"`
	Formula             string                              `json:"formula,omitempty"`
	IsRequired          bool                                `json:"is_required"`
	NumMin              *float64                            `json:"num_min,omitempty"`
	NumMax              *float64                            `json:"num_max,omitempty"`
	NumMultiple         *float64                            `json:"num_multiple,omitempty"`
	OptionTrue          string                              `json:"option_true,omitempty"`
	OptionFalse         string                              `json:"option_false,omitempty"`
	Languages           []CfgCharacteristicLanguageResponse `json:"languages,omitempty"`
	CreatedAt           time.Time                           `json:"created_at"`
}

type CfgCharacteristicLanguageResponse struct {
	ID               int64  `json:"id"`
	CharacteristicID int64  `json:"characteristic_id"`
	Language         string `json:"language"`
	Description      string `json:"description"`
	Mask             string `json:"mask,omitempty"`
}

// ─── Características do Item ───────────────────────────────────────────────────

type CfgItemCharacteristicResponse struct {
	ID                 int64   `json:"id"`
	ItemCode           int64   `json:"item_code"`
	CharacteristicID   int64   `json:"characteristic_id"`
	CharacteristicCode string  `json:"characteristic_code"`
	CharacteristicName string  `json:"characteristic_name"`
	CharacteristicType string  `json:"characteristic_type"`
	CharacteristicMask string  `json:"characteristic_mask,omitempty"`
	Sequence           int     `json:"sequence"`
	DefaultVariableID  *int64  `json:"default_variable_id,omitempty"`
	ParentID           *int64  `json:"parent_id,omitempty"`
	IsSpecial          bool    `json:"is_special"`
	IsDrawing          bool    `json:"is_drawing"`
	IsLoad             bool    `json:"is_load"`
	Formula            string  `json:"formula,omitempty"`
	DefaultAnswers     []int64 `json:"default_answers,omitempty"`
}

// ─── Geração de máscara ───────────────────────────────────────────────────────

type CfgGeneratedMaskResponse struct {
	ItemCode  int64                   `json:"item_code"`
	Mask      string                  `json:"mask"`
	MaskHash  string                  `json:"mask_hash"`
	Persisted bool                    `json:"persisted"`
	MaskID    *int64                  `json:"mask_id,omitempty"`
	Answers   []CfgMaskAnswerResponse `json:"answers"`
}

type CfgMaskAnswerResponse struct {
	Position         int    `json:"position"`
	CharacteristicID int64  `json:"characteristic_id"`
	VariableID       *int64 `json:"variable_id,omitempty"`
	Value            string `json:"value"`
}

// ─── Geração de máscara em lote (produto cartesiano) ──────────────────────────

type CfgGeneratedMasksResponse struct {
	ItemCode          int64                  `json:"item_code"`
	TotalCombinations int                    `json:"total_combinations"`
	ValidCount        int                    `json:"valid_count"`
	Persisted         int                    `json:"persisted"`
	Masks             []CfgGeneratedMaskItem `json:"masks"`
}

type CfgGeneratedMaskItem struct {
	Mask     string                  `json:"mask"`
	MaskHash string                  `json:"mask_hash"`
	Answers  []CfgMaskAnswerResponse `json:"answers"`
}

// ─── Tipos de Descrição + Descrição de Itens Configurados (Fase 4) ────────────

type CfgDescriptionTypeResponse struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	IsActive    bool   `json:"is_active"`
}

type CfgItemDescriptionResponse struct {
	ID                int64                            `json:"id"`
	ItemCode          int64                            `json:"item_code"`
	DescriptionTypeID int64                            `json:"description_type_id"`
	Lines             []CfgItemDescriptionLineResponse `json:"lines"`
}

type CfgItemDescriptionLineResponse struct {
	ID                   int64  `json:"id"`
	ItemCharacteristicID int64  `json:"item_characteristic_id"`
	CharacteristicID     int64  `json:"characteristic_id"`
	CharacteristicCode   string `json:"characteristic_code"`
	CharacteristicName   string `json:"characteristic_name"`
	CharacteristicMask   string `json:"characteristic_mask,omitempty"`
	Sequence             int    `json:"sequence"`
	OrderIndex           int    `json:"order_index"`
	ShowCharacteristic   bool   `json:"show_characteristic"`
	ShowMask             bool   `json:"show_mask"`
	DescType             string `json:"desc_type"`
	Text                 string `json:"text,omitempty"`
	LineBreak            bool   `json:"line_break"`
}

type CfgRenderedDescriptionResponse struct {
	ItemDescriptionID int64    `json:"item_description_id"`
	Text              string   `json:"text"`
	Segments          []string `json:"segments"`
}

// ─── Regras de Variáveis Equivalentes + Regras de Itens Configurados (Fase 5) ─

type CfgEquivalentRuleResponse struct {
	ID                     int64  `json:"id"`
	ParentItemCode         int64  `json:"parent_item_code"`
	ParentUOM              string `json:"parent_uom,omitempty"`
	ChildItemCode          int64  `json:"child_item_code"`
	ChildSeq               *int   `json:"child_seq,omitempty"`
	ParentCharacteristicID int64  `json:"parent_characteristic_id"`
	ParentOperator         string `json:"parent_operator"`
	ParentVariableID       *int64 `json:"parent_variable_id,omitempty"`
	ParentVariableCode     string `json:"parent_variable_code,omitempty"`
	ChildCharacteristicID  int64  `json:"child_characteristic_id"`
	ChildOperator          string `json:"child_operator"`
	ChildVariableID        *int64 `json:"child_variable_id,omitempty"`
	ChildVariableCode      string `json:"child_variable_code,omitempty"`
	Formula                string `json:"formula,omitempty"`
	IsActive               bool   `json:"is_active"`
}

type CfgAppliedEquivalentResponse struct {
	ParentItemCode int64            `json:"parent_item_code"`
	ChildAnswers   []CfgChildAnswer `json:"child_answers"`
}

type CfgChildAnswer struct {
	RuleID           int64  `json:"rule_id"`
	ChildItemCode    int64  `json:"child_item_code"`
	ChildSeq         *int   `json:"child_seq,omitempty"`
	CharacteristicID int64  `json:"characteristic_id"`
	Operator         string `json:"operator"`
	VariableID       *int64 `json:"variable_id,omitempty"`
	VariableCode     string `json:"variable_code,omitempty"`
	Formula          string `json:"formula,omitempty"`
}

type CfgItemRuleResponse struct {
	ID          int64                          `json:"id"`
	ItemCode    int64                          `json:"item_code"`
	TargetTable string                         `json:"target_table"`
	TargetField string                         `json:"target_field"`
	Content     string                         `json:"content,omitempty"`
	Formula     string                         `json:"formula,omitempty"`
	Description string                         `json:"description,omitempty"`
	Situation   string                         `json:"situation"`
	Conditions  []CfgItemRuleConditionResponse `json:"conditions"`
}

type CfgItemRuleConditionResponse struct {
	ID               int64  `json:"id"`
	CharacteristicID int64  `json:"characteristic_id"`
	Operator         string `json:"operator"`
	VariableID       *int64 `json:"variable_id,omitempty"`
	VariableCode     string `json:"variable_code,omitempty"`
}

type CfgEvaluatedRulesResponse struct {
	ItemCode    int64                `json:"item_code"`
	Assignments []CfgFieldAssignment `json:"assignments"`
}

type CfgFieldAssignment struct {
	RuleID      int64  `json:"rule_id"`
	TargetTable string `json:"target_table"`
	TargetField string `json:"target_field"`
	Content     string `json:"content,omitempty"`
	Formula     string `json:"formula,omitempty"`
	Description string `json:"description,omitempty"`
}

type CfgReceivingItemResponse struct {
	ID                 int64  `json:"id"`
	CharacteristicID   int64  `json:"characteristic_id"`
	VariableID         *int64 `json:"variable_id,omitempty"`
	VariableCode       string `json:"variable_code,omitempty"`
	ReceivingType      string `json:"receiving_type"`
	ItemCode           *int64 `json:"item_code,omitempty"`
	ClassificationCode *int64 `json:"classification_code,omitempty"`
}
