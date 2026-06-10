package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type GetRestrictionsByCustomerUseCase struct {
	Repo repository.RestrictionRepository
	Auth ports.AuthService
}

func (uc *GetRestrictionsByCustomerUseCase) Execute(
	ctx context.Context,
	customerCode int64,
) ([]*response.RestrictionResponse, error) {
	if !uc.Auth.CanListRestrictions(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.GetByCustomerCode(ctx, customerCode)
	if err != nil {
		return nil, err
	}
	return toRestrictionResponses(list), nil
}
