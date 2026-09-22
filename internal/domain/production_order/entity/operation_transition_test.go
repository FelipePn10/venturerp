package entity

import "testing"

func TestOperationTransition(t *testing.T) {
	for _, tc := range []struct {
		from, to, reason string
		valid            bool
	}{
		{"PENDING", "IN_PROGRESS", "", true}, {"IN_PROGRESS", "PAUSED", "intervalo", true},
		{"PAUSED", "IN_PROGRESS", "", true}, {"IN_PROGRESS", "INTERRUPTED", "quebra", true},
		{"INTERRUPTED", "IN_PROGRESS", "", true}, {"IN_PROGRESS", "DONE", "", true},
		{"PENDING", "DONE", "", false}, {"DONE", "IN_PROGRESS", "", false},
		{"PAUSED", "DONE", "", false}, {"IN_PROGRESS", "PAUSED", "", false},
		{"PENDING", "SKIPPED", "não se aplica", true}, {"PENDING", "SKIPPED", "", false},
		{"DONE", "DONE", "", false}, {"SKIPPED", "IN_PROGRESS", "", false},
	} {
		t.Run(tc.from+"_"+tc.to+"_"+tc.reason, func(t *testing.T) {
			err := ValidateOperationTransition(tc.from, tc.to, tc.reason)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}
