package preferencias

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/preferencias_uc"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Get(ctx context.Context, userID uuid.UUID) (preferencias_uc.Preferencias, bool, error) {
	var p preferencias_uc.Preferencias
	err := r.pool.QueryRow(ctx, `SELECT customer_name_display FROM user_preferences WHERE user_id = $1`, userID).Scan(&p.CustomerNameDisplay)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, false, nil
	}
	if err != nil {
		return p, false, fmt.Errorf("lendo as preferências do usuário: %w", err)
	}
	return p, true, nil
}

func (r *Repository) Save(ctx context.Context, userID uuid.UUID, p preferencias_uc.Preferencias) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_preferences (user_id, customer_name_display, updated_at) VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET customer_name_display = EXCLUDED.customer_name_display, updated_at = NOW()`,
		userID, p.CustomerNameDisplay)
	if err != nil {
		return fmt.Errorf("gravando as preferências do usuário: %w", err)
	}
	return nil
}
