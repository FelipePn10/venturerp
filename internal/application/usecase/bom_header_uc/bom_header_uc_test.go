package bom_header_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type fakeItems struct {
	item *itementity.Item
}

func (f *fakeItems) FindItemByBusinessCode(_ context.Context, code valueobject.BusinessCode) (*itementity.Item, error) {
	if f.item != nil && f.item.BusinessCode == code {
		return f.item, nil
	}
	return nil, itemrepo.ErrNotFound
}
func (f *fakeItems) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*itementity.Item, error) {
	if f.item != nil && f.item.Code == code {
		return f.item, nil
	}
	return nil, itemrepo.ErrNotFound
}

type fakeHeaderRepo struct {
	headers []*entity.BomHeader
	nextID  int64
}

func (r *fakeHeaderRepo) Create(_ context.Context, h *entity.BomHeader) (*entity.BomHeader, error) {
	r.nextID++
	h.ID = r.nextID
	r.headers = append(r.headers, h)
	return h, nil
}
func (r *fakeHeaderRepo) GetByID(_ context.Context, id int64) (*entity.BomHeader, error) {
	for _, h := range r.headers {
		if h.ID == id {
			return h, nil
		}
	}
	return nil, errorsuc.NewNotFoundError("cabeçalho de estrutura não encontrado nesta empresa")
}
func (r *fakeHeaderRepo) ListByItem(_ context.Context, itemCode int64) ([]*entity.BomHeader, error) {
	out := make([]*entity.BomHeader, 0)
	for _, h := range r.headers {
		if h.ItemCode == itemCode {
			out = append(out, h)
		}
	}
	return out, nil
}
func (r *fakeHeaderRepo) UpdateStatus(_ context.Context, id int64, status string) (*entity.BomHeader, error) {
	for _, h := range r.headers {
		if h.ID == id {
			h.Status = status
			return h, nil
		}
	}
	return nil, errorsuc.NewNotFoundError("cabeçalho de estrutura não encontrado nesta empresa")
}
func (r *fakeHeaderRepo) NextVersion(_ context.Context, itemCode int64, mask string) (int32, error) {
	var max int32
	for _, h := range r.headers {
		hMask := ""
		if h.Mask != nil {
			hMask = *h.Mask
		}
		if h.ItemCode == itemCode && hMask == mask && h.Version > max {
			max = h.Version
		}
	}
	return max + 1, nil
}

type fakeAuth struct {
	ports.AuthService
	actor uuid.UUID
	err   error
}

func (a *fakeAuth) UserID(context.Context) (uuid.UUID, error) { return a.actor, a.err }

func newHeaderUC(t *testing.T) (*BomHeaderUseCase, *fakeHeaderRepo, uuid.UUID) {
	t.Helper()
	actor := uuid.New()
	items := &fakeItems{item: &itementity.Item{Code: 5000, BusinessCode: "CH-1000", Name: "Chapa 1000mm"}}
	repo := &fakeHeaderRepo{}
	return New(repo, items, &fakeAuth{actor: actor}), repo, actor
}

// A tela envia o código de negócio do item; o autor vem do JWT, nunca do corpo.
func TestCreate_TextItemCodeAndActorFromToken(t *testing.T) {
	uc, repo, actor := newHeaderUC(t)
	got, err := uc.Create(context.Background(), request.CreateBomHeaderDTO{
		ItemCode: "CH-1000", CreatedBy: uuid.New(),
	})
	if err != nil {
		t.Fatalf("criação rejeitada: %v", err)
	}
	if got.ItemCode != "CH-1000" || got.ItemName != "Chapa 1000mm" {
		t.Fatalf("resposta não devolveu o código de negócio: %+v", got)
	}
	if got.LegacyCode != 5000 {
		t.Fatalf("chave legada = %d, quer 5000", got.LegacyCode)
	}
	if repo.headers[0].CreatedBy != actor {
		t.Fatalf("autor = %v, quer o do JWT (%v)", repo.headers[0].CreatedBy, actor)
	}
	if got.BomType != "MBOM" || got.Status != entity.StatusDraft || got.Version != 1 {
		t.Fatalf("padrões do servidor não aplicados: %+v", got)
	}
	if got.BomTypeLabel != "Estrutura de fabricação" || got.StatusLabel != "Rascunho" {
		t.Fatalf("rótulos PT-BR ausentes: %+v", got)
	}
}

func TestCreate_UnknownItemIsRejectedInPortuguese(t *testing.T) {
	uc, _, _ := newHeaderUC(t)
	_, err := uc.Create(context.Background(), request.CreateBomHeaderDTO{ItemCode: "NAO-EXISTE"})
	var validation *errorsuc.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
}

func TestCreate_InvalidBomTypeIsRejected(t *testing.T) {
	uc, _, _ := newHeaderUC(t)
	_, err := uc.Create(context.Background(), request.CreateBomHeaderDTO{ItemCode: "CH-1000", BomType: "PBOM"})
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
}

func TestCreate_LowercaseBomTypeIsNormalized(t *testing.T) {
	uc, _, _ := newHeaderUC(t)
	got, err := uc.Create(context.Background(), request.CreateBomHeaderDTO{ItemCode: "CH-1000", BomType: " ebom "})
	if err != nil {
		t.Fatalf("tipo em minúsculas rejeitado: %v", err)
	}
	if got.BomType != "EBOM" {
		t.Fatalf("tipo = %q, quer EBOM", got.BomType)
	}
}

func TestCreate_VersionAutoIncrementsPerItem(t *testing.T) {
	uc, _, _ := newHeaderUC(t)
	ctx := context.Background()
	for want := int32(1); want <= 3; want++ {
		got, err := uc.Create(ctx, request.CreateBomHeaderDTO{ItemCode: "CH-1000"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Version != want {
			t.Fatalf("versão = %d, quer %d", got.Version, want)
		}
	}
}

func TestListByItem_UsesBusinessCode(t *testing.T) {
	uc, _, _ := newHeaderUC(t)
	ctx := context.Background()
	if _, err := uc.Create(ctx, request.CreateBomHeaderDTO{ItemCode: "CH-1000"}); err != nil {
		t.Fatal(err)
	}
	list, err := uc.ListByItem(ctx, "CH-1000")
	if err != nil || len(list) != 1 {
		t.Fatalf("listagem = %d err=%v, quer 1", len(list), err)
	}
	if _, err := uc.ListByItem(ctx, "NAO-EXISTE"); err == nil {
		t.Fatal("item inexistente devolveu lista")
	}
}

func TestUpdateStatus_RejectsUnknownStatus(t *testing.T) {
	uc, _, _ := newHeaderUC(t)
	_, err := uc.UpdateStatus(context.Background(), request.UpdateBomHeaderStatusDTO{ID: 1, Status: "LIBERADO"})
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
}

func TestUpdateStatus_NormalizesCase(t *testing.T) {
	uc, _, _ := newHeaderUC(t)
	ctx := context.Background()
	created, err := uc.Create(ctx, request.CreateBomHeaderDTO{ItemCode: "CH-1000"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := uc.UpdateStatus(ctx, request.UpdateBomHeaderStatusDTO{ID: created.ID, Status: "approved"})
	if err != nil {
		t.Fatalf("situação em minúsculas rejeitada: %v", err)
	}
	if got.Status != entity.StatusApproved || got.StatusLabel != "Aprovado" {
		t.Fatalf("situação = %q / %q", got.Status, got.StatusLabel)
	}
}
