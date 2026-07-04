package request

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

// CreateStructureComponentDTO representa a entrada para criar um componente
// de estrutura (BOM).
//
// Regras:
//   - ParentMask nil  → componente genérico (aplica-se a todas as configurações)
//   - ParentMask != nil → componente específico para aquela configuração
//   - LossFormula substitui LossPercentage quando avaliável com os valores da máscara
type CreateStructureComponentDTO struct {
	ParentCode         int64                           `json:"parent_code"`
	ChildCode          int64                           `json:"child_code"`
	ParentMask         *string                         `json:"parent_mask,omitempty"`
	Quantity           float64                         `json:"quantity"`
	UnitOfMeasurement  types.TypeUnitOfMeasurementItem `json:"unit_of_measurement"`
	Health             types.Health                    `json:"health"`
	LossPercentage     float64                         `json:"loss_percentage"`
	LossFormula        *string                         `json:"loss_formula,omitempty"`
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
	CreatedBy          uuid.UUID                       `json:"created_by"`
}

type UpdateStructureComponentDTO struct {
	ParentCode int64   `json:"parent_code"`
	ChildCode  int64   `json:"child_code"`
	ParentMask *string `json:"parent_mask,omitempty"`

	Quantity           float64                         `json:"quantity"`
	UnitOfMeasurement  types.TypeUnitOfMeasurementItem `json:"unit_of_measurement"`
	Health             types.Health                    `json:"health"`
	LossPercentage     float64                         `json:"loss_percentage"`
	LossFormula        *string                         `json:"loss_formula,omitempty"`
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
	ItemCode          int64      `json:"item_code"`
	Mask              string     `json:"mask,omitempty"`
	EffectivenessDate *time.Time `json:"effectiveness_date,omitempty"`
	Levels            int        `json:"levels"` // 0 = todos os níveis; N > 0 = máximo N níveis
}

// GetStructureTreeDTO representa a entrada para buscar a árvore BOM genérica
// de um item (sem resolução de máscara).
type GetStructureTreeDTO struct {
	RootItemCode int64 `json:"root_item_code"`
}

// ResolveStructureForMaskDTO representa a entrada para resolver a árvore BOM
// completa de um item para uma configuração específica (máscara).
//
// A máscara é propagada automaticamente do pai para os filhos com base
// nas perguntas compartilhadas.
type ResolveStructureForMaskDTO struct {
	RootItemCode  int64  `json:"root_item_code"`
	RootMaskValue string `json:"root_mask_value"` // ex: "100#100#50"
}
