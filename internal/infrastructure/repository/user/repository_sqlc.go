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
		var authorization repository.Authorization
		queryErr := r.pool.QueryRow(ctx, `SELECT e.id,e.code::bigint,ue.role,u.auth_version
			FROM user_enterprises ue JOIN enterprise e ON e.id=ue.enterprise_id JOIN users u ON u.id=ue.user_id
			WHERE ue.user_id=$1 AND e.code=$2 AND u.is_active`, id, *enterpriseCode).Scan(
			&authorization.EnterpriseID, &authorization.EnterpriseCode, &authorization.Role, &authorization.AuthVersion)
		return authorization, queryErr
	}
	var authorization repository.Authorization
	queryErr := r.pool.QueryRow(ctx, `SELECT MIN(ue.enterprise_id)::bigint,MIN(e.code)::bigint,MIN(ue.role)::text,MIN(u.auth_version)::bigint
		FROM user_enterprises ue JOIN users u ON u.id=ue.user_id JOIN enterprise e ON e.id=ue.enterprise_id
		WHERE ue.user_id=$1 AND u.is_active HAVING COUNT(*)=1`, id).Scan(
		&authorization.EnterpriseID, &authorization.EnterpriseCode, &authorization.Role, &authorization.AuthVersion)
	return authorization, queryErr
}

func (r *repositoryUserSQLC) CurrentAuthorization(ctx context.Context, userID string, enterpriseID int64) (repository.Authorization, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return repository.Authorization{}, err
	}
	var authorization repository.Authorization
	err = r.pool.QueryRow(ctx, `SELECT ue.enterprise_id,e.code::bigint,ue.role,u.auth_version
		FROM user_enterprises ue JOIN users u ON u.id=ue.user_id JOIN enterprise e ON e.id=ue.enterprise_id
		WHERE ue.user_id=$1 AND ue.enterprise_id=$2 AND u.is_active`, id, enterpriseID).Scan(
		&authorization.EnterpriseID, &authorization.EnterpriseCode, &authorization.Role, &authorization.AuthVersion)
	return authorization, err
}

func (r *repositoryUserSQLC) FindByEmail(
	ctx context.Context,
	email string,
) (*entity.User, error) {

	var u entity.User
	err := r.pool.QueryRow(ctx, `SELECT id,name,email,password,role,auth_version FROM users WHERE email=$1 AND is_active`, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.AuthVersion)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
