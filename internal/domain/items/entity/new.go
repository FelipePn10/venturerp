package entity

import (
	"errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

var (
	ErrInvalidCode      = errors.New("invalid code")
	ErrInvalidCreatedBy = errors.New("created_by cannot be empty")
)

func NewItem(
	code valueobject.ItemCode,
	complement *string,
	nature ItemNature,
	inherit bool,
	pdm PDM,
	situation types.TypeSituationItem,
	health types.Health,
	warehouse Warehouse,
	engineering Engineering,
	planning Planning,
	planners Planners,
	supplies Supplies,
	createdBy uuid.UUID,
) (*Item, error) {

	if !code.IsValid() {
		return nil, ErrInvalidCode
	}

	if createdBy == uuid.Nil {
		return nil, ErrInvalidCreatedBy
	}

	item := &Item{
		Code:        code,
		Complement:  complement,
		Nature:      nature,
		Inherit:     inherit,
		PDM:         pdm,
		Warehouse:   warehouse,
		Engineering: engineering,
		Planning:    planning,
		Planners:    planners,
		Supplies:    supplies,
		Situation:   situation,
		Health:      health,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}

	if err := item.Validate(); err != nil {
		return nil, err
	}

	return item, nil
}
