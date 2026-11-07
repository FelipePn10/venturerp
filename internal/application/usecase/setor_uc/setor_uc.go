package setor_uc

import (
	"context"

	"github.com/google/uuid"

	dom "github.com/FelipePn10/panossoerp/internal/domain/setor"
)

// UseCase: setores, pessoas e responsáveis por grupo do PDM.
type UseCase struct {
	Repo dom.Repository
	// Usuario devolve quem está gravando, para o histórico do responsável.
	Usuario func(ctx context.Context) *uuid.UUID
}

func (uc *UseCase) Setores(ctx context.Context) ([]dom.Setor, error) { return uc.Repo.Setores(ctx) }

func (uc *UseCase) SalvarSetor(ctx context.Context, s dom.Setor) error {
	if err := s.Conferir(); err != nil {
		return err
	}
	return uc.Repo.SalvarSetor(ctx, s)
}

func (uc *UseCase) DefinirMembros(ctx context.Context, codigo string, usuarios []uuid.UUID) error {
	return uc.Repo.DefinirMembros(ctx, codigo, usuarios)
}

func (uc *UseCase) ResponsaveisDoGrupo(ctx context.Context, grupo int) (*dom.Responsaveis, error) {
	return uc.Repo.ResponsaveisDoGrupo(ctx, grupo)
}

func (uc *UseCase) SalvarResponsaveis(ctx context.Context, r dom.Responsaveis) error {
	var por *uuid.UUID
	if uc.Usuario != nil {
		por = uc.Usuario(ctx)
	}
	return uc.Repo.SalvarResponsaveis(ctx, r, por)
}

func (uc *UseCase) ResponsaveisDoItem(ctx context.Context, item string) (*dom.Responsaveis, error) {
	return uc.Repo.ResponsaveisDoItem(ctx, item)
}
