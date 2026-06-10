package component_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/component/entity"
)

func toComponentResponse(c *entity.Component) *response.ComponentResponse {
	if c == nil {
		return nil
	}
	return &response.ComponentResponse{
		ID:        c.ID,
		Name:      c.Name,
		GroupCode: c.GroupCode,
		Code:      c.Code,
		Warehouse: c.Warehouse,
		CreatedBy: c.CreatedBy,
		CreatedAt: c.CreatedAt,
	}
}
