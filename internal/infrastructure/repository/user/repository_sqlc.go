package user

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryUserSQLC) Create(
	ctx context.Context,
	user *entity.User,
) error {

	return r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:       pgutil.ToPgUUID(user.ID),
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
	})
}

func (r *repositoryUserSQLC) FindByEmail(
	ctx context.Context,
	email string,
) (*entity.User, error) {

	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &entity.User{
		ID:       pgutil.FromPgUUID(u.ID),
		Name:     u.Name,
		Email:    u.Email,
		Password: u.Password,
		Role:     u.Role,
	}, nil
}
