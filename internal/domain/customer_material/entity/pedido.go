package entity

import "github.com/shopspring/decimal"

// PedidoDeBeneficiamento é o pedido de venda do serviço de industrialização,
// visto pelo beneficiamento: quem é o cliente, em que situação está e quanto de
// cada linha ainda falta faturar.
//
// O pedido é o comercial de sempre (VVEN), com a natureza de beneficiamento. O
// beneficiamento só o LÊ: confere a remessa contra ele e traz o serviço para a
// nota. Quem baixa o atendido é a autorização da NF-e, como em qualquer venda.
type PedidoDeBeneficiamento struct {
	Code         int64
	Numero       int64
	CustomerCode int64
	Status       string
	Ativo        bool
	Linhas       []LinhaDoPedido
}

// LinhaDoPedido é uma linha do pedido com o que ainda pode ser faturado.
type LinhaDoPedido struct {
	Code             int64
	Sequencia        int
	ItemCode         int64
	ItemBusinessCode string
	Descricao        string
	Unidade          string
	Quantidade       decimal.Decimal
	Atendida         decimal.Decimal
	Cancelada        decimal.Decimal
	// EmNota é o que já está em outra nota ainda não autorizada (rascunho ou
	// aguardando a SEFAZ). Contá-lo evita faturar a mesma quantidade duas vezes.
	EmNota        decimal.Decimal
	PrecoUnitario decimal.Decimal
	Status        string
}

// Status do pedido de venda que importam aqui (sales_orders.status).
const (
	PedidoFaturado  = "F"
	PedidoCancelado = "CANCELLED"
)

// Saldo é o que a linha ainda pode faturar. Nunca negativo.
func (l LinhaDoPedido) Saldo() decimal.Decimal {
	s := l.Quantidade.Sub(l.Atendida).Sub(l.Cancelada).Sub(l.EmNota)
	if s.IsNegative() {
		return decimal.Zero
	}
	return s
}

// LinhasComSaldo devolve as linhas ativas que ainda têm o que faturar.
func (p *PedidoDeBeneficiamento) LinhasComSaldo() []LinhaDoPedido {
	out := make([]LinhaDoPedido, 0, len(p.Linhas))
	for _, l := range p.Linhas {
		if l.Status == "CANCELLED" || !l.Saldo().IsPositive() {
			continue
		}
		out = append(out, l)
	}
	return out
}

// Linha procura a linha pelo código.
func (p *PedidoDeBeneficiamento) Linha(code int64) (LinhaDoPedido, bool) {
	for _, l := range p.Linhas {
		if l.Code == code {
			return l, true
		}
	}
	return LinhaDoPedido{}, false
}
