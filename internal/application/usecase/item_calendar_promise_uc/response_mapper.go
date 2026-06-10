package item_calendar_promise_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/entity"
)

func toItemCalendarPromiseResponse(c *entity.ItemCalendarPromise) *response.ItemCalendarPromiseResponse {
	if c == nil {
		return nil
	}
	return &response.ItemCalendarPromiseResponse{
		ID:          c.ID,
		ItemCode:    c.ItemCode,
		Mask:        c.Mask,
		Year:        c.Year,
		Month:       c.Month,
		Day:         c.Day,
		IsWorkday:   c.IsWorkday,
		Description: c.Description,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func toItemCalendarPromiseResponses(list []*entity.ItemCalendarPromise) []*response.ItemCalendarPromiseResponse {
	out := make([]*response.ItemCalendarPromiseResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toItemCalendarPromiseResponse(c))
	}
	return out
}
