package item_classification_uc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

// fakeClassRepo is an in-memory stand-in keyed the same way the database is:
// masks by business code, classifications by (mask id, code).
type fakeClassRepo struct {
	masks   []*entity.ItemClassificationMask
	classes []*entity.ItemClassification
	nextID  int64
}

func newFakeClassRepo() *fakeClassRepo {
	return &fakeClassRepo{
		// Note the gap between id and code: the screen only knows the code.
		masks:  []*entity.ItemClassificationMask{{ID: 77, Code: 1, Mask: "99.99.99", Description: "Mercadológica", IsActive: true}},
		nextID: 100,
	}
}

func (r *fakeClassRepo) CreateClassificationMask(_ context.Context, m *entity.ItemClassificationMask) (*entity.ItemClassificationMask, error) {
	r.nextID++
	m.ID, m.Code = r.nextID, int64(len(r.masks)+1)
	r.masks = append(r.masks, m)
	return m, nil
}
func (r *fakeClassRepo) UpdateClassificationMask(_ context.Context, m *entity.ItemClassificationMask) (*entity.ItemClassificationMask, error) {
	for _, existing := range r.masks {
		if existing.ID == m.ID {
			existing.Description, existing.IsActive = m.Description, m.IsActive
			return existing, nil
		}
	}
	return nil, errors.New("no rows in result set")
}
func (r *fakeClassRepo) GetClassificationMaskByCode(_ context.Context, code int64) (*entity.ItemClassificationMask, error) {
	for _, m := range r.masks {
		if m.Code == code {
			return m, nil
		}
	}
	return nil, errors.New("no rows in result set")
}
func (r *fakeClassRepo) ListClassificationMasks(_ context.Context, onlyActive bool) ([]*entity.ItemClassificationMask, error) {
	out := make([]*entity.ItemClassificationMask, 0, len(r.masks))
	for _, m := range r.masks {
		if !onlyActive || m.IsActive {
			out = append(out, m)
		}
	}
	return out, nil
}
func (r *fakeClassRepo) NextClassificationMaskCode(context.Context) (int64, error) {
	return int64(len(r.masks) + 1), nil
}
func (r *fakeClassRepo) CreateItemClassification(_ context.Context, c *entity.ItemClassification) (*entity.ItemClassification, error) {
	r.nextID++
	c.ID = r.nextID
	r.classes = append(r.classes, c)
	return c, nil
}
func (r *fakeClassRepo) UpdateItemClassification(_ context.Context, c *entity.ItemClassification) (*entity.ItemClassification, error) {
	for _, existing := range r.classes {
		if existing.ID == c.ID {
			existing.Description, existing.IsActive = c.Description, c.IsActive
			return existing, nil
		}
	}
	return nil, errors.New("no rows in result set")
}
func (r *fakeClassRepo) GetItemClassificationByCode(_ context.Context, code string, maskCode int64) (*entity.ItemClassification, error) {
	var maskID int64
	for _, m := range r.masks {
		if m.Code == maskCode {
			maskID = m.ID
		}
	}
	for _, c := range r.classes {
		if c.Code == code && c.MaskID == maskID {
			return c, nil
		}
	}
	return nil, errors.New("no rows in result set")
}
func (r *fakeClassRepo) ListItemClassificationsByMask(_ context.Context, maskID int64, onlyActive bool) ([]*entity.ItemClassification, error) {
	out := make([]*entity.ItemClassification, 0)
	for _, c := range r.classes {
		if c.MaskID == maskID && (!onlyActive || c.IsActive) {
			out = append(out, c)
		}
	}
	return out, nil
}
func (r *fakeClassRepo) ListItemClassificationChildren(_ context.Context, parentID int64, onlyActive bool) ([]*entity.ItemClassification, error) {
	out := make([]*entity.ItemClassification, 0)
	for _, c := range r.classes {
		if c.ParentID != nil && *c.ParentID == parentID && (!onlyActive || c.IsActive) {
			out = append(out, c)
		}
	}
	return out, nil
}

func newTestUC(t *testing.T) (*ItemClassificationUseCase, *fakeClassRepo) {
	t.Helper()
	repo := newFakeClassRepo()
	return New(repo), repo
}

// A tela envia parent_code:"" ao cadastrar uma raiz; isso não pode virar busca
// por um pai de código vazio.
func TestCreateClassification_EmptyParentCodeMeansRoot(t *testing.T) {
	uc, _ := newTestUC(t)
	got, err := uc.CreateClassification(context.Background(), request.CreateItemClassificationDTO{
		Code: "10", MaskCode: 1, Description: "Matéria-prima", ParentCode: strPtr("  "),
	})
	if err != nil {
		t.Fatalf("raiz rejeitada: %v", err)
	}
	if got.ParentID != nil || got.ParentCode != "" {
		t.Fatalf("raiz recebeu pai: parent_id=%v parent_code=%q", got.ParentID, got.ParentCode)
	}
	if got.MaskCode != 1 || got.Mask != "99.99.99" {
		t.Fatalf("resposta sem a máscara: %+v", got)
	}
}

