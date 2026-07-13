package purchase_tolerance_uc

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_tolerance/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_tolerance/repository"
	"github.com/shopspring/decimal"
	"strings"
)

type UseCase struct {
	Repo repository.Repository
	Auth ports.AuthService
}

func (uc *UseCase) Save(ctx context.Context, d request.UpsertPurchaseToleranceDTO) (*response.PurchaseToleranceResponse, error) {
	e, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	by, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := entity.New(e, d.ToleranceType, d.AppliesTo, d.IntervalMin, d.IntervalMax, d.ToleranceValue, d.ValueType, d.SupplierCode, d.Action, by)
	if err != nil {
		return nil, err
	}
	x.ID = d.ID
	if d.IsActive != nil {
		x.IsActive = *d.IsActive
	}
	saved, err := uc.Repo.Save(ctx, x)
	if err != nil {
		return nil, err
	}
	return mapResponse(saved), nil
}
func (uc *UseCase) List(ctx context.Context, supplier *int64) ([]response.PurchaseToleranceResponse, error) {
	e, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.Repo.List(ctx, e, supplier)
	if err != nil {
		return nil, err
	}
	out := make([]response.PurchaseToleranceResponse, 0, len(x))
	for _, v := range x {
		out = append(out, *mapResponse(v))
	}
	return out, nil
}
func (uc *UseCase) Delete(ctx context.Context, id int64) error {
	e, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return err
	}
	return uc.Repo.Delete(ctx, e, id)
}
func (uc *UseCase) Evaluate(ctx context.Context, d request.EvaluatePurchaseToleranceDTO) (entity.Evaluation, error) {
	e, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return entity.Evaluation{}, err
	}
	return uc.evaluate(ctx, e, d.SupplierCode, d.ToleranceType, d.AppliesTo, d.Expected, d.Actual)
}
func (uc *UseCase) evaluate(ctx context.Context, e int64, supplier *int64, t, a string, expected, actual decimal.Decimal) (entity.Evaluation, error) {
	t, a = strings.ToUpper(t), strings.ToUpper(a)
	rule, err := uc.Repo.Resolve(ctx, e, supplier, t, a, expected.Abs())
	if err != nil {
		return entity.Evaluation{}, err
	}
	if rule == nil {
		return entity.Evaluation{Expected: expected, Actual: actual, Deviation: actual.Sub(expected).Abs()}, nil
	}
	return rule.Evaluate(expected, actual), nil
}
func (uc *UseCase) EvaluatePurchaseTolerance(ctx context.Context, supplier *int64, t, a string, expected, actual decimal.Decimal) (string, string, bool, error) {
	e, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return "", "", false, err
	}
	v, err := uc.evaluate(ctx, e, supplier, t, a, expected, actual)
	if err != nil || !v.Matched {
		return "", "", false, err
	}
	msg := fmt.Sprintf("%s deviation %s exceeds allowed %s", strings.ToLower(t), v.Deviation.String(), v.Allowed.String())
	return v.Action, msg, v.Exceeded, nil
}
func mapResponse(x *entity.Tolerance) *response.PurchaseToleranceResponse {
	return &response.PurchaseToleranceResponse{ID: x.ID, ToleranceType: x.ToleranceType, AppliesTo: x.AppliesTo, IntervalMin: x.IntervalMin, IntervalMax: x.IntervalMax, ToleranceValue: x.ToleranceValue, ValueType: x.ValueType, SupplierCode: x.SupplierCode, Action: x.Action, IsActive: x.IsActive, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, CreatedBy: x.CreatedBy}
}
