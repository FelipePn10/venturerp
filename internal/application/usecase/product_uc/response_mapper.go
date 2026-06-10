package product_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/product/entity"
)

func toProductResponse(p *entity.Product) *response.ProductResponse {
	if p == nil {
		return nil
	}
	return &response.ProductResponse{
		ID:        p.ID,
		Code:      p.Code,
		GroupCode: p.GroupCode,
		Name:      p.Name,
		CreatedBy: p.CreatedBy,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
