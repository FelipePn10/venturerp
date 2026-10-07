package repository

import (
	"context"

	"github.com/shopspring/decimal"
)

// SalesOrderInvoicingRepository é o que a nota de saída precisa saber do
// pedido de venda para faturar total ou parcialmente sem faturar duas vezes.
type SalesOrderInvoicingRepository interface {
	// QuantidadesEmNota devolve, por linha do pedido, quanto já está em nota
	// ainda não autorizada (rascunho ou aguardando a SEFAZ). O que já foi
	// autorizado está no atendido da linha.
	QuantidadesEmNota(ctx context.Context, salesOrderCode int64) (map[int64]decimal.Decimal, error)
	// RegistrarFaturamento soma (ou, no estorno, subtrai) ao atendido das
	// linhas do pedido as quantidades da nota. Devolve se a nota tinha linhas
	// ligadas ao pedido e se, depois disso, o pedido ficou todo atendido.
	RegistrarFaturamento(ctx context.Context, fiscalExitID int64, estorno bool) (ligada bool, pedidoAtendido bool, err error)
	// NotasDoPedido lista as notas de saída ativas do pedido.
	NotasDoPedido(ctx context.Context, salesOrderCode int64) ([]NotaResumo, error)
	// NomeRepresentante devolve o nome do representante.
	NomeRepresentante(ctx context.Context, code int64) (string, error)
	// TravarPedido serializa o faturamento do pedido na empresa: dois usuários
	// faturando o mesmo pedido ao mesmo tempo não podem, os dois, ler o mesmo
	// saldo e emitir duas notas pela quantidade inteira. Não espera: com o
	// pedido já em faturamento, devolve conflito. Liberar é obrigatório.
	TravarPedido(ctx context.Context, salesOrderCode int64) (liberar func(), err error)
}

type NotaResumo struct {
	ID         int64
	NumeroNF   int64
	Status     string
	ValorTotal decimal.Decimal
}
