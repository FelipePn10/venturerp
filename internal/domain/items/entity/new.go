package entity

import (
	"errors"
	"strings"
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
	code string,
	name string,
	complement *string,
	nature ItemNature,
	pdm PDM,
	situation types.TypeSituationItem,
	health types.Health,
	warehouse Warehouse,
	engineering Engineering,
	planning Planning,
	supplies Supplies,
	commercial Commercial,
	accounting Accounting,
	createdBy uuid.UUID,
) (*Item, error) {

	var businessCode valueobject.BusinessCode
	if strings.TrimSpace(code) != "" {
		var err error
		businessCode, err = valueobject.NewBusinessCode(code)
		if err != nil {
			return nil, ErrInvalidCode
		}
	}

	item := &Item{
		BusinessCode: businessCode,
		Name:         name,
		Complement:   complement,
		Nature:       nature,
		PDM:          pdm,
		Warehouse:    warehouse,
		Engineering:  engineering,
		Planning:     planning,
		Supplies:     supplies,
		Commercial:   commercial,
		Accounting:   accounting,
		Situation:    situation,
		Health:       health,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}

	if err := item.ValidateForCreation(); err != nil {
		return nil, err
	}

	return item, nil
}
