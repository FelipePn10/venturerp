package request

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

// CreateStructureComponentDTO representa a entrada para criar um componente
// de estrutura (BOM).
//
// Regras:
//   - ParentMask nil  → componente genérico (aplica-se a todas as configurações)
//   - ParentMask != nil → componente específico para aquela configuração
//   - LossFormula substitui LossPercentage quando avaliável com os valores da máscara
//   - QuantityFormula calcula a quantidade a partir das variáveis do configurador
//     (ex.: "2*(COMPRIMENTO/1000)+2*(PROFUNDIDADE/1000)"); com ela preenchida,
//     Quantity vira apenas o valor nominal usado quando a fórmula não é avaliável
type CreateStructureComponentDTO struct {
	ParentCode         TextCode                        `json:"parent_code"`
	ChildCode          TextCode                        `json:"child_code"`
	ParentMask         *string                         `json:"parent_mask,omitempty"`
	Quantity           float64                         `json:"quantity"`
	UnitOfMeasurement  types.TypeUnitOfMeasurementItem `json:"unit_of_measurement"`
	Health             types.Health                    `json:"health"`
	LossPercentage     float64                         `json:"loss_percentage"`
	LossFormula        *string                         `json:"loss_formula,omitempty"`
	QuantityFormula    *string                         `json:"quantity_formula,omitempty"`
	QuantityRounding   string                          `json:"quantity_rounding,omitempty"` // NONE | UP | DOWN | NEAREST
	QuantityScale      int16                           `json:"quantity_scale,omitempty"`    // casas decimais (0–6)
	Sequence           int                             `json:"sequence"`
	Notes              *string                         `json:"notes,omitempty"`
	IsActive           bool                            `json:"is_active"`
	Inherit            bool                            `json:"inherit"`
	StartDate          *time.Time                      `json:"start_date,omitempty"`
	EndDate            *time.Time                      `json:"end_date,omitempty"`
	IsCoproduct        bool                            `json:"is_coproduct"`        // saída (co-produto/sucata), não insumo
	IsFixedQty         bool                            `json:"is_fixed_qty"`        // quantidade por OF (lote)
	SubstituteGroup    int16                           `json:"substitute_group"`    // >0 agrupa componentes alternativos
	SubstitutePriority int16                           `json:"substitute_priority"` // menor = preferencial
}

type UpdateStructureComponentDTO struct {
	ParentCode TextCode `json:"parent_code"`
	ChildCode  TextCode `json:"child_code"`
	ParentMask *string  `json:"parent_mask,omitempty"`

	Quantity           float64                         `json:"quantity"`
	UnitOfMeasurement  types.TypeUnitOfMeasurementItem `json:"unit_of_measurement"`
	Health             types.Health                    `json:"health"`
	LossPercentage     float64                         `json:"loss_percentage"`
	LossFormula        *string                         `json:"loss_formula,omitempty"`
	QuantityFormula    *string                         `json:"quantity_formula,omitempty"`
	QuantityRounding   string                          `json:"quantity_rounding,omitempty"`
	QuantityScale      int16                           `json:"quantity_scale,omitempty"`
	Position           int                             `json:"position"`
	Notes              *string                         `json:"notes,omitempty"`
	StartDate          *time.Time                      `json:"start_date,omitempty"`
	EndDate            *time.Time                      `json:"end_date,omitempty"`
	IsCoproduct        bool                            `json:"is_coproduct"`
	IsFixedQty         bool                            `json:"is_fixed_qty"`
	SubstituteGroup    int16                           `json:"substitute_group"`
	SubstitutePriority int16                           `json:"substitute_priority"`
}

// ConsultStructureDTO é a entrada para consulta da estrutura de produtos (VENG0401).
type ConsultStructureDTO struct {
	ItemCode          TextCode   `json:"item_code"`
	Mask              string     `json:"mask,omitempty"`
	EffectivenessDate *time.Time `json:"effectiveness_date,omitempty"`
	Levels            int        `json:"levels"` // 0 = todos os níveis; N > 0 = máximo N níveis
}

// GetStructureTreeDTO representa a entrada para buscar a árvore BOM genérica
// de um item (sem resolução de máscara).
type GetStructureTreeDTO struct {
	RootItemCode TextCode `json:"root_item_code"`
}

// ResolveStructureForMaskDTO representa a entrada para resolver a árvore BOM
// completa de um item para uma configuração específica (máscara).
//
// A máscara é propagada automaticamente do pai para os filhos com base
// nas perguntas compartilhadas.
type ResolveStructureForMaskDTO struct {
	RootItemCode  TextCode `json:"root_item_code"`
	RootMaskValue string   `json:"root_mask_value"` // ex: "100#100#50"
}
