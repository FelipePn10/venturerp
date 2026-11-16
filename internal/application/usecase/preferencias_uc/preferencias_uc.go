// Package preferencias_uc guarda as preferências de exibição do usuário.
package preferencias_uc

import (
	"context"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

const (
	NomeRazaoSocial = "RAZAO_SOCIAL"
	NomeFantasia    = "FANTASIA"
)

// Preferencias do usuário. CustomerNameDisplay: o que as buscas e campos de
// cliente mostram como nome (a busca encontra pelos dois).
type Preferencias struct {
	CustomerNameDisplay string `json:"customer_name_display"`
}

func Padrao() Preferencias { return Preferencias{CustomerNameDisplay: NomeRazaoSocial} }

type Repository interface {
	// Get devolve ok=false quando o usuário nunca gravou preferência.
	Get(ctx context.Context, userID uuid.UUID) (p Preferencias, ok bool, err error)
	Save(ctx context.Context, userID uuid.UUID, p Preferencias) error
}

type UseCase struct{ Repo Repository }

func usuario(ctx context.Context) (uuid.UUID, error) {
	u, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok || u == nil {
		return uuid.Nil, errorsuc.ErrUnauthorized
	}
	id, err := uuid.Parse(u.ID)
	if err != nil {
		return uuid.Nil, errorsuc.ErrUnauthorized
	}
	return id, nil
}

func (uc *UseCase) Get(ctx context.Context) (Preferencias, error) {
	id, err := usuario(ctx)
	if err != nil {
		return Preferencias{}, err
	}
	p, ok, err := uc.Repo.Get(ctx, id)
	if err != nil || !ok {
		return Padrao(), err
	}
	return p, nil
}

func (uc *UseCase) Save(ctx context.Context, p Preferencias) (Preferencias, error) {
	id, err := usuario(ctx)
	if err != nil {
		return Preferencias{}, err
	}
	if p.CustomerNameDisplay != NomeRazaoSocial && p.CustomerNameDisplay != NomeFantasia {
		return Preferencias{}, errorsuc.NewValidationError("escolha RAZAO_SOCIAL ou FANTASIA para o nome do cliente")
	}
	if err := uc.Repo.Save(ctx, id, p); err != nil {
		return Preferencias{}, err
	}
	return p, nil
}
