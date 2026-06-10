package stock_movement_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity"
)

func toStockMovementTypeResponse(s *entity.StockMovementType) *response.StockMovementTypeResponse {
	if s == nil {
		return nil
	}
	return &response.StockMovementTypeResponse{
		ID:                   s.ID,
		Sigla:                s.Sigla,
		Description:          s.Description,
		UsageType:            string(s.UsageType),
		EntryOrder:           s.EntryOrder,
		ExitOrder:            s.ExitOrder,
		ConsidersConsumption: s.ConsidersConsumption,
		UpdatesAvgCost:       s.UpdatesAvgCost,
		IsAdjustment:         s.IsAdjustment,
		UpdatesCycleCount:    s.UpdatesCycleCount,
		ShowsInSummary:       s.ShowsInSummary,
		EntryExit:            string(s.EntryExit),
		GeneratesFCIMovement: s.GeneratesFCIMovement,
		IsActive:             s.IsActive,
		CreatedAt:            s.CreatedAt,
	}
}

func toStockMovementTypeResponses(list []*entity.StockMovementType) []*response.StockMovementTypeResponse {
	out := make([]*response.StockMovementTypeResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toStockMovementTypeResponse(s))
	}
	return out
}
