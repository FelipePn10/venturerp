package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// BOM statuses (engineering approval lifecycle).
const (
	StatusDraft    = "DRAFT"
	StatusApproved = "APPROVED"
	StatusObsolete = "OBSOLETE"
)

// BomHeader is the versioning/approval header of a product's structure. The lines
// live in item_structures; this carries version, status and BOM type per item+mask.
type BomHeader struct {
	ID        int64
	ItemCode  int64
	Mask      *string
	BomType   string // EBOM | MBOM (free VARCHAR)
	Version   int32
	Status    string // DRAFT | APPROVED | OBSOLETE
	ValidFrom *time.Time
	IsActive  bool
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func ValidStatus(s string) bool {
	switch s {
	case StatusDraft, StatusApproved, StatusObsolete:
		return true
	default:
		return false
	}
}

func NewBomHeader(itemCode int64, mask *string, bomType string, version int32, validFrom *time.Time, createdBy uuid.UUID) (*BomHeader, error) {
	if itemCode <= 0 {
		return nil, errors.New("item_code must be positive")
	}
	if bomType == "" {
		bomType = "MBOM"
	}
	if version <= 0 {
		version = 1
	}
	return &BomHeader{
		ItemCode:  itemCode,
		Mask:      mask,
		BomType:   bomType,
		Version:   version,
		Status:    StatusDraft,
		ValidFrom: validFrom,
		IsActive:  true,
		CreatedBy: createdBy,
	}, nil
}