func TestCreateClassification_ParentNotFoundIsNotFoundInPortuguese(t *testing.T) {
	uc, _ := newTestUC(t)
	_, err := uc.CreateClassification(context.Background(), request.CreateItemClassificationDTO{
		Code: "10.20", MaskCode: 1, Description: "Aço", ParentCode: strPtr("10"),
	})
	if _, ok := errorsuc.AsNotFound(err); !ok {
		t.Fatalf("esperado NotFoundError, veio %T (%v)", err, err)
	}
	if !strings.Contains(err.Error(), "classificação pai") {
		t.Fatalf("mensagem não está em PT-BR: %q", err.Error())
	}
}

func TestCreateClassification_UnknownMaskIsNotFound(t *testing.T) {
	uc, _ := newTestUC(t)
	_, err := uc.CreateClassification(context.Background(), request.CreateItemClassificationDTO{
		Code: "10", MaskCode: 999, Description: "Aço",
	})
	if _, ok := errorsuc.AsNotFound(err); !ok {
		t.Fatalf("esperado NotFoundError para máscara inexistente, veio %T (%v)", err, err)
	}
}

func TestCreateClassification_DuplicateIsConflict(t *testing.T) {
	uc, _ := newTestUC(t)
	ctx := context.Background()
	dto := request.CreateItemClassificationDTO{Code: "10", MaskCode: 1, Description: "Matéria-prima"}
	if _, err := uc.CreateClassification(ctx, dto); err != nil {
		t.Fatal(err)
	}
	_, err := uc.CreateClassification(ctx, dto)
	if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("esperado ConflictError, veio %T (%v)", err, err)
	}
}

// ListByMaskCode recebe o código da máscara (1), não o id interno (77) — era
// essa troca que devolvia 500 ao abrir a máscara na tela.
func TestListByMaskCode_UsesBusinessCodeNotInternalID(t *testing.T) {
	uc, _ := newTestUC(t)
	ctx := context.Background()
	if _, err := uc.CreateClassification(ctx, request.CreateItemClassificationDTO{Code: "10", MaskCode: 1, Description: "Matéria-prima"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateClassification(ctx, request.CreateItemClassificationDTO{Code: "10.20", MaskCode: 1, Description: "Aço", ParentCode: strPtr("10")}); err != nil {
		t.Fatal(err)
	}
	list, err := uc.ListByMaskCode(ctx, 1, true)
	if err != nil {
		t.Fatalf("listagem por código da máscara falhou: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("classificações listadas = %d, quer 2", len(list))
	}
	child := list[1]
	if child.ParentCode != "10" {
		t.Fatalf("parent_code do filho = %q, quer \"10\"", child.ParentCode)
	}
	if child.FullDescription != "Matéria-prima > Aço" {
		t.Fatalf("descrição hierárquica = %q", child.FullDescription)
	}
	if _, err := uc.ListByMaskCode(ctx, 77, true); err == nil {
		t.Fatal("o id interno da máscara não deveria ser aceito como código")
	}
}

// A tela altera pelo código; o id interno não trafega no payload.
func TestUpdateClassification_ByCodeAndMask(t *testing.T) {
	uc, _ := newTestUC(t)
	ctx := context.Background()
	if _, err := uc.CreateClassification(ctx, request.CreateItemClassificationDTO{Code: "10", MaskCode: 1, Description: "Matéria-prima"}); err != nil {
		t.Fatal(err)
	}
	got, err := uc.UpdateClassification(ctx, request.UpdateItemClassificationDTO{
		Code: "10", MaskCode: 1, Description: "Matérias-primas", IsActive: &ativo,
	})
	if err != nil {
		t.Fatalf("alteração por código falhou: %v", err)
	}
	if got.Description != "Matérias-primas" {
		t.Fatalf("descrição não persistida: %+v", got)
	}
}

// `IsActive` é ponteiro: omitido, a atualização preserva a situação atual.
var ativo = true

func TestUpdateMask_ByCode(t *testing.T) {
	uc, repo := newTestUC(t)
	got, err := uc.UpdateMask(context.Background(), request.UpdateClassificationMaskDTO{
		Code: 1, Description: "Mercadológica revisada", IsActive: &ativo,
	})
	if err != nil {
		t.Fatalf("alteração de máscara por código falhou: %v", err)
	}
	if got.Description != "Mercadológica revisada" || repo.masks[0].Description != "Mercadológica revisada" {
		t.Fatalf("descrição da máscara não persistida: %+v", got)
	}
}

func TestGetByCode_UnknownClassificationIsNotFound(t *testing.T) {
	uc, _ := newTestUC(t)
	_, err := uc.GetByCode(context.Background(), "99", 1)
	if _, ok := errorsuc.AsNotFound(err); !ok {
		t.Fatalf("esperado NotFoundError, veio %T (%v)", err, err)
	}
}

func strPtr(s string) *string { return &s }
