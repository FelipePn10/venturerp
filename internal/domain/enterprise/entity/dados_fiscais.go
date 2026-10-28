package entity

import (
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// DadosFiscais: a identificação da empresa (fiscal_configs). É mantida num
// lugar só, a VEMP0100; a configuração fiscal (alíquotas, Focus, contas) fica
// na VFIS0100 e não mexe nela.
type DadosFiscais struct {
	CNPJ             string
	RazaoSocial      string
	NomeFantasia     string
	IE               string
	RegimeTributario string // "1" Simples, "2" Presumido, "3" Real
	UF               string
	Municipio        string
	CodigoMunicipio  string
	CEP              string
	Logradouro       string
	Numero           string
	Complemento      string
	Bairro           string
	Telefone         string
	Email            string
}

// PrefixoIBGEDaUF: os dois primeiros dígitos do código IBGE do município são o
// código da UF. Código de outra UF passa na NF-e (a Focus resolve pelo nome),
// mas sai errado no CT-e, na NFS-e e no SPED.
var PrefixoIBGEDaUF = map[string]string{
	"RO": "11", "AC": "12", "AM": "13", "RR": "14", "PA": "15", "AP": "16", "TO": "17",
	"MA": "21", "PI": "22", "CE": "23", "RN": "24", "PB": "25", "PE": "26", "AL": "27", "SE": "28", "BA": "29",
	"MG": "31", "ES": "32", "RJ": "33", "SP": "35", "PR": "41", "SC": "42", "RS": "43",
	"MS": "50", "MT": "51", "GO": "52", "DF": "53",
}

// ErroDoCodigoIBGE: vazio (ou zeros) é aceito; preenchido tem 7 dígitos e é da
// UF da empresa. Devolve a mensagem do erro, ou "".
func ErroDoCodigoIBGE(codigo, uf string) string {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" || strings.Trim(codigo, "0") == "" {
		return ""
	}
	if len(codigo) != 7 || strings.Trim(codigo, "0123456789") != "" {
		return fmt.Sprintf("código IBGE do município inválido (%q): são 7 dígitos, ex.: 4106902 para Curitiba", codigo)
	}
	u := strings.ToUpper(strings.TrimSpace(uf))
	if p, ok := PrefixoIBGEDaUF[u]; ok && !strings.HasPrefix(codigo, p) {
		return fmt.Sprintf("o código IBGE %s não é de município do %s (os do %s começam com %s)", codigo, u, u, p)
	}
	return ""
}

func soDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Normalizar e conferir: CNPJ com 14 dígitos, razão social, UF e regime.
func (d *DadosFiscais) Conferir() error {
	d.CNPJ, d.CEP, d.CodigoMunicipio, d.IE = soDigitos(d.CNPJ), soDigitos(d.CEP), soDigitos(d.CodigoMunicipio), strings.TrimSpace(d.IE)
	d.RazaoSocial, d.NomeFantasia = strings.TrimSpace(d.RazaoSocial), strings.TrimSpace(d.NomeFantasia)
	d.UF = strings.ToUpper(strings.TrimSpace(d.UF))
	d.RegimeTributario = strings.TrimSpace(d.RegimeTributario)
	if len(d.CNPJ) != 14 {
		return domainerr.NewValidation("CNPJ da empresa com 14 dígitos")
	}
	if d.RazaoSocial == "" {
		return domainerr.NewValidation("informe a razão social da empresa")
	}
	if len(d.UF) != 2 {
		return domainerr.NewValidation("informe a UF da empresa")
	}
	switch d.RegimeTributario {
	case "1", "2", "3":
	default:
		return domainerr.NewValidation("regime tributário: 1 (Simples), 2 (Presumido) ou 3 (Real)")
	}
	if d.CEP != "" && len(d.CEP) != 8 {
		return domainerr.NewValidation("CEP com 8 dígitos")
	}
	if msg := ErroDoCodigoIBGE(d.CodigoMunicipio, d.UF); msg != "" {
		return domainerr.NewValidation(msg)
	}
	if _, ok := PrefixoIBGEDaUF[d.UF]; !ok {
		return domainerr.NewValidation(fmt.Sprintf("UF %q inválida", d.UF))
	}
	d.Logradouro, d.Numero, d.Complemento = strings.TrimSpace(d.Logradouro), strings.TrimSpace(d.Numero), strings.TrimSpace(d.Complemento)
	d.Bairro, d.Municipio, d.Telefone, d.Email = strings.TrimSpace(d.Bairro), strings.TrimSpace(d.Municipio), strings.TrimSpace(d.Telefone), strings.TrimSpace(d.Email)
	return nil
}
