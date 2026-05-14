package entity

import (
	"time"

	"github.com/google/uuid"
)

type PlanningParam struct {
	ID          int64
	ParamNumber int
	ParamKey    string
	Value       string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	UpdatedBy   uuid.UUID
}
