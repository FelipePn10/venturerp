package entity

import (
	"time"

	"github.com/google/uuid"
)

type OrderPriority struct {
	Code          int64
	IntervalStart float64
	IntervalEnd   float64
	Priority      string
	Description   *string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CreatedBy     uuid.UUID
}
