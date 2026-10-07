package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/frete"
)

const (
	FreteStatusPendente  = "PENDENTE"
	FreteStatusLancado   = "LANCADO"
	FreteStatusCancelado = "CANCELADO"
	// Origens do frete na contabilidade e no estoque.
	OrigemFrete        = "FRETE_COMPRA"
	OrigemFreteEstorno = "FRETE_COMPRA_ESTORNO"
)

// FreightDocument é o CT-e da transportadora que trouxe a mercadoria comprada.
type FreightDocument struct {
	ID                 int64
	ChaveCTe           *string
	Numero             int64
	Serie              string
	DataEmissao        time.Time
	CNPJTransportadora string
	NomeTransportadora string
	UFTransportadora   *string
	SupplierCode       *int64
	SupplierName       *string
	CFOP               *string
	ValorFrete         decimal.Decimal
	BaseICMS           decimal.Decimal
	AliqICMS           decimal.Decimal
	ValorICMS          decimal.Decimal
	CreditaICMS        bool
	TipoRateio         string
	DataVencimento     time.Time
	Status             string
	ContaPagarID       *int64
	Observacao         *string
	XMLContent         *string
	CreatedBy          uuid.UUID
	CreatedAt          time.Time
	LancadoEm          *time.Time
	CanceladoEm        *time.Time
	CancelReason       *string
	Notas              []FreightEntryRef
	Alocacoes          []FreightAllocation
}

// FreightEntryRef é uma nota de entrada transportada pelo frete.
type FreightEntryRef struct {
	FiscalEntryID int64
	NumeroNF      int64
	Serie         string
	Emitente      string
	ValorTotal    decimal.Decimal
	Status        string
	ChaveAcesso   *string
}

// FreightAllocation é a parte do frete atribuída a um item, já lançada.
type FreightAllocation struct {
	FiscalEntryID     int64
	FiscalEntryItemID int64
	ItemCode          *int64
	WarehouseID       *int64
	PlanoContasID     *int64
	CentroCustoID     *int64
	Descricao         string
	Valor             decimal.Decimal
	ValorEstoque      decimal.Decimal
	ValorDespesa      decimal.Decimal
	StockMovementID   *int64
}

// MontagemFrete recebe o rateio calculado sobre o saldo travado e devolve o
// título da transportadora e a contabilização (nil quando desligada).
type MontagemFrete func(partes []frete.Parte) (TituloAPagar, *contabilizacao.Lote, error)

// LancamentoFrete é o que o caso de uso manda o repositório efetivar.
type LancamentoFrete struct {
	FreightID int64
	Custo     decimal.Decimal
	Montar    MontagemFrete
	UserID    uuid.UUID
}

type CancelamentoFrete struct {
	FreightID   int64
	Motivo      string
	UserID      uuid.UUID
	DataEstorno time.Time
}

type FreightRepository interface {
	CreateFreight(ctx context.Context, f *FreightDocument, entryIDs []int64) (*FreightDocument, error)
	UpdateFreight(ctx context.Context, f *FreightDocument, entryIDs []int64) error
	GetFreight(ctx context.Context, id int64) (*FreightDocument, error)
	ListFreights(ctx context.Context, status string) ([]*FreightDocument, error)
	// EntriesByChaves acha as notas de entrada (não canceladas) pelas chaves.
	EntriesByChaves(ctx context.Context, chaves []string) ([]int64, error)
	// FreteItens devolve os itens das notas para o rateio, com o saldo atual.
	FreteItens(ctx context.Context, entryIDs []int64) ([]frete.Item, error)
	LancarFrete(ctx context.Context, l LancamentoFrete) error
	CancelarFrete(ctx context.Context, c CancelamentoFrete) error
	// FreteDoTitulo: o frete de origem do título a pagar (nil se não é de frete).
	FreteDoTitulo(ctx context.Context, contaPagarID int64) (*int64, error)
}
