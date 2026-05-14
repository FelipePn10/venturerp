package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type GetRestrictionsByItemUseCase struct {
	Repo repository.RestrictionRepository
	Auth ports.AuthService
}

func (uc *GetRestrictionsByItemUseCase) Execute(ctx context.Context, itemCode int64) ([]*entity.Restriction, error) {
	if !uc.Auth.CanListRestrictions(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetByItemCode(ctx, itemCode)
}
