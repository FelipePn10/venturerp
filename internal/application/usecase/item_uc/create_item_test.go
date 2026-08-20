package item_uc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

type createItemAuth struct{ ports.AuthService }

func (createItemAuth) CanCreateItem(context.Context) bool          { return true }
func (createItemAuth) EnterpriseID(context.Context) (int64, error) { return 7, nil }
func (createItemAuth) UserID(context.Context) (uuid.UUID, error) {
	return uuid.MustParse("00000000-0000-0000-0000-000000000007"), nil
}

type createItemRepository struct {
	missingItemRepository
	base      *entity.Item
	created   *entity.Item
	pdmErr    error
	pdmTenant int64
}

func (r *createItemRepository) ValidatePDMReferences(_ context.Context, tenant int64, _, _ int) error {
	r.pdmTenant = tenant
	return r.pdmErr
}

func (r *createItemRepository) NextAutomaticBusinessCode(context.Context, int64) (valueobject.BusinessCode, error) {
	return "1", nil
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

func TestCreateItemGeneratesCodeWhenOmitted(t *testing.T) {
	repo := &createItemRepository{}
	uc := NewCreateItemUseCase(repo, createItemAuth{})
	item := itemForCreateTest()
	item.BusinessCode = ""
	created, err := uc.Execute(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if created.Code != "1" {
		t.Fatalf("codigo automatico inesperado: %q", created.Code)
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

func TestCreateItemRejectsInvalidOrCrossTenantPDMBeforePersisting(t *testing.T) {
	for _, tt := range []struct {
		name            string
		group, modifier int
	}{{"grupo inexistente", 999999, 0}, {"modificador inexistente", 0, 999999}, {"referência de outra empresa", 77, 88}} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &createItemRepository{pdmErr: fmt.Errorf("%w: referência PDM fora da empresa", repository.ErrInvalidReference)}
			item := itemForCreateTest()
			item.PDM.GroupCode = int32(tt.group)
			item.PDM.ModifierCode = int32(tt.modifier)
			_, err := NewCreateItemUseCase(repo, createItemAuth{}).Execute(context.Background(), item)
			if !errors.Is(err, repository.ErrInvalidReference) {
				t.Fatalf("erro inesperado: %v", err)
			}
			if repo.created != nil {
				t.Fatal("item foi persistido parcialmente")
			}
			if repo.pdmTenant != 7 {
				t.Fatalf("tenant validado=%d", repo.pdmTenant)
			}
		})
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
