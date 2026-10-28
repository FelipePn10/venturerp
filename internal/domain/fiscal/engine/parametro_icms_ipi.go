package engine

import (
	"strings"

	"github.com/shopspring/decimal"
)

// ParametroICMSIPI é a regra do cadastro de parâmetros de ICMS/IPI (VFIS0320)
// que casou com a linha da nota: item ou NCM, UF de destino, cliente/segmento.
// Sem ela o motor usa a regra geral da empresa — alíquota interna única da
// configuração fiscal, tabela interestadual, IPI da tabela de NCM. Com ela, o
// que a regra traz prevalece, separado entre destinatário contribuinte e não
// contribuinte: é o que permite tributar diferente consumo final, revenda e
// cada NCM, em vez de cravar uma tributação padrão para a empresa inteira.
//
// Percentuais vêm como no cadastro (18 = 18%). Zero significa "a regra não
// informa" e o motor mantém o cálculo geral; para zerar o imposto, a regra
// informa o CST (40, 41, 50, 60 no ICMS; 51 a 55 no IPI).
type ParametroICMSIPI struct {
	ID         int64
	Contrib    LadoParametroICMSIPI
	NaoContrib LadoParametroICMSIPI
	// DiferimentoPct substitui o diferimento da configuração fiscal (só para
	// destinatário contribuinte, como na configuração).
	DiferimentoPct float64
	// AliqImportadoPct: alíquota interestadual de mercadoria importada (origens
	// 1, 2, 3 e 8 do cadastro). Sem ela vale a Resolução SF 13/2012 (4%).
	AliqImportadoPct float64
	// CSTIPI substitui o CST de IPI da tabela de NCM na saída.
	CSTIPI string
	// CodBeneficio é o cBenef geral da regra; o de cada lado vence este.
	CodBeneficio string
	// CSOSN do Simples Nacional; vazio = deduzido do CST calculado.
	CSOSN string
	// Substituição tributária: MvaPct é a MVA original (40 = 40%); MvaAjustar
	// liga o ajuste da MVA na venda interestadual; RedBaseSTPct reduz a base da ST.
	MvaPct       float64
	MvaAjustar   bool
	RedBaseSTPct float64
	// DIFAL: DifalTipo (TRIBUTADO, ISENTO_OUTRAS, NAO_CONSIDERA) e redução da
	// base do diferencial (%).
	DifalTipo   string
	DifalRedPct float64
	// FCPProdutoPct: FCP (%) do produto na UF de destino — o acréscimo de ICMS
	// da regra para não contribuinte do tipo Fundo de Combate à Pobreza. Fora do
	// RJ e de SE o FCP só incide sobre listas de produtos: vai na regra, e vence
	// o FCP geral da tabela de ICMS por UF. Zero = vale o da tabela.
	FCPProdutoPct float64
	// Dispositivos: o texto da base legal (VFIS0310) de cada tratamento da
	// regra — vai nas informações complementares da nota quando o tratamento
	// é aplicado (isenção, redução, diferimento, IPI).
	Dispositivos DispositivosLegais
}

// DispositivosLegais: a base legal de cada tratamento da regra.
type DispositivosLegais struct {
	ICMSContrib, ICMSNaoContrib       string
	ReducaoContrib, ReducaoNaoContrib string
	Diferimento                       string
	IPIContrib, IPINaoContrib         string
}

// DispositivosAplicados devolve, sem repetir, a base legal do que a linha
// calculada de fato usou: a do ICMS do destinatário quando o CST é de
// benefício ou sem imposto, a da redução quando houve redução de base, a do
// diferimento quando houve diferimento e a do IPI quando a regra o tratou.
func (p *ParametroICMSIPI) DispositivosAplicados(destinoTipo string, r TaxItemResult) []string {
	if p == nil {
		return nil
	}
	contrib := destinoTipo == "contribuinte"
	escolher := func(c, n string) string {
		if contrib {
			return c
		}
		return n
	}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		for _, x := range out {
			if x == s {
				return
			}
		}
		out = append(out, s)
	}
	switch r.CSTICMS {
	case "20", "30", "40", "41", "50", "60", "70", "90", "300", "400", "500":
		add(escolher(p.Dispositivos.ICMSContrib, p.Dispositivos.ICMSNaoContrib))
	}
	if r.ReducaoBaseICMSPct > 0 {
		add(escolher(p.Dispositivos.ReducaoContrib, p.Dispositivos.ReducaoNaoContrib))
	}
	if r.ValorICMSDiferido > 0 {
		add(p.Dispositivos.Diferimento)
	}
	redIPI := p.NaoContrib.RedIPIPct > 0
	if contrib {
		redIPI = p.Contrib.RedIPIPct > 0
	}
	if p.CSTIPI != "" || redIPI {
		add(escolher(p.Dispositivos.IPIContrib, p.Dispositivos.IPINaoContrib))
	}
	return out
}

