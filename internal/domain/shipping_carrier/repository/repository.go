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
	ShipmentCode   *int64 // código do romaneio (único por empresa); gravado como shipments.id
	DelayDays      int16
	CostImpact     decimal.Decimal
	Description    *string
	CreatedAt      time.Time
}

// Desempenho é a leitura das ocorrências: é o número que responde "esta
// transportadora atrasa?" sem depender da memória de quem despacha.
type Desempenho struct {
	Ocorrencias int
	Falhas      int
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
	// Excluir apaga capa, frota e regiões. Quem chama confere o Uso antes; as
	// FKs dos documentos são a última trava.
	Excluir(ctx context.Context, id int64) error
	// Uso conta onde a transportadora está gravada (documentos e clientes).
	Uso(ctx context.Context, id int64) ([]Uso, error)
	// PlacasEmUso são as placas desta transportadora em carga ou romaneio ainda
	// não expedidos: tirar o veículo da frota deixaria o documento órfão.
	PlacasEmUso(ctx context.Context, id int64) ([]string, error)

	ListarTiposVeiculo(ctx context.Context) ([]*TipoVeiculo, error)
	SalvarTipoVeiculo(ctx context.Context, t *TipoVeiculo) (*TipoVeiculo, error)
	ExcluirTipoVeiculo(ctx context.Context, id int64) error
	ListarTiposOcorrencia(ctx context.Context) ([]*TipoOcorrencia, error)
	SalvarTipoOcorrencia(ctx context.Context, t *TipoOcorrencia) (*TipoOcorrencia, error)
	ExcluirTipoOcorrencia(ctx context.Context, id int64) error

	// HojeDaEmpresa é a data de hoje no fuso da empresa (meia-noite UTC, como
	// as colunas DATE chegam): perto da meia-noite o relógio do servidor já
	// está em outro dia, e o RNTRC "venceria" horas antes.
	HojeDaEmpresa(ctx context.Context) (time.Time, error)

	// FornecedorExiste evita perfil de transporte apontando para fornecedor de
	// outra empresa ou inexistente.
	FornecedorExiste(ctx context.Context, supplierCode int64) (string, string, bool, error)
}

// Uso é um tipo de documento que referencia a transportadora: quantos no total
// e quantos ainda em aberto (não expedidos, não recebidos, não cancelados).
type Uso struct {
	Documento string
	Total     int
	Abertos   int
}

type TipoVeiculo struct {
	ID          int64
	Code        string
	Description string
	WheelType   *string
	BodyType    *string
	Axles       *int16
	CapacityKg  decimal.Decimal
	CapacityM3  decimal.Decimal
	IsActive    bool
	EmUso       int
}

type TipoOcorrencia struct {
	ID              int64
	Code            string
	Description     string
	CountsAsFailure bool
	IsActive        bool
	EmUso           int
}
