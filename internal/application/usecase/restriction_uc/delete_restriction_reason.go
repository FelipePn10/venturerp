package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type DeleteRestrictionReasonUseCase struct {
	Repo repository.RestrictionReasonRepository
	Auth ports.AuthService
}

func (uc *DeleteRestrictionReasonUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanDeactivateRestriction(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Delete(ctx, code)
}
