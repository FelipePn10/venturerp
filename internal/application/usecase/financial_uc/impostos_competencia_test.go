package financial_uc

import (
	"slices"
	"testing"
)

func TestImpostosDaCompetencia(t *testing.T) {
	if got := impostosDaCompetencia("12/2026"); slices.Contains(got, "IBS") || slices.Contains(got, "CBS") {
		t.Fatalf("2026 é ano de teste, IBS/CBS não se apuram: %v", got)
	}
	got := impostosDaCompetencia("01/2027")
	if !slices.Contains(got, "IBS") || !slices.Contains(got, "CBS") || !slices.Contains(got, "ICMS") {
		t.Fatalf("2027 apura IBS e CBS além dos atuais: %v", got)
	}
}
