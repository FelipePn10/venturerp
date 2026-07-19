package user_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"golang.org/x/crypto/bcrypt"
)

var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-used-only-for-timing"), bcrypt.DefaultCost)

type LoginUserUseCase struct {
	Repo repository.UserRepository
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
		return "", "", "", "", 0, 0, errors.New("invalid enterprise selection")
	}
	return user.ID.String(), authorization.Role, user.Name, user.Email,
		authorization.EnterpriseID, authorization.AuthVersion, nil
}
