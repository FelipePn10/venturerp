package sales_forecast_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
)

type ActualDemandResponse struct {
	ItemCode int64   `json:"item_code"`
	Mask     *string `json:"mask,omitempty"`
	Year     int     `json:"year"`
	Month    int     `json:"month"`
	Quantity float64 `json:"quantity"`
}

type ListActualDemandUseCase struct {
	Repo repository.SalesForecastRepository
	Auth ports.AuthService
}

func (uc *ListActualDemandUseCase) Execute(ctx context.Context, year int, itemCode int64, source string) ([]ActualDemandResponse, error) {
	if !uc.Auth.CanListSalesForecasts(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if year <= 0 || itemCode < 0 {
		return nil, errorsuc.NewValidationError("ano deve ser positivo e o código do item não pode ser negativo")
	}
	source = strings.ToUpper(strings.TrimSpace(source))
	if source == "" {
		source = "ORDERS"
	}
	if source != "ORDERS" && source != "INVOICING" && source != "BOTH" {
		return nil, errorsuc.NewValidationError("fonte deve ser ORDERS, INVOICING ou BOTH")
	}
	items := []int64{}
	if itemCode > 0 {
		items = append(items, itemCode)
	}
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
	rows, err := uc.Repo.ListHistoricalDemand(ctx, source, from, to, items)
	if err != nil {
		return nil, err
	}
	out := make([]ActualDemandResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, ActualDemandResponse{ItemCode: row.ItemCode, Mask: row.Mask, Year: row.PeriodMonth.Year(), Month: int(row.PeriodMonth.Month()), Quantity: row.Quantity})
	}
	return out, nil
}
