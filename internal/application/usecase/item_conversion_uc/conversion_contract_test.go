package item_conversion_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type convItems struct{ item *itementity.Item }

func (f *convItems) FindItemByBusinessCode(_ context.Context, code valueobject.BusinessCode) (*itementity.Item, error) {
	if f.item != nil && f.item.BusinessCode == code {
		return f.item, nil
	}
	return nil, itemrepo.ErrNotFound
}
func (f *convItems) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*itementity.Item, error) {
	if f.item != nil && f.item.Code == code {
		return f.item, nil
	}
	return nil, itemrepo.ErrNotFound
}

type convAuth struct {
	ports.AuthService
	actor uuid.UUID
}

func (a *convAuth) UserID(context.Context) (uuid.UUID, error) { return a.actor, nil }

func newContractUC(t *testing.T) (*ItemConversionUseCase, *fakeRepo) {
	t.Helper()
	repo := &fakeRepo{}
	items := &convItems{item: &itementity.Item{Code: 4200, BusinessCode: "TB-40X40", Name: "Tubo 40x40"}}
	return NewItemConversionUseCase(repo, &convAuth{actor: uuid.New()}, items), repo
}

// A tela envia o código de negócio do item; a conversão precisa salvar e listar
// com esse mesmo código.
func TestCreateAndList_ByBusinessCode(t *testing.T) {
	uc, _ := newContractUC(t)
	ctx := context.Background()
	created, err := uc.Create(ctx, request.CreateItemConversionDTO{
		ItemCode: "TB-40X40", FromUOM: "kg", ToUOM: "m", Factor: 0.4,
	})
	if err != nil {
		t.Fatalf("cadastro rejeitado: %v", err)
	}
	if created.ItemCode != "TB-40X40" || created.LegacyCode != 4200 {
		t.Fatalf("resposta não carrega o código de negócio: %+v", created)
	}
	if created.FromUOM != "KG" || created.ToUOM != "M" {
		t.Fatalf("unidades não normalizadas: %+v", created)
	}

	legacy, err := uc.ResolveItem(ctx, "TB-40X40")
	if err != nil {
		t.Fatal(err)
	}
	list, err := uc.ListByItem(ctx, legacy)
	if err != nil || len(list) != 1 {
		t.Fatalf("listagem = %d err=%v, quer 1", len(list), err)
	}
	if list[0].ItemCode != "TB-40X40" || list[0].ItemName != "Tubo 40x40" {
		t.Fatalf("listagem sem os dados do item: %+v", list[0])
	}
}

func TestCreate_DuplicatePairIsConflict(t *testing.T) {
	uc, _ := newContractUC(t)
	ctx := context.Background()
	dto := request.CreateItemConversionDTO{ItemCode: "TB-40X40", FromUOM: "KG", ToUOM: "M", Factor: 0.4}
	if _, err := uc.Create(ctx, dto); err != nil {
		t.Fatal(err)
	}
	_, err := uc.Create(ctx, dto)
	if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("esperado ConflictError, veio %T (%v)", err, err)
	}
}

func TestCreate_UnknownItemIsRejected(t *testing.T) {
	uc, _ := newContractUC(t)
	_, err := uc.Create(context.Background(), request.CreateItemConversionDTO{
		ItemCode: "NAO-EXISTE", FromUOM: "KG", ToUOM: "M", Factor: 1,
	})
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
}

func TestCreate_ValidationMessagesArePortuguese(t *testing.T) {
	uc, _ := newContractUC(t)
	ctx := context.Background()
	cases := []struct {
		name string
		dto  request.CreateItemConversionDTO
	}{
		{"sem unidades", request.CreateItemConversionDTO{ItemCode: "TB-40X40", Factor: 1}},
		{"unidades iguais", request.CreateItemConversionDTO{ItemCode: "TB-40X40", FromUOM: "KG", ToUOM: "kg", Factor: 1}},
		{"fator zerado", request.CreateItemConversionDTO{ItemCode: "TB-40X40", FromUOM: "KG", ToUOM: "M", Factor: 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.Create(ctx, tc.dto)
			v, ok := errorsuc.AsValidation(err)
			if !ok {
				t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
			}
			for _, english := range []string{"required", "must", "invalid"} {
				if strings.Contains(v.Error(), english) {
					t.Fatalf("mensagem em inglês: %q", v.Error())
				}
			}
		})
	}
}
