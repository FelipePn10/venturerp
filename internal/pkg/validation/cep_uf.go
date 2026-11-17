package validation

// faixasCEP são as faixas de CEP por estado dos Correios (5 primeiros dígitos).
// A ordem importa onde um estado está no meio do outro (DF dentro de GO, RR e
// AC junto do AM).
var faixasCEP = []struct {
	de, ate int
	uf      string
}{
	{1000, 19999, "SP"}, {20000, 28999, "RJ"}, {29000, 29999, "ES"}, {30000, 39999, "MG"},
	{40000, 48999, "BA"}, {49000, 49999, "SE"}, {50000, 56999, "PE"}, {57000, 57999, "AL"},
	{58000, 58999, "PB"}, {59000, 59999, "RN"}, {60000, 63999, "CE"}, {64000, 64999, "PI"},
	{65000, 65999, "MA"}, {66000, 68899, "PA"}, {68900, 68999, "AP"}, {69000, 69299, "AM"},
	{69300, 69399, "RR"}, {69400, 69899, "AM"}, {69900, 69999, "AC"}, {70000, 72799, "DF"},
	{72800, 72999, "GO"}, {73000, 73699, "DF"}, {73700, 76799, "GO"}, {76800, 76999, "RO"},
	{77000, 77999, "TO"}, {78000, 78899, "MT"}, {79000, 79999, "MS"}, {80000, 87999, "PR"},
	{88000, 89999, "SC"}, {90000, 99999, "RS"},
}

// UFDoCEP devolve o estado de um CEP de 8 dígitos (só dígitos), ou "" quando o
// CEP não cai em nenhuma faixa.
func UFDoCEP(cep string) string {
	s := stripNonDigits(cep)
	if len(s) != 8 {
		return ""
	}
	n := 0
	for _, r := range s[:5] {
		n = n*10 + int(r-'0')
	}
	for _, f := range faixasCEP {
		if n >= f.de && n <= f.ate {
			return f.uf
		}
	}
	return ""
}
