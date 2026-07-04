package entity

import "testing"

func TestSelectPrimarySubstituteComponents(t *testing.T) {
	children := []*ItemStructure{
		{ChildCode: 20, Quantity: 2, SubstituteGroup: 1, SubstitutePriority: 2},
		{ChildCode: 10, Quantity: 3, SubstituteGroup: 1, SubstitutePriority: 1},
		{ChildCode: 30, Quantity: 5},
	}

	selected := SelectPrimarySubstituteComponents(children)

	got := map[int64]float64{}
	for _, child := range selected {
		got[child.ChildCode] = child.Quantity
	}
	if _, ok := got[20]; ok {
		t.Error("substituto secundário não deve permanecer na seleção primária")
	}
	if got[10] != 3 {
		t.Errorf("primário = %v, want 3", got[10])
	}
	if got[30] != 5 {
		t.Errorf("standalone = %v, want 5", got[30])
	}
}

func TestSelectPrimarySubstituteComponents_TieBreaksBySequenceThenCode(t *testing.T) {
	children := []*ItemStructure{
		{ChildCode: 30, SubstituteGroup: 1, SubstitutePriority: 1, Sequence: 20},
		{ChildCode: 20, SubstituteGroup: 1, SubstitutePriority: 1, Sequence: 10},
		{ChildCode: 10, SubstituteGroup: 1, SubstitutePriority: 1, Sequence: 10},
	}

	selected := SelectPrimarySubstituteComponents(children)
	if len(selected) != 1 {
		t.Fatalf("selected = %d, want 1", len(selected))
	}
	if selected[0].ChildCode != 10 {
		t.Errorf("selected child = %d, want 10", selected[0].ChildCode)
	}
}
