package entity

import "github.com/google/uuid"

type Modifier struct {
	ID          int
	Description string
	CreatedBy   uuid.UUID
	IsActive    bool
}
