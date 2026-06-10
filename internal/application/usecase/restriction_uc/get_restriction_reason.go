package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type GetRestrictionReasonUseCase struct {
	Repo repository.RestrictionReasonRepository
	Auth ports.AuthService
}

func (uc *GetRestrictionReasonUseCase) Execute(ctx context.Context, code int64) (*response.RestrictionReasonResponse, error) {
	if !uc.Auth.CanGetRestriction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	r, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toRestrictionReasonResponse(r), nil
}
