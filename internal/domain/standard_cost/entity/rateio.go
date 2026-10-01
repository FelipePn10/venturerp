package entity

import (
	"sort"
	"time"
)

// AplicarRateios aplica o esquema de indiretos aos componentes de custo e devolve
// o total de overhead com o rastro de cada regra.
//
// A ORDEM importa e é fixa, não a ordem de cadastro: primeiro as bases
// específicas (material, setup, máquina, mão de obra, subcontratação), e só
// depois CONVERSAO e TOTAL. Sem isso, duas instalações com as mesmas regras
// cadastradas em ordem diferente chegariam a custos diferentes — e ninguém
// conseguiria explicar a divergência.
//
// Nenhuma regra incide sobre o overhead já aplicado: a base de TOTAL é
// material + conversão + subcontratação, de propósito. Indireto sobre indireto
// composto faria a ordem das regras mudar o resultado, que é justamente o que
// esta função elimina.
func AplicarRateios(
	componentes ComponentesDeCusto,
	horas HorasDoRoteiro,
	regras []*RegraDeRateio,
) (float64, []RateioAplicado) {
	ordenadas := make([]*RegraDeRateio, len(regras))
	copy(ordenadas, regras)
	sort.SliceStable(ordenadas, func(i, j int) bool {
		pi, pj := precedenciaDaBase(ordenadas[i].Base), precedenciaDaBase(ordenadas[j].Base)
		if pi != pj {
			return pi < pj
		}
		// Empate resolvido pelo código da regra: resultado reproduzível não pode
		// depender da ordem em que o banco devolveu as linhas.
		return ordenadas[i].Code < ordenadas[j].Code
	})

	var total float64
	rastro := make([]RateioAplicado, 0, len(ordenadas))
	for _, regra := range ordenadas {
		var baseValor, aplicado float64
		switch regra.Method {
		case MetodoPercentual:
			baseValor = regra.ValorDaBase(componentes)
			aplicado = baseValor * regra.Rate
		case MetodoValorPorHora:
			baseValor = regra.HorasDaBase(horas.Maquina, horas.MaoDeObra, horas.Setup)
			aplicado = baseValor * regra.Rate
		case MetodoValorPorUnidade:
			// Já é por unidade: o custo aqui é sempre unitário.
			baseValor = 1
			aplicado = regra.Rate
		default:
			continue
		}
		if aplicado == 0 {
			// Base zerada não gera linha de rastro: item sem roteiro apareceria com
			// cinco indiretos de R$ 0,00 e esconderia os que valem.
			continue
		}
		total += aplicado
		rastro = append(rastro, RateioAplicado{
			RuleID: regra.ID, Code: regra.Code, Description: regra.Description,
			Base: regra.Base, Method: regra.Method, Rate: regra.Rate,
			BaseValue: baseValor, Applied: aplicado,
		})
	}
	return total, rastro
}

// precedenciaDaBase fixa a ordem de aplicação. Bases específicas primeiro; as
// agregadoras (CONVERSAO, TOTAL) depois, porque dependem do que as outras já
// somaram nos componentes de conversão.
func precedenciaDaBase(b BaseDeRateio) int {
	switch b {
	case BaseMaterial, BaseSetup, BaseMaquina, BaseMaoDeObra, BaseSubcontratacao:
		return 0
	case BaseConversao:
		return 1
	case BaseTotal:
		return 2
	default:
		return 3
	}
}

// RegrasVigentes filtra as regras que valem para este item na data da apuração:
// vigência E escopo. As duas condições juntas — uma regra vigente de outro centro
// de trabalho, ou uma regra deste centro fora de vigência, não podem entrar.
func RegrasVigentes(
	todas []*RegraDeRateio,
	itemCode int64,
	centrosDoRoteiro map[int64]bool,
	quando time.Time,
) []*RegraDeRateio {
	out := make([]*RegraDeRateio, 0, len(todas))
	for _, r := range todas {
		if r.VigenteEm(quando) && r.AplicaAoItem(itemCode, centrosDoRoteiro) {
			out = append(out, r)
		}
	}
	return out
}
