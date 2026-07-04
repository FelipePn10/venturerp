package entity

import (
	"errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

func NewItemStructure(
	parentCode, childCode int64,
	parentMask *string,
	quantity float64,
	uom types.TypeUnitOfMeasurementItem,
	health types.Health,
	lossPercentage float64,
	sequence int,
	notes *string,
	isActive bool,
	inherit bool,
	startDate *time.Time,
	endDate *time.Time,
	lossFormula *string,
	createdBy uuid.UUID,
) (*ItemStructure, error) {
	if parentCode <= 0 {
		return nil, errors.New("parent_code deve ser positivo")
	}
	if childCode <= 0 {
		return nil, errors.New("child_item_id deve ser positivo")
	}
	if quantity <= 0 {
		return nil, errors.New("quantity deve ser maior que zero")
	}
	if lossPercentage < 0 || lossPercentage > 100 {
		return nil, errors.New("loss_percentage deve estar entre 0 e 100")
	}
	if sequence < 1 {
		sequence = 10
	}
	if parentMask != nil && *parentMask == "" {
		return nil, errors.New("parent_mask não pode ser uma string vazia; use nil para genérico")
	}
	if lossFormula != nil && *lossFormula == "" {
		lossFormula = nil
	}

	return &ItemStructure{
		ParentCode:        parentCode,
		ChildCode:         childCode,
		ParentMask:        parentMask,
		Quantity:          quantity,
		UnitOfMeasurement: uom,
		Health:            health,
		LossPercentage:    lossPercentage,
		LossFormula:       lossFormula,
		Sequence:          sequence,
		Notes:             notes,
		IsActive:          isActive,
		Inherit:           inherit,
		StartDate:         startDate,
		EndDate:           endDate,
		CreatedBy:         createdBy,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil
}

func (s *ItemStructure) SetSubstitute(group, priority int16) {
	if group < 0 {
		group = 0
	}
	if priority < 1 {
		priority = 1
	}
	s.SubstituteGroup = group
	s.SubstitutePriority = priority
}

// IsGeneric retorna true quando o componente se aplica a TODAS as configurações.
func (s *ItemStructure) IsGeneric() bool {
	return s.ParentMask == nil
}

// EffectiveQuantity retorna a quantidade considerando o percentual de perda.
// Para avaliação de fórmula, use o pacote formula junto com os valores de máscara.
func (s *ItemStructure) EffectiveQuantity() float64 {
	return s.Quantity * (1 + s.LossPercentage/100.0)
}

// Deactivate realiza o soft-delete do componente.
func (s *ItemStructure) Deactivate() {
	s.IsActive = false
	s.UpdatedAt = time.Now()
}

func (s *ItemStructure) Update(
	quantity float64,
	uom types.TypeUnitOfMeasurementItem,
	health types.Health,
	lossPercentage float64,
	sequence int,
	notes *string,
	startDate *time.Time,
	endDate *time.Time,
	lossFormula *string,
) error {
	if quantity <= 0 {
		return errors.New("quantity deve ser maior que zero")
	}
	if lossPercentage < 0 || lossPercentage > 100 {
		return errors.New("loss_percentage deve estar entre 0 e 100")
	}
	if sequence < 1 {
		sequence = 10
	}
	if lossFormula != nil && *lossFormula == "" {
		lossFormula = nil
	}
	s.Quantity = quantity
	s.UnitOfMeasurement = uom
	s.Health = health
	s.LossPercentage = lossPercentage
	s.LossFormula = lossFormula
	s.Sequence = sequence
	s.Notes = notes
	s.StartDate = startDate
	s.EndDate = endDate
	s.UpdatedAt = time.Now()
	return nil
}
