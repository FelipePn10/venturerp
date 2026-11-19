package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/project/entity"
)

// Filtro da lista de projetos. Busca vale para código, nome e cliente.
type Filtro struct {
	Busca  string
	Status string
}

// ProjectRepository: projetos da empresa do contexto (tenant).
type ProjectRepository interface {
	List(ctx context.Context, f Filtro) ([]*entity.Project, error)
	GetByCode(ctx context.Context, code string) (*entity.Project, error)
	GetByID(ctx context.Context, id int64) (*entity.Project, error)
	Create(ctx context.Context, p *entity.Project) (*entity.Project, error)
	Update(ctx context.Context, p *entity.Project) (*entity.Project, error)
}
