package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type DeactivateRestrictionUseCase struct {
	Repo repository.RestrictionRepository
	Auth ports.AuthService
}

func (uc *DeactivateRestrictionUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanDeactivateRestriction(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Deactivate(ctx, code)
}
