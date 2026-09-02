package warehouse_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity"
	"github.com/google/uuid"
)

type warehouseAuth struct {
	ports.AuthService
	user uuid.UUID
}

func (a warehouseAuth) CanCreateWarehouse(context.Context) bool   { return true }
func (a warehouseAuth) UserID(context.Context) (uuid.UUID, error) { return a.user, nil }

type warehouseRepo struct{ created *entity.Warehouse }

func (r *warehouseRepo) Create(_ context.Context, value *entity.Warehouse) (*entity.Warehouse, error) {
	r.created = value
	return value, nil
}
func (*warehouseRepo) List(context.Context) ([]*entity.Warehouse, error)            { return nil, nil }
func (*warehouseRepo) GetByCode(context.Context, string) (*entity.Warehouse, error) { return nil, nil }

func TestCreateWarehouseUsesJWTActor(t *testing.T) {
	actor := uuid.New()
	repo := &warehouseRepo{}
	uc := NewCreateWarehouseUseCase(repo, warehouseAuth{user: actor})
	_, err := uc.Execute(context.Background(), request.CreateWarehouseRequestDTO{Code: "ALMOX01", Description: "Principal", Location: types.INTERNO, Type: types.NORMAL})
	if err != nil {
		t.Fatal(err)
	}
	if repo.created.CreatedBy != actor {
		t.Fatalf("ator=%s", repo.created.CreatedBy)
	}
	if repo.created.Code != "ALMOX01" {
		t.Fatalf("code=%q", repo.created.Code)
	}
}
