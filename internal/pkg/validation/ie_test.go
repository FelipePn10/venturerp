package validation

import "testing"

// Exemplos dos roteiros do SINTEGRA e IEs reais (Tecnofer, autorizada pela SEFAZ-PR).
func TestIEValida(t *testing.T) {
	validas := map[string]string{
		"AC": "01.004.823/001-12", "AL": "240000048", "AP": "030123459", "AM": "999999990",
		"BA": "123456-63", "CE": "06000001-5", "DF": "07300001001-09", "ES": "999999990",
		"GO": "10.987.654-7", "MA": "120000385", "MT": "0013000001-9", "MS": "280000006",
		"MG": "062.307.904/0081", "PA": "15-999999-5", "PB": "06000001-5", "PR": "9103144679",
		"PE": "0321418-40", "PI": "012345679", "RJ": "99.999.99-3", "RN": "20.040.040-1",
		"RS": "224/3658792", "RO": "0000000062521-3", "RR": "24006628-1", "SC": "251.040.852",
		"SP": "110.042.490.114", "SE": "27123456-3", "TO": "29.01.022783-6",
	}
	for uf, ie := range validas {
		if !IEValida(ie, uf) {
			t.Errorf("%s %s deveria ser válida", uf, ie)
		}
	}
	extras := [][2]string{{"BA", "1000003-06"}, {"PE", "18.1.001.0000004-9"}, {"RN", "20.0.040.040-0"}, {"PR", "123.45678-50"}, {"SP", "P-01100424.3/002"}, {"SP", "ISENTO"}, {"SP", ""}}
	for _, c := range extras {
		if !IEValida(c[1], c[0]) {
			t.Errorf("%s %s deveria ser válida", c[0], c[1])
		}
	}
	invalidas := [][2]string{{"SP", "100005962"}, {"SP", "110042490115"}, {"PR", "9103144678"}, {"SP", "200000419"}, {"MG", "0623079040082"}, {"XX", "123"}, {"SP", "ABC"}}
	for _, c := range invalidas {
		if IEValida(c[1], c[0]) {
			t.Errorf("%s %s deveria ser inválida", c[0], c[1])
		}
	}
}
