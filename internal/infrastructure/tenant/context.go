package tenant

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

var ErrMissingEnterprise = errors.New("enterprise not selected in authenticated context")

func ID(ctx context.Context) (int64, error) {
	user, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok || user == nil || user.EnterpriseID <= 0 {
		return 0, ErrMissingEnterprise
	}
	return user.EnterpriseID, nil
}

func IDPtr(ctx context.Context) (*int64, error) {
	id, err := ID(ctx)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
