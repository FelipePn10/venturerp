package production_order_uc

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/shopspring/decimal"
	"time"
)

type DeliveryCandidateFilter struct {
	OrderFrom, OrderTo, ItemFrom, ItemTo, PlannerFrom, PlannerTo *int64
	From, To                                                     *time.Time
	OrderType                                                    string
}
type DeliveryCandidate struct {
	ID                 int64           `json:"id"`
	OrderNumber        int64           `json:"order_number"`
	Planner            *int64          `json:"planner,omitempty"`
	OrderType          string          `json:"order_type"`
	ItemCode           int64           `json:"item_code"`
	Description        string          `json:"description"`
	Mask               string          `json:"mask,omitempty"`
	Planned            decimal.Decimal `json:"planned"`
	Delivered          decimal.Decimal `json:"delivered"`
	Pending            decimal.Decimal `json:"pending"`
	WarehouseID        *int64          `json:"warehouse_id,omitempty"`
	StartDate, EndDate *time.Time
}
type deliveryCandidateReader interface {
	ListDeliveryCandidates(context.Context, DeliveryCandidateFilter) ([]DeliveryCandidate, error)
}
type ListDeliveryCandidatesUseCase struct {
	Reader deliveryCandidateReader
	Auth   ports.AuthService
}

func (uc *ListDeliveryCandidatesUseCase) Execute(ctx context.Context, filter DeliveryCandidateFilter) ([]DeliveryCandidate, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Reader.ListDeliveryCandidates(ctx, filter)
}
