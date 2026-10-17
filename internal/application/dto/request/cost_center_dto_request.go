package request

import (
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

type CreateCostCenterDTO struct {
	Code        int32        `json:"code"`
	Description string       `json:"description"`
	ParentCode  *int32       `json:"parent_code,omitempty"`
	Type        types.TypeCC `json:"type"`
	IsRatio     bool         `json:"is_ratio"`
	StartDate   string       `json:"start_date"`
	EndDate     *string      `json:"end_date,omitempty"`
	CreatedBy   uuid.UUID    `json:"created_by"`
}

// UpdateCostCenterDTO altera um centro de custo industrial. O código não muda
// (é a chave que máquinas, estruturas e ordens guardam). Descrição, data de fim e
// situação mudam a qualquer momento; tipo, centro-pai, rateio e data de início só
// enquanto nada usa o centro.
type UpdateCostCenterDTO struct {
	Description string  `json:"description"`
	ParentCode  *int32  `json:"parent_code,omitempty"`
	Type        string  `json:"type"`
	IsRatio     *bool   `json:"is_ratio,omitempty"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}
