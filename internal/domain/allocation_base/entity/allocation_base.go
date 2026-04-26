package entity

import (
	"time"

	"github.com/google/uuid"
)

type AllocationBase struct {
	ID          int64
	Code        string
	Description string
	Period      string
	Observation *string
	Items       []*AllocationBaseItem
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uuid.UUID
}

type AllocationBaseItem struct {
	ID               int64
	AllocationBaseID int64
	CostCenterID     int64
	Amount           float64
	Percentage       float64
	CreatedAt        time.Time
}
