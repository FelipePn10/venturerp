package user_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"golang.org/x/crypto/bcrypt"
)

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
) (id string, role string, enterpriseID int64, authVersion int64, err error) {
	user, err := uc.Repo.FindByEmail(ctx, login.Email)
	if err != nil {
		return "", "", 0, 0, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(login.Password),
	); err != nil {
		return "", "", 0, 0, errors.New("invalid credentials")
	}
	r := user.Role
	if r == "" {
		r = "USER"
	}
	enterpriseID, err = uc.Repo.ResolveEnterprise(ctx, user.ID.String(), login.EnterpriseCode)
	if err != nil {
		return "", "", 0, 0, errors.New("invalid enterprise selection")
	}
	return user.ID.String(), r, enterpriseID, user.AuthVersion, nil
}
