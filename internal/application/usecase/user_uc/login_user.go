package user_uc

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"golang.org/x/crypto/bcrypt"
)

var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-used-only-for-timing"), bcrypt.DefaultCost)
var ErrIdentitySync = errors.New("training identity synchronization failed")

// EmpresaDoUsuario: empresa a que o usuário tem acesso (para escolher no login).
type EmpresaDoUsuario = repository.EmpresaDoUsuario

// ErrEmpresaObrigatoria: senha conferida, mas o usuário tem acesso a mais de
// uma empresa e não disse em qual entra. Leva a lista para a tela escolher.
type ErrEmpresaObrigatoria struct{ Empresas []EmpresaDoUsuario }

func (e *ErrEmpresaObrigatoria) Error() string { return "escolha a empresa" }

type empresasDoUsuario interface {
	EmpresasDoUsuario(ctx context.Context, userID string) ([]EmpresaDoUsuario, error)
}

type IdentityMirror interface {
	SyncIdentity(ctx context.Context, userID string, enterpriseID int64) error
}

type LoginUserUseCase struct {
	Repo   repository.UserRepository
	Mirror IdentityMirror
}

func NewLoginUserUseCase(
	repo repository.UserRepository,
) *LoginUserUseCase {
	return &LoginUserUseCase{Repo: repo}
}

func (uc *LoginUserUseCase) Execute(
	ctx context.Context,
	login request.LoginUserDTO,
) (id string, role string, name string, email string, enterpriseID int64, authVersion int64, err error) {
	user, err := uc.Repo.FindByEmail(ctx, login.Email)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(login.Password))
		return "", "", "", "", 0, 0, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(login.Password),
	); err != nil {
		return "", "", "", "", 0, 0, errors.New("invalid credentials")
	}
	authorization, err := uc.Repo.ResolveAuthorization(ctx, user.ID.String(), login.EnterpriseCode)
	if err != nil {
		// Com acesso a mais de uma empresa e sem a escolha, a tela precisa
		// perguntar em qual entrar (antes respondia "credenciais inválidas" e o
		// usuário não entrava de jeito nenhum).
		if login.EnterpriseCode == nil {
			if r, ok := uc.Repo.(empresasDoUsuario); ok {
				if lista, errLista := r.EmpresasDoUsuario(ctx, user.ID.String()); errLista == nil && len(lista) > 1 {
					return "", "", "", "", 0, 0, &ErrEmpresaObrigatoria{Empresas: lista}
				}
			}
		}
		return "", "", "", "", 0, 0, errors.New("invalid enterprise selection")
	}
	if uc.Mirror != nil {
		if err := uc.Mirror.SyncIdentity(ctx, user.ID.String(), authorization.EnterpriseID); err != nil {
			return "", "", "", "", 0, 0, fmt.Errorf("%w: %v", ErrIdentitySync, err)
		}
	}
	return user.ID.String(), authorization.Role, user.Name, user.Email,
		authorization.EnterpriseID, authorization.AuthVersion, nil
}
