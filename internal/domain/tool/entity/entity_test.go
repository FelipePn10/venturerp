package entity

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewTool_ValidationAndDefaults(t *testing.T) {
	if _, err := NewTool(0, "X", "", "", 0, 0, uuid.New()); err == nil {
		t.Error("expected error for non-positive code")
	}
	if _, err := NewTool(1, "", "", "", 0, 0, uuid.New()); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := NewTool(1, "Matriz", "", "XPTO", 0, 0, uuid.New()); err == nil {
		t.Error("expected error for invalid life_type")
	}
	tl, err := NewTool(1, "Matriz Corte", "", "", 0, 0, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tl.LifeType != LifePieces || tl.ToolType != "FERRAMENTA" || tl.Status != StatusActive || !tl.IsActive {
		t.Errorf("defaults not applied: %+v", tl)
	}
}

func TestTool_LifeHelpers(t *testing.T) {
	// Untracked (limit 0) → remaining -1, never needs replacement.
	untracked := &Tool{LifeLimit: 0, LifeUsed: 999}
	if untracked.RemainingLife() != -1 || untracked.NeedsReplacement() {
		t.Errorf("untracked tool mishandled: rem=%v needs=%v", untracked.RemainingLife(), untracked.NeedsReplacement())
	}
	// Tracked, half-used.
	tl := &Tool{LifeType: LifeStrokes, LifeLimit: 1000, LifeUsed: 400}
	if tl.RemainingLife() != 600 || tl.NeedsReplacement() {
		t.Errorf("tracked half-used: rem=%v needs=%v", tl.RemainingLife(), tl.NeedsReplacement())
	}
	// At/over limit → needs replacement, remaining clamped to 0.
	tl.LifeUsed = 1000
	if tl.RemainingLife() != 0 || !tl.NeedsReplacement() {
		t.Errorf("at limit: rem=%v needs=%v", tl.RemainingLife(), tl.NeedsReplacement())
	}
	tl.LifeUsed = 1200
	if tl.RemainingLife() != 0 || !tl.NeedsReplacement() {
		t.Errorf("over limit: rem=%v needs=%v", tl.RemainingLife(), tl.NeedsReplacement())
	}
}
