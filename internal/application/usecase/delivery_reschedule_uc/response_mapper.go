package delivery_reschedule_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
)

func toDeliveryRescheduleResponse(r *entity.DeliveryReschedule) *response.DeliveryRescheduleResponse {
	if r == nil {
		return nil
	}
	return &response.DeliveryRescheduleResponse{
		Code:           r.Code,
		SalesOrderCode: r.SalesOrderCode,
		ItemCode:       int64(r.ItemCode),
		OldDate:        r.OldDate,
		NewDate:        r.NewDate,
		Reason:         r.Reason,
		CreatedAt:      r.CreatedAt,
		CreatedBy:      r.CreatedBy,
	}
}

func toDeliveryRescheduleResponses(list []*entity.DeliveryReschedule) []*response.DeliveryRescheduleResponse {
	out := make([]*response.DeliveryRescheduleResponse, 0, len(list))
	for _, r := range list {
		out = append(out, toDeliveryRescheduleResponse(r))
	}
	return out
}
