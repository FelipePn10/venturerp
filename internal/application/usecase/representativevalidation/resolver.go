package representativevalidation

import (
	"context"
	"fmt"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
)

type Repository interface {
	Get(context.Context, int64) (*entity.Representative, error)
}

func Validate(ctx context.Context, repo Repository, code *int64) error {
	if code == nil || *code == 0 {
		return nil
	}
	if repo == nil {
		return errorsuc.NewValidationError("a validação do representante não está configurada")
	}
	representative, err := repo.Get(ctx, *code)
	if err != nil || representative == nil {
		return errorsuc.NewValidationError(fmt.Sprintf("representante %d não encontrado na empresa autenticada", *code))
	}
	if representative.Blocked {
		message := fmt.Sprintf("o representante %d está bloqueado", *code)
		if representative.BlockReason != nil && strings.TrimSpace(*representative.BlockReason) != "" {
			message += ": " + strings.TrimSpace(*representative.BlockReason)
		}
		return errorsuc.NewValidationError(message)
	}
	if !representative.IsActive {
		return errorsuc.NewValidationError(fmt.Sprintf("o representante %d está inativo", *code))
	}
	return nil
}

func ValidateRequired(ctx context.Context, repo Repository, code int64) error {
	if code == 0 {
		return errorsuc.NewValidationError("informe um representante válido")
	}
	return Validate(ctx, repo, &code)
}
