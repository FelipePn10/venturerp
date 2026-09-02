package itemresolution

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type resolverRepository struct {
	byBusiness map[valueobject.BusinessCode]*itementity.Item
	byLegacy   map[valueobject.ItemCode]*itementity.Item
	withMasks  []itementity.ItemWithMasks
}

func (r resolverRepository) ListAllWithMasks(context.Context) ([]itementity.ItemWithMasks, error) {
	return r.withMasks, nil
}

func (r resolverRepository) FindItemByBusinessCode(_ context.Context, code valueobject.BusinessCode) (*itementity.Item, error) {
	if item := r.byBusiness[code]; item != nil {
		return item, nil
	}
	return nil, itemrepo.ErrNotFound
}

func (r resolverRepository) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*itementity.Item, error) {
	if item := r.byLegacy[code]; item != nil {
		return item, nil
	}
	return nil, itemrepo.ErrNotFound
}

func TestResolvePrefersBusinessCodeAndSupportsLegacyFallback(t *testing.T) {
	business := &itementity.Item{Code: 91, BusinessCode: "0007"}
	legacy := &itementity.Item{Code: 7, BusinessCode: "LEGACY-7"}
	repo := resolverRepository{
		byBusiness: map[valueobject.BusinessCode]*itementity.Item{"0007": business},
		byLegacy:   map[valueobject.ItemCode]*itementity.Item{7: legacy},
	}
	got, err := Resolve(context.Background(), repo, request.TextCode("0007"))
	if err != nil || got.Code != 91 {
		t.Fatalf("código comercial ambíguo: item=%v err=%v", got, err)
	}
	got, err = Resolve(context.Background(), repo, request.TextCode("7"))
	if err != nil || got.Code != 7 {
		t.Fatalf("fallback legado: item=%v err=%v", got, err)
	}
}

func TestResolveRejectsMissingTenantItem(t *testing.T) {
	_, err := Resolve(context.Background(), resolverRepository{}, request.TextCode("OUTRO-TENANT"))
	if err == nil {
		t.Fatal("esperado erro para item inexistente no tenant")
	}
}

func TestValidateMaskUsesTenantCatalog(t *testing.T) {
	repo := resolverRepository{withMasks: []itementity.ItemWithMasks{{
		Item: &itementity.Item{Code: 10}, Masks: []itementity.MaskSummary{{Mask: "AZUL-01"}},
	}}}
	if err := ValidateMask(context.Background(), repo, 10, "azul-01"); err != nil {
		t.Fatalf("máscara cadastrada rejeitada: %v", err)
	}
	if err := ValidateMask(context.Background(), repo, 10, "OUTRA"); err == nil {
		t.Fatal("máscara inexistente deveria ser rejeitada")
	}
}
