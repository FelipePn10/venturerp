package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/entity"
)

type TankReservationRepository interface {
	NextReservationCode(ctx context.Context) (int64, error)
	CreateReservation(ctx context.Context, r *entity.TankReservation) (*entity.TankReservation, error)
	ListActiveReservations(ctx context.Context, from, to time.Time, tankCodes []int64) ([]*entity.TankReservation, error)
	CancelReservation(ctx context.Context, code int64) error
	ExpireReservations(ctx context.Context, now time.Time) (int64, error)
}
