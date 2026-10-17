package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity"
)

type CostCenterRepository interface {
	Create(ctx context.Context, cc *entity.CostCenter) (*entity.CostCenter, error)
	Update(ctx context.Context, cc *entity.CostCenter) (*entity.CostCenter, error)
	GetByCode(ctx context.Context, code int32) (*entity.CostCenter, error)
	List(ctx context.Context) ([]*entity.CostCenter, error)
	ListByType(ctx context.Context, ccType string) ([]*entity.CostCenter, error)
	Delete(ctx context.Context, code int32) error
}

// UsoCentroCusto: onde o centro está em uso e quantos registros dependem dele.
type UsoCentroCusto struct {
	Onde       string
	Quantidade int64
}

// AlteracaoCentroCusto é o que a manutenção grava (o código não muda).
type AlteracaoCentroCusto struct {
	ID          int64
	Description string
	ParentCode  *int32
	Type        string
	IsRatio     bool
	StartDate   time.Time
	EndDate     *time.Time
	IsActive    bool
}

// CostCenterManutencaoRepository: alterar, excluir e conferir o uso.
type CostCenterManutencaoRepository interface {
	ListarComInativos(ctx context.Context) ([]*entity.CostCenter, error)
	Usos(ctx context.Context, cc *entity.CostCenter) ([]UsoCentroCusto, error)
	FilhosAtivos(ctx context.Context, code int32) (int64, error)
	Alterar(ctx context.Context, a AlteracaoCentroCusto) error
	Excluir(ctx context.Context, id int64) error
}
