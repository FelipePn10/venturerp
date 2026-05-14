package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type GetRestrictionUseCase struct {
	Repo repository.RestrictionRepository
	Auth ports.AuthService
}

func (uc *GetRestrictionUseCase) Execute(ctx context.Context, code int64) (*entity.Restriction, error) {
	if !uc.Auth.CanGetRestriction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetByCode(ctx, code)
}
