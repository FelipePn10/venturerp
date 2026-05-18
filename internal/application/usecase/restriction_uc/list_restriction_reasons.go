package restriction_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type ListRestrictionReasonsUseCase struct {
	Repo repository.RestrictionReasonRepository
	Auth ports.AuthService
}

func (uc *ListRestrictionReasonsUseCase) Execute(ctx context.Context) ([]*entity.RestrictionReason, error) {
	if !uc.Auth.CanListRestrictions(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}
