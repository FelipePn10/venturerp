package procurement_uc

import (
	"testing"
)

func ptrInt64(v int64) *int64 { return &v }

func TestRatioScore(t *testing.T) {
	cases := []struct {
		name     string
		num, den float64
		want     float64
	}{
		{"no data defaults to 100", 0, 0, 100},
		{"perfect", 100, 100, 100},
		{"half", 5, 10, 50},
		{"clamped to 100 when num>den", 12, 10, 100},
		{"clamped to 0 when negative", -3, 10, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ratioScore(c.num, c.den); got != c.want {
				t.Errorf("ratioScore(%v,%v)=%v want %v", c.num, c.den, got, c.want)
			}
		})
	}
}

func TestOverallIQF(t *testing.T) {
	// quality 40% + delivery 30% + commercial 20% + service 10%
	got := overallIQF(100, 100, 100, 100)
	if got != 100 {
		t.Fatalf("all-100 overall = %v, want 100", got)
	}
	got = overallIQF(50, 100, 100, 100) // 20 + 30 + 20 + 10
	if got != 80 {
		t.Fatalf("quality-50 overall = %v, want 80", got)
	}
}
