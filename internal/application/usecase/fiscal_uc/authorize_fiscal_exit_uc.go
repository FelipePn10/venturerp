package fiscal_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/google/uuid"
)

type AuthorizeFiscalExitUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *AuthorizeFiscalExitUseCase) Execute(ctx context.Context, id int64) (*entity.FiscalExit, error) {
	if !uc.Auth.CanAuthorizeFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	chaveAcesso := fmt.Sprintf("NFe%013d%02d%06d%s",
		id,
		time.Now().Year()%100,
		time.Now().Unix()%1000000,
		uuid.New().String()[:8],
	)
	protocolo := fmt.Sprintf("%d%s", time.Now().Unix(), uuid.New().String()[:8])
	focusRef := fmt.Sprintf("ref_%s", uuid.New().String()[:12])

	return uc.Repo.UpdateExitAuthorization(ctx, id, chaveAcesso, protocolo, focusRef)
}
