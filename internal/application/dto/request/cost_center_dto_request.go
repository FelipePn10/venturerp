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

type UpdateCostCenterDTO struct {
	Code        int32   `json:"code"`
	Description string  `json:"description"`
	ParentCode  *int32  `json:"parent_code,omitempty"`
	Type        string  `json:"type"`
	IsRatio     bool    `json:"is_ratio"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}
