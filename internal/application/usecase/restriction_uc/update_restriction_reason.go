package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type UpdateRestrictionReasonUseCase struct {
	Repo repository.RestrictionReasonRepository
	Auth ports.AuthService
}

func (uc *UpdateRestrictionReasonUseCase) Execute(
	ctx context.Context,
	r *entity.RestrictionReason,
) (*entity.RestrictionReason, error) {
	if !uc.Auth.CanUpdateRestriction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.Update(ctx, r)
}
