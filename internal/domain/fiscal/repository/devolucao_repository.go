package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
)

const (
	OrigemDevolucao        = "NFE_DEVOLUCAO"
	OrigemDevolucaoEstorno = "NFE_DEVOL_ESTORNO" // source_type tem 20 caracteres
)

// SaidaDevolucao é a linha da devolução que movimenta o estoque.
type SaidaDevolucao struct {
	FiscalEntryItemID int64
	ItemCode          int64
	WarehouseID       int64
	Quantidade        decimal.Decimal // na unidade de estoque
}

// MontagemDevolucao recebe o custo médio que saiu do estoque por linha da nota
// de entrada e devolve a contabilização (nil quando desligada).
type MontagemDevolucao func(custoPorItem map[int64]decimal.Decimal) (*contabilizacao.Lote, error)

// EfetivacaoDevolucao é o que a autorização da NF-e de devolução efetiva.
type EfetivacaoDevolucao struct {
	ExitID         int64
	EntryID        int64
	NumeroNF       int64
	Valor          decimal.Decimal // o que o fornecedor deve de volta
	FornecedorCode *int64
	FornecedorCNPJ string
	Data           time.Time
	Saidas         []SaidaDevolucao
	Montar         MontagemDevolucao
	UserID         uuid.UUID
}

type ResultadoDevolucao struct {
	Abatido        decimal.Decimal // abatido dos títulos em aberto da nota
	CreditoReceber decimal.Decimal // virou título a receber do fornecedor
	ContaReceberID *int64
	Movimentos     int
}

type DevolucaoRepository interface {
	// QuantidadesDevolvidas: por linha da nota de entrada, o que já está em
	// notas de devolução não canceladas nem rejeitadas.
	QuantidadesDevolvidas(ctx context.Context, entryID int64) (map[int64]decimal.Decimal, error)
	// EfetivarDevolucao (idempotente): estoque, abatimento dos títulos, crédito
	// a receber do que sobrar e contabilização, numa transação.
	EfetivarDevolucao(ctx context.Context, e EfetivacaoDevolucao) (*ResultadoDevolucao, error)
	// EstornarDevolucao desfaz o financeiro e a contabilização da devolução
	// cancelada (o estoque volta pelo estorno da saída).
	EstornarDevolucao(ctx context.Context, exitID int64, data time.Time) error
	// ChecarEstornoDevolucao diz, sem alterar nada, se a devolução pode ser
	// desfeita (o cancelamento na SEFAZ só vai se puder).
	ChecarEstornoDevolucao(ctx context.Context, exitID int64) error
}
