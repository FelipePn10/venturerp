package entity

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewEntryOperationType_Validation(t *testing.T) {
	if _, err := NewEntryOperationType(0, "x", "1102", uuid.New()); err == nil {
		t.Error("expected error for code 0")
	}
	if _, err := NewEntryOperationType(1, "", "1102", uuid.New()); err == nil {
		t.Error("expected error for empty description")
	}
	if _, err := NewEntryOperationType(1, "x", "", uuid.New()); err == nil {
		t.Error("expected error for empty nature")
	}
	if _, err := NewEntryOperationType(1, "x", "9999", uuid.New()); err == nil {
		t.Error("expected error for nature not starting with 1/2/3")
	}
	o, err := NewEntryOperationType(1, "Compra dentro do estado", "1102", uuid.New())
	if err != nil || !o.IsActive {
		t.Fatalf("expected valid active operation, err=%v", err)
	}
}

func TestEntryOperationType_ValidateUF(t *testing.T) {
	cases := []struct {
		name      string
		nature    string
		ufInGroup bool
		wantErr   bool
	}{
		{"nature1 in group → ok", "1102", true, false},
		{"nature1 NOT in group → erro", "1102", false, true},
		{"nature2 NOT in group → ok", "2102", false, false},
		{"nature2 in group → erro", "2102", true, true},
		{"nature3 foreign → ok regardless", "3102", false, false},
		{"nature3 foreign → ok even if in group", "3102", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := &EntryOperationType{NatureOperation: tc.nature}
			err := o.ValidateUF("PR", tc.ufInGroup)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateUF(nature=%s, inGroup=%v) err=%v, wantErr=%v", tc.nature, tc.ufInGroup, err, tc.wantErr)
			}
		})
	}
}
