package mrp_calculation_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

type profileStockReader interface {
	ListBalancesByItem(context.Context, int64) ([]*stockentity.StockBalance, error)
}

type GetItemProfileUseCase struct {
	Repo  repository.MRPCalculationRepository
	Auth  ports.AuthService
	Stock profileStockReader
}

func (uc *GetItemProfileUseCase) Consult(ctx context.Context, itemCode, planCode int64, position string, from, to *time.Time) (*response.MRPItemProfileConsultationResponse, error) {
	rows, err := uc.Execute(ctx, itemCode, planCode)
	if err != nil {
		return nil, err
	}
	position = strings.ToUpper(strings.TrimSpace(position))
	if position == "" {
		position = "CALCULATION"
	}
	if position != "CALCULATION" && position != "CURRENT" {
		return nil, errorsuc.NewValidationError("position must be CALCULATION or CURRENT")
	}
	filtered := make([]*response.MRPItemProfileResponse, 0, len(rows))
	totals := map[string]float64{}
	for _, row := range rows {
		if from != nil && row.NeedDate.Before(*from) || to != nil && row.NeedDate.After(*to) {
			continue
		}
		filtered = append(filtered, row)
		totals["demand"] += row.Demand
		totals["orders_planned"] += row.OrdersPlanned
		totals["orders_firm"] += row.OrdersFirm
	}
	if len(filtered) > 0 {
		totals["stock_projected"] = filtered[len(filtered)-1].StockProjected
	}
	if position == "CURRENT" && uc.Stock != nil {
		balances, err := uc.Stock.ListBalancesByItem(ctx, itemCode)
		if err != nil {
			return nil, err
		}
		totals["stock_current"] = 0
		for _, balance := range balances {
			totals["stock_current"] += balance.Quantity
		}
	}
	return &response.MRPItemProfileConsultationResponse{Position: position, Rows: filtered, Totals: totals}, nil
}

func (uc *GetItemProfileUseCase) Execute(
	ctx context.Context,
	itemCode, planCode int64,
) ([]*response.MRPItemProfileResponse, error) {
	if !uc.Auth.CanRunMRPCalculation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.GetProfiles(ctx, itemCode, planCode)
	if err != nil {
		return nil, err
	}
	return toMRPItemProfileResponses(list), nil
}
