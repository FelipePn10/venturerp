// Package parametros_comerciais_uc: parâmetros comerciais da empresa (Fase 5.3).
package parametros_comerciais_uc

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/parametroscomerciais"
)

type UseCase struct {
	Repo parametroscomerciais.Repository
	Auth ports.AuthService
}

func (uc *UseCase) Obter(ctx context.Context) (parametroscomerciais.Parametros, error) {
	return uc.Repo.Obter(ctx)
}

// Salvar: desconto_max_pct nulo = sem alçada; senão entre 0 e 100.
func (uc *UseCase) Salvar(ctx context.Context, p parametroscomerciais.Parametros) (parametroscomerciais.Parametros, error) {
	por, err := uc.Auth.UserID(ctx)
	if err != nil || por == uuid.Nil {
		return p, errorsuc.ErrUnauthorized
	}
	if d := p.DescontoMaxPct; d != nil {
		if d.IsNegative() || d.GreaterThan(decimal.NewFromInt(100)) {
			return p, errorsuc.NewValidationError("a alçada de desconto fica entre 0 e 100%")
		}
		r := d.Round(2)
		p.DescontoMaxPct = &r
	}
	if err := uc.Repo.Salvar(ctx, p, por); err != nil {
		return p, err
	}
	return uc.Repo.Obter(ctx)
}
