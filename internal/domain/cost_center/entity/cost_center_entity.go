package entity

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

type CostCenter struct {
	ID          int64
	Code        int32
	Description string
	ParentCode  *int32
	Type        types.TypeCC
	IsRatio     bool
	StartDate   time.Time
	EndDate     *time.Time
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uuid.UUID
}
