package user

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/google/uuid"
)

func (r *repositoryUserSQLC) Create(
	ctx context.Context,
	user *entity.User,
	enterpriseCode int64,
) error {

	if err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:             pgutil.ToPgUUID(user.ID),
		Name:           user.Name,
		Email:          user.Email,
		Password:       user.Password,
		EnterpriseCode: int32(enterpriseCode),
	}); err != nil {
		return err
	}
	return nil
}

func (r *repositoryUserSQLC) ResolveEnterprise(ctx context.Context, userID string, enterpriseCode *int64) (int64, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return 0, err
	}
	if enterpriseCode != nil {
		return r.q.GetUserEnterpriseByCode(ctx, sqlc.GetUserEnterpriseByCodeParams{UserID: pgutil.ToPgUUID(id), Code: int32(*enterpriseCode)})
	}
	return r.q.GetOnlyUserEnterprise(ctx, pgutil.ToPgUUID(id))
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
		ID:          pgutil.FromPgUUID(u.ID),
		Name:        u.Name,
		Email:       u.Email,
		Password:    u.Password,
		Role:        u.Role,
		AuthVersion: u.AuthVersion,
	}, nil
}
