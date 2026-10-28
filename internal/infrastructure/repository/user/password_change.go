package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	userentity "github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var errPasswordChangeNotAllowed = errors.New("este pedido de troca de senha não pode ser usado")

// passwordChangeSelect lê a solicitação com a situação efetiva: aprovação
// vencida e não usada aparece como EXPIRED mesmo antes de alguém tentar usá-la.
const passwordChangeSelect = `SELECT p.id, p.user_id, p.enterprise_id, u.name, u.email,
		CASE WHEN p.status = 'APPROVED' AND p.expires_at <= now() THEN 'EXPIRED' ELSE p.status END,
		p.reason, p.expires_at, ap.name, p.approved_at, rj.name, p.rejected_at, p.rejection_reason,
		p.used_at, p.created_at
	FROM password_change_requests p
	JOIN users u ON u.id = p.user_id
	LEFT JOIN users ap ON ap.id = p.approved_by
	LEFT JOIN users rj ON rj.id = p.rejected_by`

func scanPasswordChange(row pgx.Row) (userentity.PasswordChangeRequest, error) {
	var p userentity.PasswordChangeRequest
	err := row.Scan(&p.ID, &p.UserID, &p.EnterpriseID, &p.UserName, &p.UserEmail, &p.Status,
		&p.Reason, &p.ExpiresAt, &p.ApprovedByName, &p.ApprovedAt, &p.RejectedByName, &p.RejectedAt,
		&p.RejectionReason, &p.UsedAt, &p.CreatedAt)
	return p, err
}

func (r *repositoryUserSQLC) queryPasswordChanges(ctx context.Context, where string, args ...any) ([]userentity.PasswordChangeRequest, error) {
	rows, err := r.pool.Query(ctx, passwordChangeSelect+" "+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	requests := make([]userentity.PasswordChangeRequest, 0)
	for rows.Next() {
		request, err := scanPasswordChange(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

// CreatePasswordChangeRequest abre a solicitação. Antes encerra a aprovação
// vencida do próprio usuário, que de outro modo bloquearia o pedido novo pelo
// índice de uma solicitação ativa por usuário.
func (r *repositoryUserSQLC) CreatePasswordChangeRequest(ctx context.Context, userID uuid.UUID, enterpriseID int64, reason string) (*userentity.PasswordChangeRequest, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var member bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_enterprises WHERE enterprise_id = $1 AND user_id = $2)`,
		enterpriseID, userID).Scan(&member); err != nil {
		return nil, err
	}
	if !member {
		return nil, userentity.ErrPasswordChangeNotMember
	}
	if _, err := tx.Exec(ctx, `UPDATE password_change_requests SET status = 'EXPIRED', updated_at = now()
		WHERE enterprise_id = $1 AND user_id = $2 AND status = 'APPROVED' AND expires_at <= now()`, enterpriseID, userID); err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO password_change_requests (enterprise_id, user_id, requested_by, reason)
		VALUES ($1, $2, $2, NULLIF($3, '')) RETURNING id`, enterpriseID, userID, reason).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, userentity.ErrPasswordChangeActive
	}
	if err != nil {
		return nil, err
	}
	request, err := scanPasswordChange(tx.QueryRow(ctx, passwordChangeSelect+" WHERE p.id = $1", id))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *repositoryUserSQLC) ListPasswordChangeRequests(ctx context.Context, enterpriseID int64, status string) ([]userentity.PasswordChangeRequest, error) {
	return r.queryPasswordChanges(ctx, `WHERE p.enterprise_id = $1
		AND ($2 = '' OR (CASE WHEN p.status = 'APPROVED' AND p.expires_at <= now() THEN 'EXPIRED' ELSE p.status END) = $2)
		ORDER BY p.created_at DESC LIMIT 200`, enterpriseID, status)
}

// ListUserPasswordChangeRequests: as solicitações do próprio usuário, da mais
// recente para a mais antiga.
func (r *repositoryUserSQLC) ListUserPasswordChangeRequests(ctx context.Context, userID uuid.UUID, enterpriseID int64) ([]userentity.PasswordChangeRequest, error) {
	return r.queryPasswordChanges(ctx, `WHERE p.enterprise_id = $1 AND p.user_id = $2
		ORDER BY p.created_at DESC LIMIT 20`, enterpriseID, userID)
}

func (r *repositoryUserSQLC) GetPasswordChangeRequest(ctx context.Context, requestID uuid.UUID, enterpriseID int64) (*userentity.PasswordChangeRequest, error) {
	request, err := scanPasswordChange(r.pool.QueryRow(ctx, passwordChangeSelect+" WHERE p.id = $1 AND p.enterprise_id = $2", requestID, enterpriseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, userentity.ErrPasswordChangeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &request, nil
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
			WHERE id=$1 AND user_id=$2 AND enterprise_id=$3 AND status='APPROVED' AND expires_at<=now()`, requestID, userID, enterpriseID)
		if expireErr == nil && expired.RowsAffected() == 1 {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return commitErr
			}
			return userentity.ErrPasswordChangeExpired
		}
		return errPasswordChangeNotAllowed
	}
	result, err = tx.Exec(ctx, `UPDATE users SET password=$2, auth_version=auth_version+1, updated_at=now() WHERE id=$1`, userID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if result.RowsAffected() != 1 {
		return errors.New("nenhum usuário foi atualizado na troca de senha")
	}
	return tx.Commit(ctx)
}
