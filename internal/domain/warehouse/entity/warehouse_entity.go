package entity

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

type Warehouse struct {
	ID          int32
	Code        string
	Description string

	Location types.TypeLocation
	Type     types.TypeWarehouse

	Disposition         bool
	ReservationsAllowed bool

	CreatedBy uuid.UUID
	CreatedAt time.Time
}
