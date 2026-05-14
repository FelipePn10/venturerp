package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type UpdateRestrictionUseCase struct {
	Repo repository.RestrictionRepository
	Auth ports.AuthService
}

func (uc *UpdateRestrictionUseCase) Execute(
	ctx context.Context,
	res *entity.Restriction,
) (*entity.Restriction, error) {
	if !uc.Auth.CanUpdateRestriction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.Update(ctx, res)
}
