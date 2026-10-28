// Package reforma calcula o grupo IBS/CBS da NF-e na transição da reforma
// tributária (LC 214/2025).
//
// As alíquotas vêm do cadastro da empresa (por ano) e, na falta dele, da lei:
// 2026 tem CBS 0,9% e IBS estadual 0,1% (arts. 343 e 346); 2027 e 2028 têm IBS
// estadual 0,05% e municipal 0,05% (art. 344). A CBS de 2027 em diante é a
// alíquota de referência fixada pela União menos 0,1 ponto (art. 347) — não é
// número da lei, então só existe se a empresa cadastrar.
//
// O tratamento do item (CST, cClassTrib, redução) vem da tabela oficial de
// classificação tributária: CST sem exigência de tributo (isenção, imunidade,
// não incidência) leva só os códigos, sem o grupo de valores; CST com redução
// aplica o percentual de redução sobre a alíquota.
package reforma

import (
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"

	"github.com/shopspring/decimal"
)

// Aliquotas do IBS (estadual e municipal) e da CBS, em %.
type Aliquotas struct {
	IBSUF, IBSMun, CBS decimal.Decimal
	// Fonte diz de onde vieram: "LEI" ou "EMPRESA".
	Fonte string
}

const (
	FonteLei     = "LEI"
	FonteEmpresa = "EMPRESA"
)

// AliquotasDaLei devolve o que a LC 214/2025 fixa para o ano. A CBS volta nil
// quando a lei não traz o número (2027 em diante), e ok é falso fora da
// transição coberta (2026 a 2028).
func AliquotasDaLei(ano int) (ibsUF, ibsMun decimal.Decimal, cbs *decimal.Decimal, ok bool) {
	switch {
	case ano == 2026:
		c := decimal.RequireFromString("0.9")
		return decimal.RequireFromString("0.1"), decimal.Zero, &c, true
	case ano == 2027 || ano == 2028:
		return decimal.RequireFromString("0.05"), decimal.RequireFromString("0.05"), nil, true
	}
	return decimal.Zero, decimal.Zero, nil, false
}

// ResolverAliquotas: o cadastro da empresa prevalece; sem ele, a lei. Sem CBS
// conhecida a nota não sai — melhor recusar antes da SEFAZ do que transmitir
// tributo com alíquota errada.
func ResolverAliquotas(ano int, cadastro *Aliquotas) (Aliquotas, error) {
	if cadastro != nil {
		a := *cadastro
		a.Fonte = FonteEmpresa
		return a, nil
	}
	ibsUF, ibsMun, cbs, ok := AliquotasDaLei(ano)
	if !ok {
		return Aliquotas{}, domainerr.NewValidation(fmt.Sprintf(
			"alíquotas de IBS/CBS de %d não cadastradas: informe-as em Reforma Tributária › Alíquotas", ano))
	}
	if cbs == nil {
		return Aliquotas{}, domainerr.NewValidation(fmt.Sprintf(
			"a alíquota da CBS de %d não está cadastrada: ela é a alíquota de referência fixada pela União menos 0,1 ponto (LC 214/2025, art. 347) — informe-a em Reforma Tributária › Alíquotas", ano))
	}
	return Aliquotas{IBSUF: ibsUF, IBSMun: ibsMun, CBS: *cbs, Fonte: FonteLei}, nil
}

// Classificacao é a linha da tabela oficial que trata o item.
type Classificacao struct {
	ClassTrib, CST  string
	ExigeTributacao bool
	// Percentuais de redução da alíquota (0 a 100) do IBS e da CBS.
	PercReducaoIBS, PercReducaoCBS decimal.Decimal
}

// CSTPadrao e ClassTribPadrao: tributação integral (CST 000, cClassTrib 000001),
// o caso geral da venda e da devolução de mercadoria.
const (
	CSTPadrao       = "000"
	ClassTribPadrao = "000001"
)

// ClassificacaoPadrao é a tributação integral, sem redução.
func ClassificacaoPadrao() Classificacao {
	return Classificacao{ClassTrib: ClassTribPadrao, CST: CSTPadrao, ExigeTributacao: true}
}

// Origem do cClassTrib aplicado ao item, na ordem de precedência.
const (
	OrigemItem          = "ITEM_DA_NOTA"
	OrigemCFOP          = "CFOP"
	OrigemClassificacao = "CLASSIFICACAO_FISCAL"
	OrigemPadrao        = "PADRAO"
)

// EscolherClassTrib aplica a precedência: o que foi dito na linha da nota vale
// mais que o padrão da operação (CFOP), que vale mais que o do produto
// (classificação fiscal). Sem nenhum, a tributação integral. Bonificação,
// transferência e exportação são decididas pela operação, por isso o CFOP vence
// o produto.
func EscolherClassTrib(doItem, doCFOP, daClassificacao string) (string, string) {
	switch {
	case doItem != "":
		return doItem, OrigemItem
	case doCFOP != "":
		return doCFOP, OrigemCFOP
	case daClassificacao != "":
		return daClassificacao, OrigemClassificacao
	}
	return ClassTribPadrao, OrigemPadrao
}

// Tributo é o IBS estadual, o IBS municipal ou a CBS de um item.
type Tributo struct {
	Aliquota, PercReducao, AliquotaEfetiva, Valor decimal.Decimal
}

// Grupo é o IBS/CBS calculado de um item.
type Grupo struct {
	CST, ClassTrib string
	// ComValores falso: CST sem exigência de tributo — vão só os códigos.
	ComValores         bool
	Base               decimal.Decimal
	IBSUF, IBSMun, CBS Tributo
	IBS                decimal.Decimal
	ComReducao         bool
	FonteAliquotas     string
}

// Calcular monta o grupo do item. A base é o valor da operação (item + frete +
// seguro + outras − desconto incondicional) sem o ICMS, o PIS e a COFINS, que
// não a integram de 2026 a 2032 (art. 12, § 2º, V); o IPI já fica de fora do
// valor do item.
func Calcular(a Aliquotas, c Classificacao, valorOperacao, icms, pis, cofins decimal.Decimal) Grupo {
	g := Grupo{CST: c.CST, ClassTrib: c.ClassTrib, FonteAliquotas: a.Fonte}
	if !c.ExigeTributacao {
		return g
	}
	base := valorOperacao.Sub(icms).Sub(pis).Sub(cofins)
	if base.IsNegative() {
		base = decimal.Zero
	}
	g.ComValores = true
	g.Base = base.Round(2)
	g.ComReducao = c.PercReducaoIBS.IsPositive() || c.PercReducaoCBS.IsPositive()
	g.IBSUF = tributo(g.Base, a.IBSUF, c.PercReducaoIBS)
	g.IBSMun = tributo(g.Base, a.IBSMun, c.PercReducaoIBS)
	g.CBS = tributo(g.Base, a.CBS, c.PercReducaoCBS)
	g.IBS = g.IBSUF.Valor.Add(g.IBSMun.Valor)
	return g
}

var cem = decimal.NewFromInt(100)

func tributo(base, aliquota, reducao decimal.Decimal) Tributo {
	efetiva := aliquota.Mul(cem.Sub(reducao)).Div(cem).Round(4)
	return Tributo{Aliquota: aliquota, PercReducao: reducao, AliquotaEfetiva: efetiva,
		Valor: base.Mul(efetiva).Div(cem).Round(2)}
}
