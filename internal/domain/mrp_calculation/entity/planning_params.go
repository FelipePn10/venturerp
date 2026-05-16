package entity

import (
	"strconv"
	"strings"
)

type TypedPlanningParams struct {
	AgrupaDemandaEstoque            bool
	CodFornecedorInterface          string
	CodClienteInterface             string
	GerarDemandaSegurancaTodos      bool
	ObrigatoriedadeRefugo           bool
	DataNecessidadeEstoqueFuturo    bool
	GerarPrioridadesOrdens          bool
	DiasPrioridades                 int
	ItensFantasmasGravar            bool
	DesconsideraSemanasPassadas     bool
	ConsideraDatasTanques           bool
	VerificaSituacaoPedidoProjeto   bool
	UtilizaCalculoMPS               bool
	ProporcaoEntrega                string
	ValidaRestricoesEstrutura       bool
	TrataAssistenciaTecnica         bool
	PorcentagemProporcaoValorizacao float64
	DefaultPosicao                  string
	FormulaPerdasEstrutura          int
	NumeracaoOrdens                 string
	ObrigarControleEstoqueTerceiros bool
}

func DefaultTypedPlanningParams() *TypedPlanningParams {
	return &TypedPlanningParams{
		AgrupaDemandaEstoque:         true,
		GerarDemandaSegurancaTodos:   true,
		DataNecessidadeEstoqueFuturo: true,
		GerarPrioridadesOrdens:       true,
		DiasPrioridades:              5,
		ItensFantasmasGravar:         false,
		DesconsideraSemanasPassadas:  true,
		ValidaRestricoesEstrutura:    true,
		FormulaPerdasEstrutura:       2,
		NumeracaoOrdens:              "AUTO",
	}
}

func (p *TypedPlanningParams) LoadFromDB(raw map[int]string) {
	*p = *DefaultTypedPlanningParams()

	if v, ok := raw[1]; ok {
		p.AgrupaDemandaEstoque = parseBool(v)
	}
	if v, ok := raw[2]; ok {
		p.CodFornecedorInterface = strings.TrimSpace(v)
	}
	if v, ok := raw[3]; ok {
		p.CodClienteInterface = strings.TrimSpace(v)
	}
	if v, ok := raw[4]; ok {
		p.GerarDemandaSegurancaTodos = parseBool(v)
	}
	if v, ok := raw[5]; ok {
		p.ObrigatoriedadeRefugo = parseBool(v)
	}
	if v, ok := raw[6]; ok {
		p.DataNecessidadeEstoqueFuturo = parseBool(v)
	}
	if v, ok := raw[7]; ok {
		p.GerarPrioridadesOrdens = parseBool(v)
	}
	if v, ok := raw[8]; ok {
		p.DiasPrioridades = parseInt(v, 5)
	}
	if v, ok := raw[10]; ok {
		p.ItensFantasmasGravar = parseBool(v)
	}
	if v, ok := raw[11]; ok {
		p.DesconsideraSemanasPassadas = parseBool(v)
	}
	if v, ok := raw[12]; ok {
		p.ConsideraDatasTanques = parseBool(v)
	}
	if v, ok := raw[13]; ok {
		p.VerificaSituacaoPedidoProjeto = parseBool(v)
	}
	if v, ok := raw[14]; ok {
		p.UtilizaCalculoMPS = parseBool(v)
	}
	if v, ok := raw[15]; ok {
		p.ProporcaoEntrega = strings.TrimSpace(v)
	}
	if v, ok := raw[16]; ok {
		p.ValidaRestricoesEstrutura = parseBool(v)
	}
	if v, ok := raw[17]; ok {
		p.TrataAssistenciaTecnica = parseBool(v)
	}
	if v, ok := raw[18]; ok {
		p.PorcentagemProporcaoValorizacao = parseFloat(v, 0)
	}
	if v, ok := raw[19]; ok {
		p.DefaultPosicao = strings.TrimSpace(v)
	}
	if v, ok := raw[20]; ok {
		p.FormulaPerdasEstrutura = parseInt(v, 2)
	}
	if v, ok := raw[24]; ok {
		p.NumeracaoOrdens = strings.TrimSpace(v)
	}
	if v, ok := raw[45]; ok {
		p.ObrigarControleEstoqueTerceiros = parseBool(v)
	}
}

func parseBool(v string) bool {
	v = strings.TrimSpace(strings.ToUpper(v))
	return v == "S" || v == "1" || v == "TRUE" || v == "YES"
}

func parseInt(v string, defaultVal int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func parseFloat(v string, defaultVal float64) float64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defaultVal
	}
	return f
}
