package ibpt_uc

import "testing"

func TestParseBRDecimalAceitaPontoDoArquivoOficial(t *testing.T) {
	for in, want := range map[string]float64{"13.45": 13.45, "13,45": 13.45, "1.234,5": 1234.5, " 0 ": 0, "19.2300": 19.23} {
		if got := parseBRDecimal(in); got != want {
			t.Errorf("parseBRDecimal(%q) = %v, want %v", in, got, want)
		}
	}
}
