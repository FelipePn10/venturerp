package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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
) (*response.RestrictionResponse, error) {
	if !uc.Auth.CanUpdateRestriction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	updated, err := uc.Repo.Update(ctx, res)
	if err != nil {
		return nil, err
	}
	return toRestrictionResponse(updated), nil
}
