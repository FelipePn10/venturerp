package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
)

// ErrCodigoEmUso: já existe demanda com o código na empresa.
var ErrCodigoEmUso = errors.New("código de demanda em uso")

type IndependentDemandRepository interface {
	Create(ctx context.Context, d *entity.IndependentDemand) (*entity.IndependentDemand, error)
	Update(ctx context.Context, d *entity.IndependentDemand) (*entity.IndependentDemand, error)
	GetByCode(ctx context.Context, code int64) (*entity.IndependentDemand, error)
	// List devolve as demandas abertas, com o atendido: é a fonte do MRP.
	List(ctx context.Context) ([]*entity.IndependentDemand, error)
	ListByItem(ctx context.Context, itemCode int64) ([]*entity.IndependentDemand, error)
	ListFromDate(ctx context.Context, date time.Time) ([]*entity.IndependentDemand, error)
	Delete(ctx context.Context, code int64) error
}

// AcompanhamentoRepository é o lado de gestão da demanda: numeração,
// encerramento, lista com filtros, ordens vinculadas e as vendas que a
// consomem.
type AcompanhamentoRepository interface {
	NextCode(ctx context.Context) (int64, error)
	// TravarNumeracao serializa a numeração da empresa até o fim da transação
	// (sem transação, não trava).
	TravarNumeracao(ctx context.Context) error
	// Encerrar muda a demanda aberta para ENCERRADA ou CANCELADA; devolve
	// falso se ela não estava aberta.
	Encerrar(ctx context.Context, code int64, situacao, motivo string, por uuid.UUID) (bool, error)
	Buscar(ctx context.Context, f entity.Filtro) ([]*entity.IndependentDemand, error)
	Ordens(ctx context.Context, demandID int64) ([]entity.OrdemDaDemanda, error)
	// Cobertura: por demanda (id), quanto as ordens ligadas cobrem.
	Cobertura(ctx context.Context, demandIDs []int64) (map[int64]decimal.Decimal, error)
	// TravarDemanda trava a demanda até o fim da transação.
	TravarDemanda(ctx context.Context, code int64) error
	// AlmoxarifadoDaEmpresa diz se o almoxarifado é da empresa da sessão.
	AlmoxarifadoDaEmpresa(ctx context.Context, id int64) (bool, error)
	// Vendas devolve a quantidade vendida (sem a cancelada) dos itens, por
	// linha de pedido, para o consumo da demanda pela venda.
	Vendas(ctx context.Context, itens []int64) ([]entity.Venda, error)
	// ClienteDaEmpresa diz se o cliente existe na empresa da sessão.
	ClienteDaEmpresa(ctx context.Context, code int64) (bool, error)
	// IDDoCentro traduz o código do centro de custo (o que a demanda guarda)
	// para o id (o que a OF guarda), na empresa da sessão.
	IDDoCentro(ctx context.Context, code int64) (*int64, error)
}
