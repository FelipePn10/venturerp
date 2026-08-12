package item_uc

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type createItemAuth struct{ ports.AuthService }

func (createItemAuth) CanCreateItem(context.Context) bool          { return true }
func (createItemAuth) EnterpriseID(context.Context) (int64, error) { return 7, nil }
func (createItemAuth) UserID(context.Context) (uuid.UUID, error) {
	return uuid.MustParse("00000000-0000-0000-0000-000000000007"), nil
}

type createItemRepository struct {
	missingItemRepository
	base    *entity.Item
	created *entity.Item
}

func (r *createItemRepository) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*entity.Item, error) {
	if r.base == nil || r.base.Code != code {
		return nil, repository.ErrNotFound
	}
	return r.base, nil
}

func (r *createItemRepository) Create(_ context.Context, item *entity.Item) (*entity.Item, error) {
	r.created = item
	return item, nil
}

func TestCreateItemAcceptsNoItemBase(t *testing.T) {
	repo := &createItemRepository{}
	uc := NewCreateItemUseCase(repo, createItemAuth{})
	item := itemForCreateTest()

	if _, err := uc.Execute(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	if repo.created != item {
		t.Fatal("expected item to be persisted")
	}
}

func TestCreateItemValidatesExistingBaseWithoutCopyingItsFields(t *testing.T) {
	baseCode := 10
	repo := &createItemRepository{base: &entity.Item{Code: valueobject.ItemCode(baseCode), Name: "Nome do modelo"}}
	uc := NewCreateItemUseCase(repo, createItemAuth{})
	item := itemForCreateTest()
	item.Name = "Nome informado"
	item.Engineering.ItemBaseCod = &baseCode
	commercialDescription := "Descrição alterada"
	item.Commercial.Description = &commercialDescription
	accountingInputCode := "ENT-ALTERADO"
	item.Accounting.InputCode = &accountingInputCode

	created, err := uc.Execute(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Nome informado" ||
		created.Commercial.Description == nil || *created.Commercial.Description != commercialDescription ||
		created.Accounting.InputCode == nil || *created.Accounting.InputCode != accountingInputCode {
		t.Fatalf("frontend values were not preserved: %#v", created)
	}
}

func TestCreateItemRejectsUnknownBaseInPortuguese(t *testing.T) {
	baseCode := 999
	repo := &createItemRepository{}
	uc := NewCreateItemUseCase(repo, createItemAuth{})
	item := itemForCreateTest()
	item.Engineering.ItemBaseCod = &baseCode

	_, err := uc.Execute(context.Background(), item)
	if !errors.Is(err, ErrItemBaseNotFound) {
		t.Fatalf("expected localized missing base error, got %v", err)
	}
}

func itemForCreateTest() *entity.Item {
	return &entity.Item{
		Code:         valueobject.ItemCode(12345),
		BusinessCode: valueobject.BusinessCode("TEA452-0"),
		Name:         "Transformador 30 kVA",
		Nature:       entity.ItemConfigured,
		Situation:    types.LINHA,
		Health:       types.ACTIVE,
		Engineering:  entity.Engineering{},
	}
}
