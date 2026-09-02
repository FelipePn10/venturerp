package sales_division_uc

import (
	"context"
	"errors"
	"testing"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity"
)

func (r *deleteDivisionRepo) SetActive(_ context.Context, code int64, active bool) (*entity.SalesDivision, error) {
	return &entity.SalesDivision{Code: code, IsActive: active}, nil
}

func TestSetSalesDivisionStatusRequiresAdminPermission(t *testing.T) {
	uc := &SetSalesDivisionStatusUseCase{Repo: &deleteDivisionRepo{}, Auth: deleteDivisionAuth{}}
	_, err := uc.Execute(context.Background(), 10, false)
	if !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("esperava falta de permissão: %v", err)
	}
}

func TestSetSalesDivisionStatusPreservesRecord(t *testing.T) {
	uc := &SetSalesDivisionStatusUseCase{Repo: &deleteDivisionRepo{}, Auth: deleteDivisionAuth{allowed: true}}
	result, err := uc.Execute(context.Background(), 10, false)
	if err != nil || result.Code != 10 || result.IsActive {
		t.Fatalf("resultado inesperado: result=%+v err=%v", result, err)
	}
}
