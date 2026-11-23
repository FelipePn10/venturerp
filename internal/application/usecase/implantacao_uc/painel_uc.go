// Package implantacao_uc: o painel "pronto para operar" da empresa (melhoria 1).
package implantacao_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/implantacao"
)

// Contador lê do banco as contagens das verificações da empresa da sessão.
type Contador interface {
	Contagens(ctx context.Context) (map[string]implantacao.Contagem, error)
}

type PainelUseCase struct{ Repo Contador }

// Painel monta o placar da empresa da sessão.
func (uc *PainelUseCase) Painel(ctx context.Context) (implantacao.Painel, error) {
	c, err := uc.Repo.Contagens(ctx)
	if err != nil {
		return implantacao.Painel{}, err
	}
	return implantacao.Montar(c), nil
}
