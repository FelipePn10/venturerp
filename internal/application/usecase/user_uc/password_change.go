package user_uc

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	userentity "github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordChangeForbidden = errors.New("password change operation forbidden")
	ErrPasswordChangeInvalid   = errors.New("password change request is invalid or expired")
	ErrCurrentPasswordInvalid  = errors.New("current password is invalid")
	ErrWeakPassword            = errors.New("new password must have at least 12 characters, uppercase, lowercase, number and special character")
)

type PasswordChangeAuth interface {
	UserID(context.Context) (uuid.UUID, error)
	EnterpriseID(context.Context) (int64, error)
	IsAdmin(context.Context) bool
}

type PasswordChangeUseCase struct {
	repo repository.UserRepository
	auth PasswordChangeAuth
	now  func() time.Time
}

func NewPasswordChangeUseCase(repo repository.UserRepository, auth PasswordChangeAuth) *PasswordChangeUseCase {
	return &PasswordChangeUseCase{repo: repo, auth: auth, now: time.Now}
}

func (uc *PasswordChangeUseCase) identity(ctx context.Context) (uuid.UUID, int64, error) {
	userID, err := uc.auth.UserID(ctx)
	if err != nil {
		return uuid.Nil, 0, ErrPasswordChangeForbidden
	}
	enterpriseID, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return uuid.Nil, 0, ErrPasswordChangeForbidden
	}
	return userID, enterpriseID, nil
}

func (uc *PasswordChangeUseCase) Request(ctx context.Context) (*userentity.PasswordChangeRequest, error) {
	userID, enterpriseID, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	return uc.repo.CreatePasswordChangeRequest(ctx, userID, enterpriseID)
}

func (uc *PasswordChangeUseCase) List(ctx context.Context, status string) ([]userentity.PasswordChangeRequest, error) {
	_, enterpriseID, err := uc.identity(ctx)
	if err != nil || !uc.auth.IsAdmin(ctx) {
		return nil, ErrPasswordChangeForbidden
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "" && status != "PENDING" && status != "APPROVED" && status != "REJECTED" && status != "USED" && status != "EXPIRED" {
		return nil, ErrPasswordChangeInvalid
	}
	return uc.repo.ListPasswordChangeRequests(ctx, enterpriseID, status)
}

func (uc *PasswordChangeUseCase) Approve(ctx context.Context, requestID uuid.UUID) error {
	adminID, enterpriseID, err := uc.identity(ctx)
	if err != nil || !uc.auth.IsAdmin(ctx) {
		return ErrPasswordChangeForbidden
	}
	if err := uc.repo.ApprovePasswordChangeRequest(ctx, requestID, enterpriseID, adminID, uc.now().Add(15*time.Minute)); err != nil {
		return ErrPasswordChangeInvalid
	}
	return nil
}

func (uc *PasswordChangeUseCase) Reject(ctx context.Context, requestID uuid.UUID, reason string) error {
	adminID, enterpriseID, err := uc.identity(ctx)
	if err != nil || !uc.auth.IsAdmin(ctx) {
		return ErrPasswordChangeForbidden
	}
	if len(reason) > 500 {
		return ErrPasswordChangeInvalid
	}
	if err := uc.repo.RejectPasswordChangeRequest(ctx, requestID, enterpriseID, adminID, strings.TrimSpace(reason)); err != nil {
		return ErrPasswordChangeInvalid
	}
	return nil
}

var (
	upperPattern   = regexp.MustCompile(`[A-Z]`)
	lowerPattern   = regexp.MustCompile(`[a-z]`)
	digitPattern   = regexp.MustCompile(`[0-9]`)
	specialPattern = regexp.MustCompile(`[^A-Za-z0-9]`)
)

func validNewPassword(password string) bool {
	return len(password) >= 12 && len(password) <= 128 && upperPattern.MatchString(password) &&
		lowerPattern.MatchString(password) && digitPattern.MatchString(password) && specialPattern.MatchString(password)
}

func (uc *PasswordChangeUseCase) Complete(ctx context.Context, requestID uuid.UUID, currentPassword, newPassword string) error {
	userID, enterpriseID, err := uc.identity(ctx)
	if err != nil {
		return err
	}
	if currentPassword == newPassword {
		return ErrWeakPassword
	}
	if !validNewPassword(newPassword) {
		return ErrWeakPassword
	}
	currentHash, err := uc.repo.PasswordHash(ctx, userID, enterpriseID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(currentPassword)) != nil {
		return ErrCurrentPasswordInvalid
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	if err := uc.repo.CompletePasswordChange(ctx, requestID, userID, enterpriseID, currentHash, string(newHash)); err != nil {
		return ErrPasswordChangeInvalid
	}
	return nil
}
