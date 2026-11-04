package entity

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

func TestNewTool_ValidationAndDefaults(t *testing.T) {
	if _, err := NewTool(0, "X", "", "", decimal.NewFromFloat(0), decimal.NewFromFloat(0), uuid.New()); err == nil {
		t.Error("expected error for non-positive code")
	}
	if _, err := NewTool(1, "", "", "", decimal.NewFromFloat(0), decimal.NewFromFloat(0), uuid.New()); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := NewTool(1, "Matriz", "", "XPTO", decimal.NewFromFloat(0), decimal.NewFromFloat(0), uuid.New()); err == nil {
		t.Error("expected error for invalid life_type")
	}
	tl, err := NewTool(1, "Matriz Corte", "", "", decimal.NewFromFloat(0), decimal.NewFromFloat(0), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tl.LifeType != LifePieces || tl.ToolType != "FERRAMENTA" || tl.Status != StatusActive || !tl.IsActive {
		t.Errorf("defaults not applied: %+v", tl)
	}
}

func TestNewToolSerial_ValidationAndDefaults(t *testing.T) {
	if _, err := NewToolSerial(0, "SN-1", "", "", "", uuid.New()); err == nil {
		t.Error("expected error for non-positive tool_id")
	}
	if _, err := NewToolSerial(1, "", "", "", "", uuid.New()); err == nil {
		t.Error("expected error for empty serial_number")
	}
	if _, err := NewToolSerial(1, "SN-1", "XPTO", "", "", uuid.New()); err == nil {
		t.Error("expected error for invalid status")
	}
	s, err := NewToolSerial(1, "SN-1", "", "Almox A", "nova", uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status != SerialActive || !s.IsActive || !s.Available() {
		t.Errorf("defaults not applied: %+v", s)
	}
	// A serial under maintenance is not available to run an operation.
	s.Status = SerialMaintenance
	if s.Available() {
		t.Error("maintenance serial must not be available")
	}
}

func TestTool_LifeHelpers(t *testing.T) {
	// Untracked (limit 0) → remaining -1, never needs replacement.
	untracked := &Tool{LifeLimit: decimal.NewFromFloat(0), LifeUsed: decimal.NewFromFloat(999)}
	if !untracked.RemainingLife().Equal(decimal.NewFromFloat(-1)) || untracked.NeedsReplacement() {
		t.Errorf("untracked tool mishandled: rem=%v needs=%v", untracked.RemainingLife(), untracked.NeedsReplacement())
	}
	// Tracked, half-used.
	tl := &Tool{LifeType: LifeStrokes, LifeLimit: decimal.NewFromFloat(1000), LifeUsed: decimal.NewFromFloat(400)}
	if !tl.RemainingLife().Equal(decimal.NewFromFloat(600)) || tl.NeedsReplacement() {
		t.Errorf("tracked half-used: rem=%v needs=%v", tl.RemainingLife(), tl.NeedsReplacement())
	}
	// At/over limit → needs replacement, remaining clamped to 0.
	tl.LifeUsed = decimal.NewFromFloat(1000)
	if !tl.RemainingLife().IsZero() || !tl.NeedsReplacement() {
		t.Errorf("at limit: rem=%v needs=%v", tl.RemainingLife(), tl.NeedsReplacement())
	}
	tl.LifeUsed = decimal.NewFromFloat(1200)
	if !tl.RemainingLife().IsZero() || !tl.NeedsReplacement() {
		t.Errorf("over limit: rem=%v needs=%v", tl.RemainingLife(), tl.NeedsReplacement())
	}
}
