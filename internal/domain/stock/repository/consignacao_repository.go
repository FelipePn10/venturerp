package repository

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// ConsignacaoEstoque é a consignação e a transferência entre CNPJs vistas pela NF-e
// e pelas telas de estoque.
type ConsignacaoEstoque interface {
	AlmoxDeConsignacao(ctx context.Context) (int64, error)
	ConferirAlmoxDeConsignacao(ctx context.Context, almox int64) error
	RegistrarRemessa(ctx context.Context, m entity.RemessaConsignada) error
	SaldoConsignado(ctx context.Context, cliente, item int64, mask string) (decimal.Decimal, error)
	RegistrarVendaConsignada(ctx context.Context, v entity.VendaConsignada) error
	DevolverConsignacao(ctx context.Context, b entity.BaixaConsignacao) (*entity.Consignacao, error)
	ConsumirConsignacao(ctx context.Context, b entity.BaixaConsignacao) (*entity.Consignacao, error)
	ReceberConsignacao(ctx context.Context, c entity.Consignacao) (*entity.Consignacao, error)
	ListarConsignacoes(ctx context.Context, direcao string, soAbertas bool) ([]entity.Consignacao, error)
	PodeCancelarNotaDeEstoque(ctx context.Context, nota int64) error
	EmpresaIrma(ctx context.Context, cnpj string) (int64, error)
	EnviarEntreCNPJs(ctx context.Context, e entity.EnvioEntreCNPJs) error
	TransferenciasEntreCNPJs(ctx context.Context) ([]entity.TransferenciaEntreCNPJs, error)
}
