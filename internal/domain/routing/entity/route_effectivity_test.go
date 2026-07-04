package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewManufacturingRoute_ValidityWindow(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	after := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	// valid_to before valid_from → error.
	if _, err := NewManufacturingRoute(1, 10, nil, 1, nil, true, &from, &before, uuid.New()); err == nil {
		t.Error("expected error when valid_to precedes valid_from")
	}
	// Proper window is accepted.
	rt, err := NewManufacturingRoute(1, 10, nil, 1, nil, true, &from, &after, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.ValidFrom == nil || rt.ValidTo == nil {
		t.Error("validity window not stored")
	}
	// Open-ended (both nil) is fine.
	if _, err := NewManufacturingRoute(1, 10, nil, 1, nil, true, nil, nil, uuid.New()); err != nil {
		t.Errorf("open-ended route should be valid: %v", err)
	}
}
