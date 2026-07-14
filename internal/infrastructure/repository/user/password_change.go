package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	userentity "github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var errPasswordChangeNotAllowed = errors.New("password change request is not eligible")

func (r *repositoryUserSQLC) CreatePasswordChangeRequest(ctx context.Context, userID uuid.UUID, enterpriseID int64) (*userentity.PasswordChangeRequest, error) {
	const query = `INSERT INTO password_change_requests (enterprise_id, user_id, requested_by)
		SELECT $1, $2, $2 WHERE EXISTS (
			SELECT 1 FROM user_enterprises WHERE enterprise_id = $1 AND user_id = $2
		) RETURNING id, user_id, enterprise_id, status, expires_at, rejection_reason, created_at`
	var request userentity.PasswordChangeRequest
	err := r.pool.QueryRow(ctx, query, enterpriseID, userID).Scan(
		&request.ID, &request.UserID, &request.EnterpriseID, &request.Status,
		&request.ExpiresAt, &request.RejectionReason, &request.CreatedAt,
	)
	return &request, err
}

func (r *repositoryUserSQLC) ListPasswordChangeRequests(ctx context.Context, enterpriseID int64, status string) ([]userentity.PasswordChangeRequest, error) {
	const query = `SELECT p.id, p.user_id, p.enterprise_id, u.name, u.email, p.status,
		p.expires_at, p.rejection_reason, p.created_at
		FROM password_change_requests p JOIN users u ON u.id = p.user_id
		WHERE p.enterprise_id = $1 AND ($2 = '' OR p.status = $2)
		ORDER BY p.created_at DESC LIMIT 200`
	rows, err := r.pool.Query(ctx, query, enterpriseID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	requests := make([]userentity.PasswordChangeRequest, 0)
	for rows.Next() {
		var request userentity.PasswordChangeRequest
		if err := rows.Scan(&request.ID, &request.UserID, &request.EnterpriseID, &request.UserName,
			&request.UserEmail, &request.Status, &request.ExpiresAt, &request.RejectionReason, &request.CreatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

func (r *repositoryUserSQLC) ApprovePasswordChangeRequest(ctx context.Context, requestID uuid.UUID, enterpriseID int64, adminID uuid.UUID, expiresAt time.Time) error {
	const query = `UPDATE password_change_requests SET status='APPROVED', approved_by=$3,
		approved_at=now(), expires_at=$4, updated_at=now()
		WHERE id=$1 AND enterprise_id=$2 AND status='PENDING' AND user_id<>$3`
	result, err := r.pool.Exec(ctx, query, requestID, enterpriseID, adminID, expiresAt)
	if err == nil && result.RowsAffected() != 1 {
		return errPasswordChangeNotAllowed
	}
	return err
}

func (r *repositoryUserSQLC) RejectPasswordChangeRequest(ctx context.Context, requestID uuid.UUID, enterpriseID int64, adminID uuid.UUID, reason string) error {
	const query = `UPDATE password_change_requests SET status='REJECTED', rejected_by=$3,
		rejected_at=now(), rejection_reason=NULLIF($4,''), updated_at=now()
		WHERE id=$1 AND enterprise_id=$2 AND status='PENDING'`
	result, err := r.pool.Exec(ctx, query, requestID, enterpriseID, adminID, reason)
	if err == nil && result.RowsAffected() != 1 {
		return errPasswordChangeNotAllowed
	}
	return err
}

func (r *repositoryUserSQLC) PasswordHash(ctx context.Context, userID uuid.UUID, enterpriseID int64) (string, error) {
	const query = `SELECT u.password FROM users u JOIN user_enterprises ue ON ue.user_id=u.id
		WHERE u.id=$1 AND ue.enterprise_id=$2`
	var hash string
	err := r.pool.QueryRow(ctx, query, userID, enterpriseID).Scan(&hash)
	return hash, err
}

func (r *repositoryUserSQLC) CompletePasswordChange(ctx context.Context, requestID, userID uuid.UUID, enterpriseID int64, expectedPasswordHash, newPasswordHash string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var storedHash string
	err = tx.QueryRow(ctx, `SELECT u.password FROM users u JOIN user_enterprises ue ON ue.user_id=u.id
		WHERE u.id=$1 AND ue.enterprise_id=$2 FOR UPDATE`, userID, enterpriseID).Scan(&storedHash)
	if err != nil {
		return err
	}
	if storedHash != expectedPasswordHash {
		return errPasswordChangeNotAllowed
	}
	result, err := tx.Exec(ctx, `UPDATE password_change_requests SET status='USED', used_at=now(), updated_at=now()
		WHERE id=$1 AND user_id=$2 AND enterprise_id=$3 AND status='APPROVED' AND expires_at>now()`, requestID, userID, enterpriseID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		expired, expireErr := tx.Exec(ctx, `UPDATE password_change_requests SET status='EXPIRED', updated_at=now()
			WHERE id=$1 AND enterprise_id=$2 AND status='APPROVED' AND expires_at<=now()`, requestID, enterpriseID)
		if expireErr == nil && expired.RowsAffected() == 1 {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return commitErr
			}
		}
		return errPasswordChangeNotAllowed
	}
	result, err = tx.Exec(ctx, `UPDATE users SET password=$2, auth_version=auth_version+1, updated_at=now() WHERE id=$1`, userID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if result.RowsAffected() != 1 {
		return errors.New("update password affected no user")
	}
	return tx.Commit(ctx)
}

func (r *repositoryUserSQLC) AuthVersion(ctx context.Context, userID string) (int64, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return 0, err
	}
	var version int64
	err = r.pool.QueryRow(ctx, `SELECT auth_version FROM users WHERE id=$1`, id).Scan(&version)
	return version, err
}
