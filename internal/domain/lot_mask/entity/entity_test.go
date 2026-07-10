package entity

import (
	"testing"
	"time"
)

func TestGenerate_MixedParts(t *testing.T) {
	now := time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC)
	parts := []LotMaskPart{
		{ID: 1, Sequence: 2, PartType: PartSeqNumerica, Value: "1", Size: 4, CurrentValue: ""},
		{ID: 2, Sequence: 1, PartType: PartCaracter, Value: "LT", Size: 2},
		{ID: 3, Sequence: 3, PartType: PartData, DateFormat: "YYMM"},
	}
	res, err := Generate(parts, now)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// ordered: CARACTER "LT" + SEQ "0001" + DATA "2607"
	if res.Code != "LT00012607" {
		t.Fatalf("code = %q, want LT00012607", res.Code)
	}
	// numeric sequence emitted its initial value (1) and stores "1"
	if len(res.Updates) != 1 || res.Updates[0].NewCurrent != "1" {
		t.Fatalf("updates = %+v, want NewCurrent=1", res.Updates)
	}
}

func TestGenerate_NumericIncrementAndYearReset(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	last := 2025
	// already generated "0009" last year, zero-on-year-change → restart at initial "1"
	p := LotMaskPart{ID: 1, Sequence: 1, PartType: PartSeqNumerica, Value: "1", Size: 4,
		CurrentValue: "9", ZeroOnYearChange: true, LastYear: &last}
	res, err := Generate([]LotMaskPart{p}, now)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if res.Code != "0001" {
		t.Fatalf("year reset code = %q, want 0001", res.Code)
	}

	// same year → increment 9 → 10
	sameYear := 2026
	p.LastYear = &sameYear
	res, _ = Generate([]LotMaskPart{p}, now)
	if res.Code != "0010" {
		t.Fatalf("increment code = %q, want 0010", res.Code)
	}
}

func TestIncAlpha(t *testing.T) {
	cases := map[string]string{"A": "B", "Z": "AA", "AZ": "BA", "ZZ": "AAA", "": "A"}
	for in, want := range cases {
		if got := incAlpha(in); got != want {
			t.Errorf("incAlpha(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGenerate_AlphaSequence(t *testing.T) {
	now := time.Now()
	// first generation uses the initial value "A"
	p := LotMaskPart{ID: 1, Sequence: 1, PartType: PartSeqCaracter, Value: "A"}
	res, _ := Generate([]LotMaskPart{p}, now)
	if res.Code != "A" || res.Updates[0].NewCurrent != "A" {
		t.Fatalf("first alpha = %q (state %q), want A/A", res.Code, res.Updates[0].NewCurrent)
	}
	// next generation increments A → B
	p.CurrentValue = "A"
	res, _ = Generate([]LotMaskPart{p}, now)
	if res.Code != "B" {
		t.Fatalf("next alpha = %q, want B", res.Code)
	}
}

func TestGenerate_LengthCap(t *testing.T) {
	now := time.Now()
	p := LotMaskPart{ID: 1, Sequence: 1, PartType: PartCaracter, Value: "ABCDEFGHIJKLMNOPQRSTUVWXYZ", Size: 26}
	if _, err := Generate([]LotMaskPart{p}, now); err == nil {
		t.Error("esperado erro ao exceder 20 caracteres")
	}
}
