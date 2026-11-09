package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/entity"
)

// Resolucao é o resultado de traduzir um texto recebido para o catálogo.
type Resolucao struct {
	Texto   string // como veio
	Unidade string // código no catálogo; vazio = não reconhecida
	// Como foi reconhecida: CODIGO (já é o código), SINONIMO_FORNECEDOR,
	// SINONIMO (geral da empresa) ou CODIGO_FISCAL.
	Via string
}

// UnitRepository: catálogo da empresa do contexto (tenant).
type UnitRepository interface {
	ListUnits(ctx context.Context, includeInactive bool) ([]*entity.Unit, error)
	GetUnit(ctx context.Context, code string) (*entity.Unit, error)
	CreateUnit(ctx context.Context, u *entity.Unit) (*entity.Unit, error)
	UpdateUnit(ctx context.Context, u *entity.Unit) (*entity.Unit, error)
	SetUnitActive(ctx context.Context, code string, active bool) error
	// UnitUsage conta onde a unidade está em uso (itens, estruturas,
	// conversões) — para recusar a inativação de unidade em uso.
	UnitUsage(ctx context.Context, code string) (int64, error)

	ListAliases(ctx context.Context) ([]*entity.Alias, error)
	CreateAlias(ctx context.Context, a *entity.Alias) (*entity.Alias, error)
	DeleteAlias(ctx context.Context, id int64) error
	SupplierExists(ctx context.Context, supplierCode int64) (bool, error)

	ListStandardConversions(ctx context.Context) ([]*entity.StandardConversion, error)
	UpsertStandardConversion(ctx context.Context, c *entity.StandardConversion) (*entity.StandardConversion, error)
	DeleteStandardConversion(ctx context.Context, id int64) error

	// Resolver traduz textos recebidos (ex.: uCom da NF-e) para o catálogo.
	Resolver(ctx context.Context, supplierCode *int64, textos []string) (map[string]Resolucao, error)
}
