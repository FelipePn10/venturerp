// Package pendencias: a central de pendências do dia (melhoria 8). Uma caixa
// só, filtrada pelo perfil: cada um vê o que os módulos dele podem resolver.
package pendencias

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
)

// Tipo é uma pendência do catálogo.
type Tipo struct {
	Chave      string        `json:"key"`
	Area       string        `json:"area"`
	Titulo     string        `json:"title"`
	Tela       string        `json:"screen"`
	Modulo     acesso.Modulo `json:"module"`
	Severidade string        `json:"severity"` // ALTA, MEDIA, BAIXA
}

// Contagem é o que o banco respondeu para um tipo.
type Contagem struct {
	Quantidade int
	Valor      decimal.Decimal
	MaisAntiga *time.Time
}

// Pendencia é um tipo com quantidade (só aparece o que tem).
type Pendencia struct {
	Tipo
	Quantidade int             `json:"count"`
	Valor      decimal.Decimal `json:"value,omitempty"`
	MaisAntiga *time.Time      `json:"oldest,omitempty"`
}

var Catalogo = []Tipo{
	{Chave: "ofs_atrasadas", Area: "Produção", Titulo: "OFs atrasadas (fim previsto vencido)", Tela: "VPRO0900", Modulo: acesso.Producao, Severidade: "ALTA"},
	{Chave: "sugestoes_a_firmar", Area: "Planejamento", Titulo: "Sugestões do MRP para firmar nesta semana", Tela: "VMRP0100", Modulo: acesso.Planejamento, Severidade: "ALTA"},
	{Chave: "excecoes_mrp", Area: "Planejamento", Titulo: "Exceções do último cálculo do MRP", Tela: "VMRP0100", Modulo: acesso.Planejamento, Severidade: "MEDIA"},
	{Chave: "compras_sem_preco", Area: "Suprimentos", Titulo: "Linhas de pedido de compra sem preço", Tela: "VPDC0200", Modulo: acesso.Suprimentos, Severidade: "ALTA"},
	{Chave: "remessas_terceiro_vencidas", Area: "Suprimentos", Titulo: "Peças com terceiro além do prazo (OS ou prazo legal)", Tela: "VTPS0100", Modulo: acesso.Suprimentos, Severidade: "ALTA"},
	{Chave: "pedidos_portal", Area: "Comercial", Titulo: "Pedidos que o cliente fez pelo portal, a confirmar", Tela: "VVND0200", Modulo: acesso.Comercial, Severidade: "ALTA"},
	{Chave: "notas_a_lancar", Area: "Fiscal", Titulo: "Notas de entrada a conferir e aprovar", Tela: "VFIS0210", Modulo: acesso.Fiscal, Severidade: "MEDIA"},
	{Chave: "efeitos_nfe_pendentes", Area: "Fiscal", Titulo: "NF-e autorizadas com estoque/títulos pendentes", Tela: "VFIS0200", Modulo: acesso.Fiscal, Severidade: "ALTA"},
	{Chave: "pagar_vencendo", Area: "Financeiro", Titulo: "Títulos a pagar vencidos ou vencendo em 3 dias", Tela: "VFIN0200", Modulo: acesso.Financeiro, Severidade: "ALTA"},
	{Chave: "receber_vencidos", Area: "Financeiro", Titulo: "Títulos a receber vencidos", Tela: "VFIN0210", Modulo: acesso.Financeiro, Severidade: "MEDIA"},
	{Chave: "inventarios_abertos", Area: "Estoque", Titulo: "Inventários abertos", Tela: "VEST0200", Modulo: acesso.Estoque, Severidade: "BAIXA"},
}

// DoPerfil devolve as pendências com quantidade que o perfil pode resolver,
// na ordem do catálogo (severidade e fluxo).
func DoPerfil(contagens map[string]Contagem, pode func(acesso.Modulo) bool) []Pendencia {
	out := []Pendencia{}
	for _, t := range Catalogo {
		c := contagens[t.Chave]
		if c.Quantidade == 0 || (pode != nil && !pode(t.Modulo)) {
			continue
		}
		out = append(out, Pendencia{Tipo: t, Quantidade: c.Quantidade, Valor: c.Valor, MaisAntiga: c.MaisAntiga})
	}
	return out
}
