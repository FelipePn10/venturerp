package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/entity"
	"github.com/shopspring/decimal"
)

type Filtro struct {
	Busca         string
	UF            string
	Modal         string
	SomenteAtivas bool
}

type Ocorrencia struct {
	ID             int64
	EnterpriseID   int64
	CarrierID      int64
	OccurrenceDate time.Time
	OccurrenceType string
	SalesOrderCode *int64
	DelayDays      int16
	CostImpact     decimal.Decimal
	Description    *string
	CreatedAt      time.Time
}

// Desempenho é a leitura das ocorrências: é o número que responde "esta
// transportadora atrasa?" sem depender da memória de quem despacha.
type Desempenho struct {
	Ocorrencias int
	AtrasoMedio decimal.Decimal
	CustoTotal  decimal.Decimal
	UltimaData  *time.Time
}

type Repository interface {
	Listar(ctx context.Context, f Filtro) ([]*entity.Transportadora, error)
	Obter(ctx context.Context, id int64) (*entity.Transportadora, error)
	ObterPorFornecedor(ctx context.Context, supplierCode int64) (*entity.Transportadora, error)
	Salvar(ctx context.Context, t *entity.Transportadora) (*entity.Transportadora, error)
	DefinirSituacao(ctx context.Context, id int64, ativa bool) error
	RegistrarOcorrencia(ctx context.Context, o *Ocorrencia) (*Ocorrencia, error)
	ListarOcorrencias(ctx context.Context, carrierID int64, limite int) ([]*Ocorrencia, error)
	Desempenho(ctx context.Context, carrierID int64, desde time.Time) (Desempenho, error)
	// FornecedorExiste evita perfil de transporte apontando para fornecedor de
	// outra empresa ou inexistente.
	FornecedorExiste(ctx context.Context, supplierCode int64) (string, string, bool, error)
}
