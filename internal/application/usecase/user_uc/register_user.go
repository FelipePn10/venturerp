package user_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	userentity "github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrRegisterUserForbidden = errors.New("user registration is restricted to the authenticated enterprise")

type RegisterUserAuth interface {
	EnterpriseID(context.Context) (int64, error)
}

type RegisterUserUseCase struct {
	Repo repository.UserRepository
	Auth RegisterUserAuth
}

func NewRegisterUserUseCase(
	repo repository.UserRepository,
	auth RegisterUserAuth,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{Repo: repo, Auth: auth}
}

func (uc *RegisterUserUseCase) Execute(
	ctx context.Context,
	dto request.RegisterUserDTO,
) error {
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil || enterpriseID <= 0 {
		return ErrRegisterUserForbidden
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user, err := userentity.NewUser(
		uuid.New(),
		dto.Name,
		dto.Email,
		string(hash),
	)
	if err != nil {
		return err
	}

	return uc.Repo.Create(ctx, user, enterpriseID)
}
