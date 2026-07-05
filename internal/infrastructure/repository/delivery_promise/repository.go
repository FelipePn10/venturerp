package delivery_promise

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) NextReservationCode(ctx context.Context) (int64, error) {
	var code int64
	err := r.pool.QueryRow(ctx, `
		UPDATE delivery_tank_reservation_sequences
		SET last_code = last_code + 1
		WHERE id = 1
		RETURNING last_code
	`).Scan(&code)
	if err != nil {
		return 0, fmt.Errorf("next delivery tank reservation code: %w", err)
	}
	return code, nil
}

func (r *Repository) CreateReservation(ctx context.Context, res *entity.TankReservation) (*entity.TankReservation, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO delivery_tank_reservations (
			code, customer_code, item_code, mask, tank_code, requested_qty, reserved_qty,
			allocation_date, expires_at, status, notes, created_by
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, code, customer_code, item_code, mask, tank_code, requested_qty, reserved_qty,
			allocation_date, expires_at, status, notes, created_at, updated_at, created_by
	`, res.Code, res.CustomerCode, res.ItemCode, res.Mask, res.TankCode, res.RequestedQty, res.ReservedQty,
		res.AllocationDate, res.ExpiresAt, string(res.Status), res.Notes, res.CreatedBy)
	out, err := scanReservation(row)
	if err != nil {
		return nil, fmt.Errorf("creating delivery tank reservation: %w", err)
	}
	return out, nil
}

func (r *Repository) ListActiveReservations(ctx context.Context, from, to time.Time, tankCodes []int64) ([]*entity.TankReservation, error) {
	args := []any{from, to}
	filter := ""
	if len(tankCodes) > 0 {
		placeholders := make([]string, 0, len(tankCodes))
		for _, code := range tankCodes {
			args = append(args, code)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		filter = " AND tank_code IN (" + strings.Join(placeholders, ",") + ")"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, customer_code, item_code, mask, tank_code, requested_qty, reserved_qty,
			allocation_date, expires_at, status, notes, created_at, updated_at, created_by
		FROM delivery_tank_reservations
		WHERE status = 'ACTIVE'
		  AND allocation_date BETWEEN $1 AND $2
		`+filter+`
		ORDER BY allocation_date, tank_code, code
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("listing active tank reservations: %w", err)
	}
	defer rows.Close()

	out := []*entity.TankReservation{}
	for rows.Next() {
		res, err := scanReservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, rows.Err()
}

func (r *Repository) CancelReservation(ctx context.Context, code int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE delivery_tank_reservations
		SET status = 'CANCELLED', updated_at = NOW()
		WHERE code = $1 AND status = 'ACTIVE'
	`, code)
	if err != nil {
		return fmt.Errorf("cancelling tank reservation %d: %w", code, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("active tank reservation %d not found", code)
	}
	return nil
}

func (r *Repository) ExpireReservations(ctx context.Context, now time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE delivery_tank_reservations
		SET status = 'EXPIRED', updated_at = NOW()
		WHERE status = 'ACTIVE' AND expires_at < $1
	`, now)
	if err != nil {
		return 0, fmt.Errorf("expiring tank reservations: %w", err)
	}
	return tag.RowsAffected(), nil
}

type reservationScanner interface {
	Scan(dest ...any) error
}

func scanReservation(row reservationScanner) (*entity.TankReservation, error) {
	var res entity.TankReservation
	var status string
	var customerCode *int64
	var notes *string
	var createdBy uuid.UUID
	err := row.Scan(
		&res.ID,
		&res.Code,
		&customerCode,
		&res.ItemCode,
		&res.Mask,
		&res.TankCode,
		&res.RequestedQty,
		&res.ReservedQty,
		&res.AllocationDate,
		&res.ExpiresAt,
		&status,
		&notes,
		&res.CreatedAt,
		&res.UpdatedAt,
		&createdBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("scanning tank reservation: %w", err)
	}
	res.CustomerCode = customerCode
	res.Notes = notes
	res.Status = entity.TankReservationStatus(status)
	res.CreatedBy = createdBy
	return &res, nil
}
