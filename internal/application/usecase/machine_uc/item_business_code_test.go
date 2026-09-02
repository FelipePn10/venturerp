package machine_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	machinerepo "github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type machineItemAuth struct{ ports.AuthService }

func (machineItemAuth) CanCreateItemTimeMachine(context.Context) bool { return true }
func (machineItemAuth) CanListItemMachineTimes(context.Context) bool  { return true }

type machineItemRepository struct {
	machinerepo.MachineRepository
	created  *machineentity.ItemMachineTime
	listCode int64
}

func (r *machineItemRepository) GetByCode(context.Context, int64) (*machineentity.Machine, error) {
	return &machineentity.Machine{CapacityUnit: types.Units}, nil
}

func (r *machineItemRepository) CreateItemMachineTime(_ context.Context, value *machineentity.ItemMachineTime) (*machineentity.ItemMachineTime, error) {
	r.created = value
	return value, nil
}

func (r *machineItemRepository) ListItemMachineTimes(_ context.Context, itemCode int64) ([]*machineentity.ItemMachineTime, error) {
	r.listCode = itemCode
	return []*machineentity.ItemMachineTime{{ItemCode: itemCode}}, nil
}

type machineItemCatalog struct {
	itemrepo.ItemRepository
	item *itementity.Item
}

func (r machineItemCatalog) FindItemByBusinessCode(_ context.Context, code valueobject.BusinessCode) (*itementity.Item, error) {
	if r.item != nil && r.item.BusinessCode == code {
		return r.item, nil
	}
	return nil, itemrepo.ErrNotFound
}

func (r machineItemCatalog) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*itementity.Item, error) {
	if r.item != nil && r.item.Code == code {
		return r.item, nil
	}
	return nil, itemrepo.ErrNotFound
}

func TestCreateItemMachineTimeResolvesBusinessCodeToLegacyKey(t *testing.T) {
	repo := &machineItemRepository{}
	catalog := machineItemCatalog{item: &itementity.Item{
		Code: 42, BusinessCode: "TEA452-0",
		Warehouse: itementity.Warehouse{UnitOfMeasurement: types.TypeUnitOfMeasurementItem(types.Units)},
	}}
	uc := CreateItemMachineTimeUseCase{Repo: repo, ItemRepo: catalog, Auth: machineItemAuth{}}

	_, err := uc.Execute(context.Background(), request.CreateItemMachineTimeDTO{
		ItemCode: request.TextCode("TEA452-0"), MachineCode: 7,
	})
	if err != nil {
		t.Fatalf("criação por código comercial falhou: %v", err)
	}
	if repo.created == nil || repo.created.ItemCode != 42 {
		t.Fatalf("chave legada persistida = %#v; esperada 42", repo.created)
	}
}

func TestListItemMachineTimesResolvesBusinessCodeAndKeepsOptionalFilter(t *testing.T) {
	repo := &machineItemRepository{}
	catalog := machineItemCatalog{item: &itementity.Item{Code: 42, BusinessCode: "TEA452-0"}}
	uc := ListItemMachineTimesUseCase{Repo: repo, ItemRepo: catalog, Auth: machineItemAuth{}}

	if _, err := uc.Execute(context.Background(), request.TextCode("TEA452-0")); err != nil {
		t.Fatalf("listagem por código comercial falhou: %v", err)
	}
	if repo.listCode != 42 {
		t.Fatalf("filtro legado = %d; esperado 42", repo.listCode)
	}
	if _, err := uc.Execute(context.Background(), ""); err != nil {
		t.Fatalf("listagem sem filtro falhou: %v", err)
	}
	if repo.listCode != 0 {
		t.Fatalf("listagem sem filtro usou código %d; esperado 0", repo.listCode)
	}
}
