package entity

import (
	"errors"

	"github.com/google/uuid"
)

func NewRestriction(situation RestrictionSituation, itemCode, reasonCode *int64, classificationType, classificationOrigin *string, divisionID *int64, createdBy uuid.UUID) (*Restriction, error) {
	if situation != RestrictionActive && situation != RestrictionInactive {
		return nil, errors.New("invalid restriction situation")
	}

	// Weight: item=16, classification=8, division=4, none=2
	weight := 2
	if itemCode != nil {
		weight = 16
	} else if classificationType != nil {
		weight = 8
	} else if divisionID != nil {
		weight = 4
	}

	return &Restriction{
		Situation:            situation,
		ItemCode:             itemCode,
		ReasonCode:           reasonCode,
		ClassificationType:   classificationType,
		ClassificationOrigin: classificationOrigin,
		DivisionID:           divisionID,
		Weight:               weight,
		CreatedBy:            createdBy,
	}, nil
}
