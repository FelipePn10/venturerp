package repository

import (
	"context"
	"time"

	user "github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/google/uuid"
)

type Authorization struct {
	EnterpriseID   int64
	EnterpriseCode int64
	Role           string
	AuthVersion    int64
}

type UserRepository interface {
	Create(ctx context.Context, user *user.User, enterpriseID int64) error
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	ResolveAuthorization(ctx context.Context, userID string, enterpriseCode *int64) (Authorization, error)
	CurrentAuthorization(ctx context.Context, userID string, enterpriseID int64) (Authorization, error)
	CreatePasswordChangeRequest(ctx context.Context, userID uuid.UUID, enterpriseID int64) (*user.PasswordChangeRequest, error)
	ListPasswordChangeRequests(ctx context.Context, enterpriseID int64, status string) ([]user.PasswordChangeRequest, error)
	ApprovePasswordChangeRequest(ctx context.Context, requestID uuid.UUID, enterpriseID int64, adminID uuid.UUID, expiresAt time.Time) error
	RejectPasswordChangeRequest(ctx context.Context, requestID uuid.UUID, enterpriseID int64, adminID uuid.UUID, reason string) error
	PasswordHash(ctx context.Context, userID uuid.UUID, enterpriseID int64) (string, error)
	CompletePasswordChange(ctx context.Context, requestID, userID uuid.UUID, enterpriseID int64, expectedPasswordHash, newPasswordHash string) error
}
