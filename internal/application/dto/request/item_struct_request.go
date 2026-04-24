package request

import (
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

// CreateStructureComponentDTO representa a entrada para criar um componente
// de estrutura (BOM).
//
// Regras:
//   - ParentMask nil  → componente genérico (aplica-se a todas as configurações)
//   - ParentMask != nil → componente específico para aquela configuração
type CreateStructureComponentDTO struct {
	ParentCode        int64                           `json:"parent_code"`
	ChildCode         int64                           `json:"child_code"`
	ParentMask        *string                         `json:"parent_mask,omitempty"`
	Quantity          float64                         `json:"quantity"`
	UnitOfMeasurement types.TypeUnitOfMeasurementItem `json:"unit_of_measurement"`
	Health            types.Health                    `json:"health"`
	LossPercentage    float64                         `json:"loss_percentage"`
	Sequence          int                             `json:"sequence"`
	Notes             *string                         `json:"notes,omitempty"`
	IsActive          bool                            `json:"is_active"`
	CreatedBy         uuid.UUID                       `json:"created_by"`
}

type UpdateStructureComponentDTO struct {
	ParentCode int64   `json:"parent_code"`
	ChildCode  int64   `json:"child_code"`
	ParentMask *string `json:"parent_mask,omitempty"`

	Quantity          float64                         `json:"quantity"`
	UnitOfMeasurement types.TypeUnitOfMeasurementItem `json:"unit_of_measurement"`
	Health            types.Health                    `json:"health"`
	LossPercentage    float64                         `json:"loss_percentage"`
	Position          int                             `json:"position"`
	Notes             *string                         `json:"notes,omitempty"`
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
