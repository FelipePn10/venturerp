package entity

import (
	"errors"

	"github.com/google/uuid"
)

// Weight rules per spec: customer=32, item=16, classification=8, division=4, none=2.
// Both customer+item accumulate: 32+16=48.
func NewRestriction(
	situation RestrictionSituation,
	customerCode, itemCode, reasonCode *int64,
	classificationType, classificationOrigin *string,
	divisionID *int64,
	createdBy uuid.UUID,
) (*Restriction, error) {
	if situation != RestrictionActive && situation != RestrictionInactive {
		return nil, errors.New("invalid restriction situation")
	}

	return &Restriction{
		Situation:            situation,
		CustomerCode:         customerCode,
		ItemCode:             itemCode,
		ReasonCode:           reasonCode,
		ClassificationType:   classificationType,
		ClassificationOrigin: classificationOrigin,
		DivisionID:           divisionID,
		Weight:               CalcWeight(customerCode, itemCode, classificationType, divisionID),
		CreatedBy:            createdBy,
	}, nil
}

func CalcWeight(customerCode, itemCode *int64, classificationType *string, divisionID *int64) int {
	w := 0
	if customerCode != nil {
		w += 32
	}
	if itemCode != nil {
		w += 16
	}
	if w > 0 {
		return w
	}
	if classificationType != nil {
		return 8
	}
	if divisionID != nil {
		return 4
	}
	return 2
}
