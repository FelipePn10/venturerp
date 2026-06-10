package sales_forecast_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
)

func toSalesForecastResponse(f *entity.SalesForecast) *response.SalesForecastResponse {
	if f == nil {
		return nil
	}
	return &response.SalesForecastResponse{
		ID:        f.ID,
		ItemCode:  f.ItemCode,
		Mask:      f.Mask,
		Week:      f.Week,
		Year:      f.Year,
		Quantity:  f.Quantity,
		CreatedBy: f.CreatedBy,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

func toSalesForecastResponses(forecasts []*entity.SalesForecast) []*response.SalesForecastResponse {
	out := make([]*response.SalesForecastResponse, 0, len(forecasts))
	for _, f := range forecasts {
		out = append(out, toSalesForecastResponse(f))
	}
	return out
}

func toForecastBlockResponse(b *entity.SalesForecastBlock) *response.SalesForecastBlockResponse {
	if b == nil {
		return nil
	}
	return &response.SalesForecastBlockResponse{
		ID:        b.ID,
		StartDate: b.StartDate,
		EndDate:   b.EndDate,
		Reason:    b.Reason,
		CreatedAt: b.CreatedAt,
		CreatedBy: b.CreatedBy,
	}
}

func toForecastBlockResponses(blocks []*entity.SalesForecastBlock) []*response.SalesForecastBlockResponse {
	out := make([]*response.SalesForecastBlockResponse, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, toForecastBlockResponse(b))
	}
	return out
}

func toAppropriationTableResponse(t *entity.AppropriationTable) *response.AppropriationTableResponse {
	if t == nil {
		return nil
	}
	return &response.AppropriationTableResponse{
		ID:           t.ID,
		Description:  t.Description,
		MondayPct:    t.MondayPct,
		TuesdayPct:   t.TuesdayPct,
		WednesdayPct: t.WednesdayPct,
		ThursdayPct:  t.ThursdayPct,
		FridayPct:    t.FridayPct,
		SaturdayPct:  t.SaturdayPct,
		SundayPct:    t.SundayPct,
		IsDefault:    t.IsDefault,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		CreatedBy:    t.CreatedBy,
	}
}

func toAppropriationTableResponses(tables []*entity.AppropriationTable) []*response.AppropriationTableResponse {
	out := make([]*response.AppropriationTableResponse, 0, len(tables))
	for _, t := range tables {
		out = append(out, toAppropriationTableResponse(t))
	}
	return out
}
