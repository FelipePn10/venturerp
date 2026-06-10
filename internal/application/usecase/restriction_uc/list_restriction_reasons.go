package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type ListRestrictionReasonsUseCase struct {
	Repo repository.RestrictionReasonRepository
	Auth ports.AuthService
}

func (uc *ListRestrictionReasonsUseCase) Execute(ctx context.Context) ([]*response.RestrictionReasonResponse, error) {
	if !uc.Auth.CanListRestrictions(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toRestrictionReasonResponses(list), nil
}
