package industrial_calendar_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity"
)

func toIndustrialCalendarResponse(c *entity.IndustrialCalendar) *response.IndustrialCalendarResponse {
	if c == nil {
		return nil
	}
	return &response.IndustrialCalendarResponse{
		Year:        c.Year,
		Month:       c.Month,
		Day:         c.Day,
		IsWorkday:   c.IsWorkday,
		Description: c.Description,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func toIndustrialCalendarResponses(list []*entity.IndustrialCalendar) []*response.IndustrialCalendarResponse {
	out := make([]*response.IndustrialCalendarResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toIndustrialCalendarResponse(c))
	}
	return out
}
