package entity

import (
	"time"

	"github.com/google/uuid"
)

type AllocationBase struct {
	Code        int32
	Description string
	Period      string
	Observation *string
	Items       []*AllocationBaseItem
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uuid.UUID
}

type AllocationBaseItem struct {
	AllocationBaseCode int32
	CostCenterCode     int32
	Amount             float64
	Percentage         float64
	CreatedAt          time.Time
}
