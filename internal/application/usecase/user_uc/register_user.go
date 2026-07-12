package user_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	userentity "github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserUseCase struct {
	Repo repository.UserRepository
}

func NewRegisterUserUseCase(
	repo repository.UserRepository,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{Repo: repo}
}

func (uc *RegisterUserUseCase) Execute(
	ctx context.Context,
	dto request.RegisterUserDTO,
) error {
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

	if dto.EnterpriseCode <= 0 {
		return userentity.ErrInvalidEnterprise
	}
	return uc.Repo.Create(ctx, user, dto.EnterpriseCode)
}
