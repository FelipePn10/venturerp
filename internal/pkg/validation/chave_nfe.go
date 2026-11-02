package validation

import (
	"fmt"
	"strconv"
	"strings"
)

// ChaveNFe é a chave de acesso de 44 dígitos desmontada. Tudo o que ela carrega
// tem de bater com a nota: uma chave de outra nota, colada por engano, passaria
// por qualquer conferência que olhasse só o tamanho.
type ChaveNFe struct {
	CodigoUF  string
	AnoMes    string // AAMM da emissão
	Documento string // CNPJ do emitente, ou CPF completado com zeros à esquerda
	Modelo    string
	Serie     int
	Numero    int64
}

// CodigoIBGEDaUF traz o código da UF usado na chave e no cadastro da Receita.
var CodigoIBGEDaUF = map[string]string{
	"RO": "11", "AC": "12", "AM": "13", "RR": "14", "PA": "15", "AP": "16", "TO": "17",
	"MA": "21", "PI": "22", "CE": "23", "RN": "24", "PB": "25", "PE": "26", "AL": "27", "SE": "28", "BA": "29",
	"MG": "31", "ES": "32", "RJ": "33", "SP": "35",
	"PR": "41", "SC": "42", "RS": "43",
	"MS": "50", "MT": "51", "GO": "52", "DF": "53",
}

// UFValida diz se a sigla é de um estado brasileiro.
func UFValida(uf string) bool {
	_, ok := CodigoIBGEDaUF[strings.ToUpper(strings.TrimSpace(uf))]
	return ok
}

// LerChaveNFe devolve a chave desmontada; false quando ela não tem 44 dígitos ou
// o dígito verificador (módulo 11) não confere.
func LerChaveNFe(chave string) (ChaveNFe, bool) {
	c := stripNonDigits(chave)
	if len(c) != 44 || digitoDaChave(c[:43]) != c[43] {
		return ChaveNFe{}, false
	}
	serie, errSerie := strconv.Atoi(c[22:25])
	numero, errNumero := strconv.ParseInt(c[25:34], 10, 64)
	if errSerie != nil || errNumero != nil {
		return ChaveNFe{}, false
	}
	return ChaveNFe{CodigoUF: c[:2], AnoMes: c[2:6], Documento: c[6:20], Modelo: c[20:22], Serie: serie, Numero: numero}, true
}

func digitoDaChave(base string) byte {
	pesos := []int{2, 3, 4, 5, 6, 7, 8, 9}
	soma := 0
	for i := len(base) - 1; i >= 0; i-- {
		soma += int(base[i]-'0') * pesos[(len(base)-1-i)%len(pesos)]
	}
	d := 11 - soma%11
	if d >= 10 {
		d = 0
	}
	return byte('0' + d)
}

// MontarChaveNFe monta uma chave válida com o dígito verificador. O emissor real é
// quem gera a chave; isto serve a testes e a dados de demonstração.
func MontarChaveNFe(codigoUF, anoMes, documento string, serie int, numero, codigoNumerico int64) string {
	base := fmt.Sprintf("%s%s%014s55%03d%09d1%08d", codigoUF, anoMes, documento, serie, numero, codigoNumerico%100000000)
	return base + string(digitoDaChave(base))
}
