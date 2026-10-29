package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

// ClassificacaoRef é a classificação de item de uma regra de política.
type ClassificacaoRef struct {
	ID        int64
	Code      string
	Descricao string
	Level     int
}

// PoliticaRepository guarda a política de planejamento por item ou
// classificação da empresa da sessão (VPLA0310).
type PoliticaRepository interface {
	Listar(ctx context.Context, f entity.FiltroPolitica) ([]*entity.PoliticaPlanejamento, error)
	Obter(ctx context.Context, id int64) (*entity.PoliticaPlanejamento, error)
	Criar(ctx context.Context, p *entity.PoliticaPlanejamento) (*entity.PoliticaPlanejamento, error)
	Alterar(ctx context.Context, p *entity.PoliticaPlanejamento) (*entity.PoliticaPlanejamento, error)
	Excluir(ctx context.Context, id int64) error
	// Classificacao devolve a classificação se ela for da empresa da sessão.
	Classificacao(ctx context.Context, id int64) (*ClassificacaoRef, error)
	// PoliticasDePlanejamento é a política efetiva de cada item com regra.
	PoliticasDePlanejamento(ctx context.Context, empresa entity.PoliticaEfetiva) (map[int64]entity.PoliticaEfetiva, error)
	// PoliticaDoItem é a política efetiva de um item (com regra ou não).
	PoliticaDoItem(ctx context.Context, item int64, empresa entity.PoliticaEfetiva) (entity.PoliticaEfetiva, error)
}