// LadoParametroICMSIPI: o que a regra diz para um tipo de destinatário.
type LadoParametroICMSIPI struct {
	AliqICMSPct  float64
	CSTICMS      string
	RedICMSPct   float64
	AlvoRedICMS  string // BASE (reduz a base) ou PERCENTUAL (reduz a alíquota)
	RedIPIPct    float64
	AlvoRedIPI   string
	CodBeneficio string
	// AliqSTPct: alíquota interna do destino usada na ST; zero = a da tabela.
	AliqSTPct float64
}

// Lado escolhe o que vale para o destinatário da nota.
func (p *ParametroICMSIPI) Lado(destinoTipo string) LadoParametroICMSIPI {
	if destinoTipo == "contribuinte" {
		return p.Contrib
	}
	return p.NaoContrib
}

// BeneficioFiscal devolve o cBenef da regra para o destinatário (o do lado,
// senão o geral). Vazio quando a regra não traz.
func (p *ParametroICMSIPI) BeneficioFiscal(destinoTipo string) string {
	if p == nil {
		return ""
	}
	if c := strings.TrimSpace(p.Lado(destinoTipo).CodBeneficio); c != "" {
		return c
	}
	return strings.TrimSpace(p.CodBeneficio)
}

var cem = decimal.NewFromInt(100)

func fracao(pct float64) decimal.Decimal { return decimal.NewFromFloat(pct).Div(cem) }

// reduzir aplica a redução do lado: na BASE (padrão) ou no PERCENTUAL (a
// alíquota). Devolve o percentual de redução da BASE — é o pRedBC que a NF-e
// exige nos CST 20 e 70; redução na alíquota não tem campo próprio no XML.
func reduzir(base, aliq decimal.Decimal, redPct float64, alvo string) (decimal.Decimal, decimal.Decimal, float64) {
	if redPct <= 0 {
		return base, aliq, 0
	}
	fator := decimal.NewFromInt(1).Sub(fracao(redPct))
	if strings.EqualFold(strings.TrimSpace(alvo), "PERCENTUAL") {
		return base, aliq.Mul(fator), 0
	}
	return base.Mul(fator), aliq, redPct
}

// cstIPISemImposto: CSTs de saída do IPI sem destaque (alíquota zero, isenta,
// não tributada, imune, suspensão).
var cstIPISemImposto = map[string]bool{"51": true, "52": true, "53": true, "54": true, "55": true}

// cstICMSSemImposto: CSTs de ICMS sem imposto próprio destacado (isenta, não
// tributada, suspensão, cobrado anteriormente por ST).
var cstICMSSemImposto = map[string]bool{"40": true, "41": true, "50": true, "60": true}

// ajustarIPI aplica o CST e a redução de IPI da regra.
func (p *ParametroICMSIPI) ajustarIPI(destinoTipo string, base, aliq decimal.Decimal, cst string) (decimal.Decimal, decimal.Decimal, string) {
	if p == nil {
		return base, aliq, cst
	}
	if c := strings.TrimSpace(p.CSTIPI); c != "" {
		cst = c
	}
	if cstIPISemImposto[cst] {
		return base, decimal.Zero, cst
	}
	lado := p.Lado(destinoTipo)
	base, aliq, _ = reduzir(base, aliq, lado.RedIPIPct, lado.AlvoRedIPI)
	return base, aliq, cst
}

// ajustarICMS aplica alíquota e redução de ICMS da regra. `importada` indica
// mercadoria importada numa operação interestadual (Resolução SF 13/2012).
func (p *ParametroICMSIPI) ajustarICMS(destinoTipo string, base, aliq decimal.Decimal, importada bool) (decimal.Decimal, decimal.Decimal, float64) {
	if p == nil {
		return base, aliq, 0
	}
	lado := p.Lado(destinoTipo)
	switch {
	case importada && p.AliqImportadoPct > 0:
		aliq = fracao(p.AliqImportadoPct)
	case !importada && lado.AliqICMSPct > 0:
		aliq = fracao(lado.AliqICMSPct)
	}
	return reduzir(base, aliq, lado.RedICMSPct, lado.AlvoRedICMS)
}

// cstICMS é o CST que a regra impõe ao destinatário ("" quando não impõe).
func (p *ParametroICMSIPI) cstICMS(destinoTipo string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(p.Lado(destinoTipo).CSTICMS)
}

// ConsultaParametroICMSIPI é o que identifica a operação na busca da regra.
type ConsultaParametroICMSIPI struct {
	ItemCode *int64
	NCM      *string
	// UF da outra ponta: destino na saída, origem (fornecedor) na entrada.
	UF      string
	Cliente *int64
	// Simples: a empresa é optante (regras "optante do Simples" passam a valer).
	Simples bool
	// Finalidade da compra; vazio = só regras sem finalidade.
	Finalidade string
	// TipoNotaID: tipo de nota (VFIS0110) da saída; nil = só regras sem tipo.
	TipoNotaID *int64
	// Entrada busca regras de ENTRADA/AMBAS; o padrão é a saída (SAIDA/AMBAS).
	Entrada bool
}
