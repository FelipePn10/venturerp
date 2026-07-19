package user

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/google/uuid"
)

func (r *repositoryUserSQLC) Create(
	ctx context.Context,
	user *entity.User,
	enterpriseID int64,
) error {

	if err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           pgutil.ToPgUUID(user.ID),
		Name:         user.Name,
		Email:        user.Email,
		Password:     user.Password,
		EnterpriseID: enterpriseID,
	}); err != nil {
		return err
	}
	return nil
}

func (r *repositoryUserSQLC) ResolveAuthorization(ctx context.Context, userID string, enterpriseCode *int64) (repository.Authorization, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return repository.Authorization{}, err
	}
	if enterpriseCode != nil {
		row, queryErr := r.q.GetUserAuthorizationByEnterpriseCode(ctx, sqlc.GetUserAuthorizationByEnterpriseCodeParams{
			UserID: pgutil.ToPgUUID(id), Code: int32(*enterpriseCode),
		})
		return repository.Authorization{EnterpriseID: row.EnterpriseID, EnterpriseCode: row.EnterpriseCode, Role: row.Role, AuthVersion: row.AuthVersion}, queryErr
	}
	row, queryErr := r.q.GetOnlyUserAuthorization(ctx, pgutil.ToPgUUID(id))
	return repository.Authorization{EnterpriseID: row.EnterpriseID, EnterpriseCode: row.EnterpriseCode, Role: row.Role, AuthVersion: row.AuthVersion}, queryErr
}

func (r *repositoryUserSQLC) CurrentAuthorization(ctx context.Context, userID string, enterpriseID int64) (repository.Authorization, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return repository.Authorization{}, err
	}
	row, err := r.q.GetCurrentUserAuthorization(ctx, sqlc.GetCurrentUserAuthorizationParams{
		UserID: pgutil.ToPgUUID(id), EnterpriseID: enterpriseID,
	})
	return repository.Authorization{EnterpriseID: row.EnterpriseID, EnterpriseCode: row.EnterpriseCode, Role: row.Role, AuthVersion: row.AuthVersion}, err
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
