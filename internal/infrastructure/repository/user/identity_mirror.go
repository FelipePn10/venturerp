package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdentityMirror copies only identity records needed by foreign keys in the
// training database. It never copies operational data.
type IdentityMirror struct {
	authority *pgxpool.Pool
	target    *pgxpool.Pool
}

func NewIdentityMirror(authority, target *pgxpool.Pool) *IdentityMirror {
	return &IdentityMirror{authority: authority, target: target}
}

func (m *IdentityMirror) SyncIdentity(ctx context.Context, userID string, enterpriseID int64) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	var name, email, globalRole, associationRole, enterpriseName string
	var authVersion, enterpriseCode int64
	var active bool
	err = m.authority.QueryRow(ctx, `
		SELECT u.name,u.email,u.role,u.auth_version,u.is_active,
		       e.code::bigint,e.name,ue.role
		FROM users u
		JOIN user_enterprises ue ON ue.user_id=u.id
		JOIN enterprise e ON e.id=ue.enterprise_id
		WHERE u.id=$1 AND e.id=$2`, id, enterpriseID).Scan(
		&name, &email, &globalRole, &authVersion, &active,
		&enterpriseCode, &enterpriseName, &associationRole,
	)
	if err != nil {
		return fmt.Errorf("read authoritative identity: %w", err)
	}

	tx, err := m.target.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin identity mirror: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `
		INSERT INTO users(id,name,email,password,role,auth_version,is_active,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,now(),now())
		ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,email=EXCLUDED.email,
		password=EXCLUDED.password,role=EXCLUDED.role,auth_version=EXCLUDED.auth_version,
		is_active=EXCLUDED.is_active,updated_at=now()`,
		id, name, email, "!managed-by-production-identity-authority", globalRole, authVersion, active); err != nil {
		return fmt.Errorf("mirror user: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO enterprise(id,code,name,created_at) VALUES($1,$2,$3,now())
		ON CONFLICT(id) DO UPDATE SET code=EXCLUDED.code,name=EXCLUDED.name`,
		enterpriseID, enterpriseCode, enterpriseName); err != nil {
		return fmt.Errorf("mirror enterprise: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO user_enterprises(user_id,enterprise_id,role,created_at)
		VALUES($1,$2,$3,now())
		ON CONFLICT(user_id,enterprise_id) DO UPDATE SET role=EXCLUDED.role`,
		id, enterpriseID, associationRole); err != nil {
		return fmt.Errorf("mirror authorization: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit identity mirror: %w", err)
	}
	return nil
}
