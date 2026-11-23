// Package implantacao: o placar "pronto para operar" de uma empresa
// (melhoria 1, .ai/tasks/rodada-validacao-bugs-e-melhorias.md).
//
// Quase todo problema que apareceu no meio do fluxo na implantação da Tecnofer
// era de cadastro: tipo de nota, plano de contas, endereço do fornecedor,
// conversão, lote múltiplo, custo-padrão. O painel confere cada um antes, dá o
// tamanho da pendência e diz em que tela corrigir.
package implantacao

import "math"

// Contagem é o que o banco respondeu para uma verificação: quantos registros
// estão pendentes num universo de Total. Total 0 com Pendentes 0 = nada a
// conferir (conta como pronto).
type Contagem struct {
	Pendentes int
	Total     int
}

// Definicao é uma verificação do catálogo.
type Definicao struct {
	Chave  string `json:"key"`
	Area   string `json:"area"`
	Titulo string `json:"title"`
	// Tela: o código da tela que corrige (o front monta o link por ele).
	Tela         string `json:"screen"`
	NomeTela     string `json:"screen_name"`
	ComoCorrigir string `json:"how_to_fix"`
	Peso         int    `json:"weight"`
}

// Item é uma verificação avaliada.
type Item struct {
	Definicao
	Pendentes int  `json:"pending"`
	Total     int  `json:"total"`
	Pronto    bool `json:"ready"`
	// Percentual da verificação (0 a 100): o que está certo do universo.
	Percentual int `json:"percent"`
}

// Painel é o placar da empresa.
type Painel struct {
	Percentual int    `json:"percent"`
	Prontos    int    `json:"ready_count"`
	Total      int    `json:"total_count"`
	Itens      []Item `json:"items"`
}

// Catalogo: a ordem é a da implantação (fiscal e contábil antes do chão de fábrica).
var Catalogo = []Definicao{
	{Chave: "config_fiscal", Area: "Fiscal", Titulo: "Configuração fiscal da empresa (CNPJ, UF e município IBGE)", Tela: "VFIS0100", NomeTela: "Configuração Fiscal",
		ComoCorrigir: "Preencha CNPJ, UF e o código IBGE do município.", Peso: 3},
	{Chave: "tipos_nota", Area: "Fiscal", Titulo: "Tipos de nota fiscal de saída", Tela: "VFIS0305", NomeTela: "Tipos de Nota Fiscal",
		ComoCorrigir: "Cadastre ao menos venda e remessa.", Peso: 2},
	{Chave: "tipos_entrada", Area: "Fiscal", Titulo: "Tipos de operação de entrada (TES)", Tela: "VFIS0360", NomeTela: "Tipos de Operação de Entrada",
		ComoCorrigir: "Cadastre compra para industrialização, uso e consumo e retorno de industrialização.", Peso: 2},
	{Chave: "plano_contas", Area: "Contábil", Titulo: "Plano de contas", Tela: "VFIN0120", NomeTela: "Plano de Contas",
		ComoCorrigir: "Importe ou cadastre o plano de contas da empresa.", Peso: 2},
	{Chave: "parametros_contabeis", Area: "Contábil", Titulo: "Parâmetros de contabilização (estoque e CMV)", Tela: "VCTB0200", NomeTela: "Parâmetros Contábeis",
		ComoCorrigir: "Informe o plano e as contas de Estoque e CMV.", Peso: 2},
	{Chave: "condicoes_pagamento", Area: "Financeiro", Titulo: "Condições de pagamento", Tela: "VFIN0110", NomeTela: "Condições de Pagamento",
		ComoCorrigir: "Cadastre as condições usadas na compra e na venda.", Peso: 1},
	{Chave: "fornecedores_endereco", Area: "Suprimentos", Titulo: "Fornecedores com endereço completo", Tela: "VSUP0500", NomeTela: "Cadastro de Fornecedor",
		ComoCorrigir: "Complete logradouro, município e UF: a NF-e de devolução e de remessa exige.", Peso: 2},
	{Chave: "fonte_suprimento", Area: "Suprimentos", Titulo: "Itens comprados com fornecedor preferencial", Tela: "VSUP0130", NomeTela: "Fornecedor Preferencial por Item",
		ComoCorrigir: "Sem fonte, a requisição e a OS de terceiro são recusadas.", Peso: 2},
	{Chave: "conversoes", Area: "Suprimentos", Titulo: "Itens comprados em outra unidade com conversão", Tela: "VSUP0110", NomeTela: "Conversão de UM por Item",
		ComoCorrigir: "Cadastre 1 UN = X KG (ou use o assistente de conversão).", Peso: 2},
	{Chave: "lote_multiplo", Area: "Planejamento", Titulo: "Itens comprados com lote múltiplo", Tela: "VENT0200", NomeTela: "Cadastro de Itens › Planejamento",
		ComoCorrigir: "Use 1 para quem compra peça inteira; o MRP arredonda a sugestão por ele.", Peso: 1},
	{Chave: "custo_padrao", Area: "Custos", Titulo: "Itens com custo-padrão calculado", Tela: "VCUS0100", NomeTela: "Custos",
		ComoCorrigir: "Rode o cálculo do custo-padrão pela estrutura.", Peso: 2},
	{Chave: "taxa_mao_obra", Area: "Custos", Titulo: "Centros de trabalho com taxa de mão de obra", Tela: "VCUS0100", NomeTela: "Custos",
		ComoCorrigir: "Informe o custo-hora de mão de obra (e de máquina) de cada centro.", Peso: 1},
	{Chave: "inventarios_abertos", Area: "Estoque", Titulo: "Nenhum inventário esquecido aberto", Tela: "VEST0200", NomeTela: "Inventário e tipos de movimento",
		ComoCorrigir: "Feche ou cancele o inventário: aberto, ele pode bloquear movimentos.", Peso: 1},
}

// Montar avalia o catálogo com as contagens do banco.
func Montar(contagens map[string]Contagem) Painel {
	var p Painel
	var pontos, pesos float64
	for _, d := range Catalogo {
		c := contagens[d.Chave]
		it := Item{Definicao: d, Pendentes: c.Pendentes, Total: c.Total, Pronto: c.Pendentes == 0}
		frac := 1.0
		if c.Pendentes > 0 {
			frac = 0
			if c.Total > c.Pendentes {
				frac = float64(c.Total-c.Pendentes) / float64(c.Total)
			}
		}
		it.Percentual = int(math.Floor(frac * 100))
		pontos += frac * float64(d.Peso)
		pesos += float64(d.Peso)
		if it.Pronto {
			p.Prontos++
		}
		p.Itens = append(p.Itens, it)
	}
	p.Total = len(Catalogo)
	if pesos > 0 {
		p.Percentual = int(math.Floor(pontos / pesos * 100))
	}
	return p
}
