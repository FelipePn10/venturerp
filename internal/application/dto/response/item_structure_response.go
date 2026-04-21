package response

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

// StructureComponentResponse representa a relação pai-filho persistida
type StructureComponentResponse struct {
	ID                int64                           `json:"id"`
	ParentItemCode    int64                           `json:"parent_item_code"`
	ChildItemCode     int64                           `json:"child_item_code"`
	ChildDescription  string                          `json:"child_description"`
	ParentMask        *string                         `json:"parent_mask,omitempty"`
	IsGeneric         bool                            `json:"is_generic"`
	Quantity          float64                         `json:"quantity"`
	EffectiveQuantity float64                         `json:"effective_quantity"`
	UnitOfMeasurement types.TypeUnitOfMeasurementItem `json:"unit_of_measurement"`
	Health            types.Health                    `json:"health"`
	LossPercentage    float64                         `json:"loss_percentage"`
	Position          int                             `json:"position"`
	Notes             *string                         `json:"notes,omitempty"`
	IsActive          bool                            `json:"is_active"`
	CreatedBy         uuid.UUID                       `json:"created_by"`
	CreatedAt         time.Time                       `json:"created_at"`
	UpdatedAt         time.Time                       `json:"updated_at"`
}

// StructureTreeNodeResponse representa o nó resolvido da estrutura
type StructureTreeNodeResponse struct {
	Component StructureComponentResponse `json:"component"`

	// Máscara efetiva deste nó no contexto da resolução
	Mask          *string `json:"mask,omitempty"`
	EffectiveMask *string `json:"effective_mask,omitempty"` // era "Mask"
	RequiresMask  bool    `json:"requires_mask,omitempty"`  // novo
	// Profundidade na árvore (1 = primeiro nível abaixo da raiz)
	Level int `json:"level"`

	Children []*StructureTreeNodeResponse `json:"children"`
}

// StructureTreeResponse representa a árvore completa resolvida
type StructureTreeResponse struct {
	RootItemCode int64   `json:"root_item_code"`
	RootMask     *string `json:"root_mask,omitempty"`

	Components []*StructureTreeNodeResponse `json:"components"`

	TotalLevels int `json:"total_levels"`
	TotalNodes  int `json:"total_nodes"`
}
