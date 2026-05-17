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
) (string, error) {
	user, err := uc.Repo.FindByEmail(ctx, login.Email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(login.Password),
	); err != nil {
		return "", errors.New("invalid credentials")
	}
	return user.ID.String(), nil
}
