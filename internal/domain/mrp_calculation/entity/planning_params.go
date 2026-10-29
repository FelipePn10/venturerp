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
	// Janela de consumo pela venda (46 e 47): a venda na data V consome a
	// previsão e a demanda de estoque com data entre V − antes e V + depois.
	ConsumoDiasAntes  int
	ConsumoDiasDepois int
	// Horizonte firme (48, dias corridos, 0 = desligado) e tempo de segurança
	// (49, dias úteis) da empresa; a política do item sobrepõe os dois.
	HorizonteFirmeDias int
	TempoSegurancaDias int
	// Politicas é a política de planejamento efetiva dos itens que têm regra
	// (item ou classificação). Item fora do mapa segue os parâmetros acima.
	// Preenchida pelo cálculo (por execução), não pelo LoadFromDB.
	Politicas map[int64]PoliticaEfetiva
}

// Politica devolve a política de planejamento do item: a efetiva da regra ou,
// sem regra, os parâmetros da empresa.
func (p *TypedPlanningParams) Politica(item int64) PoliticaEfetiva {
	if pol, ok := p.Politicas[item]; ok {
		return pol
	}
	return p.PoliticaDaEmpresa()
}

// PoliticaDaEmpresa é a política que os parâmetros da empresa dão a um item
// sem regra própria.
func (p *TypedPlanningParams) PoliticaDaEmpresa() PoliticaEfetiva {
	return PoliticaEfetiva{
		Estrategia: EstrategiaConsumo, ConsumoDiasAntes: p.ConsumoDiasAntes, ConsumoDiasDepois: p.ConsumoDiasDepois,
		HorizonteFirmeDias: p.HorizonteFirmeDias, TempoSegurancaDias: p.TempoSegurancaDias,
		Origem: OrigemPolitica{Estrategia: OrigemEmpresa, Janela: OrigemEmpresa, HorizonteFirme: OrigemEmpresa, TempoSeguranca: OrigemEmpresa},
	}
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
		ConsumoDiasAntes:             30,
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
	if v, ok := raw[46]; ok {
		p.ConsumoDiasAntes = naoNegativo(parseInt(v, 30))
	}
	if v, ok := raw[47]; ok {
		p.ConsumoDiasDepois = naoNegativo(parseInt(v, 0))
	}
	if v, ok := raw[48]; ok {
		p.HorizonteFirmeDias = naoNegativo(parseInt(v, 0))
	}
	if v, ok := raw[49]; ok {
		p.TempoSegurancaDias = naoNegativo(parseInt(v, 0))
	}
}

func naoNegativo(v int) int {
	if v < 0 {
		return 0
	}
	return v
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
