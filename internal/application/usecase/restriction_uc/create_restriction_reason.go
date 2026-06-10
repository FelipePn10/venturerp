package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type CreateRestrictionReasonUseCase struct {
	Repo repository.RestrictionReasonRepository
	Auth ports.AuthService
}

func (uc *CreateRestrictionReasonUseCase) Execute(
	ctx context.Context,
	description, situation string,
) (*response.RestrictionReasonResponse, error) {
	if !uc.Auth.CanCreateRestriction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if situation == "" {
		situation = "ACTIVE"
	}
	created, err := uc.Repo.Create(ctx, &entity.RestrictionReason{
		Description: description,
		Situation:   situation,
	})
	if err != nil {
		return nil, err
	}
	return toRestrictionReasonResponse(created), nil
}
