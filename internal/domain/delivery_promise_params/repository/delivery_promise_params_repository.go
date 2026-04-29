package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity"
)

type DeliveryPromiseParamsRepository interface {
	Get(ctx context.Context) (*entity.DeliveryPromiseParams, error)
	Save(ctx context.Context, p *entity.DeliveryPromiseParams) (*entity.DeliveryPromiseParams, error)
}
