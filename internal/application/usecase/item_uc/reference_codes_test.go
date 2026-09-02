package item_uc

import (
	"context"
	"strings"
	"testing"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

// refItems é um repositório de itens mínimo, indexado pelos dois códigos.
type refItems struct{ items []*entity.Item }

func (r *refItems) FindItemByBusinessCode(_ context.Context, code valueobject.BusinessCode) (*entity.Item, error) {
	for _, it := range r.items {
		if it.BusinessCode == code {
			return it, nil
		}
	}
	return nil, itemrepo.ErrNotFound
}

func (r *refItems) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*entity.Item, error) {
	for _, it := range r.items {
		if it.Code == code {
			return it, nil
		}
	}
	return nil, itemrepo.ErrNotFound
}

func newRefItems() *refItems {
	return &refItems{items: []*entity.Item{
		{Code: 100, BusinessCode: "CX-PAPELAO", Name: "Caixa de papelão"},
		{Code: 200, BusinessCode: "MOD-BASE", Name: "Modelo base"},
		{Code: 300, BusinessCode: "PROD-01", Name: "Produto 01"},
	}}
}

// A tela envia os códigos de negócio; o servidor grava as chaves legadas.
func TestResolveReferenceCodes_TranslatesBusinessCodes(t *testing.T) {
	item := &entity.Item{Code: 300, BusinessCode: "PROD-01"}
	item.Engineering.ItemBaseBusinessCode = "MOD-BASE"
	item.Commercial.PackagingItemBusinessCode = "CX-PAPELAO"

	if err := resolveReferenceCodes(context.Background(), newRefItems(), item); err != nil {
		t.Fatalf("resolução recusada: %v", err)
	}
	if item.Engineering.ItemBaseCod == nil || *item.Engineering.ItemBaseCod != 200 {
		t.Fatalf("item-base = %v, quer 200", item.Engineering.ItemBaseCod)
	}
	if item.Commercial.PackagingItemCode == nil || *item.Commercial.PackagingItemCode != 100 {
		t.Fatalf("embalagem = %v, quer 100", item.Commercial.PackagingItemCode)
	}
}

func TestResolveReferenceCodes_UnknownReferenceIsPortuguese(t *testing.T) {
	item := &entity.Item{Code: 300, BusinessCode: "PROD-01"}
	item.Commercial.PackagingItemBusinessCode = "NAO-EXISTE"
	err := resolveReferenceCodes(context.Background(), newRefItems(), item)
	v, ok := errorsuc.AsValidation(err)
	if !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
	if !strings.Contains(v.Error(), "item de embalagem") {
		t.Fatalf("mensagem = %q", v.Error())
	}
}

func TestResolveReferenceCodes_RejectsSelfReference(t *testing.T) {
	item := &entity.Item{Code: 300, BusinessCode: "PROD-01"}
	item.Commercial.PackagingItemBusinessCode = "PROD-01"
	if _, ok := errorsuc.AsValidation(resolveReferenceCodes(context.Background(), newRefItems(), item)); !ok {
		t.Fatal("item aceito como a própria embalagem")
	}

	other := &entity.Item{Code: 300, BusinessCode: "PROD-01"}
	other.Engineering.ItemBaseBusinessCode = "PROD-01"
	if _, ok := errorsuc.AsValidation(resolveReferenceCodes(context.Background(), newRefItems(), other)); !ok {
		t.Fatal("item aceito como o próprio item-base")
	}
}

func TestResolveReferenceCodes_EmptyReferencesAreLeftAlone(t *testing.T) {
	item := &entity.Item{Code: 300, BusinessCode: "PROD-01"}
	if err := resolveReferenceCodes(context.Background(), newRefItems(), item); err != nil {
		t.Fatalf("item sem referências recusado: %v", err)
	}
	if item.Engineering.ItemBaseCod != nil || item.Commercial.PackagingItemCode != nil {
		t.Fatalf("referências preenchidas indevidamente: %+v", item)
	}
}

// Ao reabrir o item, as referências voltam como código de negócio — é o que
// permite retomar um cadastro parcial de onde parou.
func TestFillReferenceBusinessCodes_TranslatesBack(t *testing.T) {
	base, packaging := 200, int64(100)
	item := &entity.Item{Code: 300, BusinessCode: "PROD-01"}
	item.Engineering.ItemBaseCod = &base
	item.Commercial.PackagingItemCode = &packaging

	fillReferenceBusinessCodes(context.Background(), newRefItems(), item)
	if item.Engineering.ItemBaseBusinessCode != "MOD-BASE" {
		t.Fatalf("item-base = %q, quer MOD-BASE", item.Engineering.ItemBaseBusinessCode)
	}
	if item.Commercial.PackagingItemBusinessCode != "CX-PAPELAO" {
		t.Fatalf("embalagem = %q, quer CX-PAPELAO", item.Commercial.PackagingItemBusinessCode)
	}
}

func TestFillReferenceBusinessCodes_MissingReferenceIsTolerated(t *testing.T) {
	orphan := 999
	item := &entity.Item{Code: 300, BusinessCode: "PROD-01"}
	item.Engineering.ItemBaseCod = &orphan
	fillReferenceBusinessCodes(context.Background(), newRefItems(), item)
	if item.Engineering.ItemBaseBusinessCode != "" {
		t.Fatalf("referência órfã preenchida: %q", item.Engineering.ItemBaseBusinessCode)
	}
}

// fiscalCatalog imita o cadastro canônico de classificações fiscais.
type fiscalCatalog struct {
	active   map[int64]bool
	tenantID int64
}

func (c *fiscalCatalog) GetByCode(_ context.Context, enterpriseID, code int64) (*fiscalentity.FiscalClassification, error) {
	if enterpriseID != c.tenantID {
		return nil, itemrepo.ErrNotFound
	}
	isActive, ok := c.active[code]
	if !ok {
		return nil, itemrepo.ErrNotFound
	}
	return &fiscalentity.FiscalClassification{Code: code, IsActive: isActive}, nil
}

func fiscalItem(sale, purchase string) *entity.Item {
	item := &entity.Item{Code: 300, BusinessCode: "PROD-01"}
	if sale != "" {
		item.Accounting.SaleFiscalClassificationCode = &sale
	}
	if purchase != "" {
		item.Accounting.PurchaseFiscalClassificationCode = &purchase
	}
	return item
}

// As classificações fiscais só valem se existirem no catálogo canônico.
func TestValidateFiscalClassifications_AcceptsCatalogCodes(t *testing.T) {
	catalog := &fiscalCatalog{tenantID: 7, active: map[int64]bool{10: true, 20: true}}
	if err := validateFiscalClassifications(context.Background(), catalog, 7, fiscalItem("10", "20")); err != nil {
		t.Fatalf("classificações do catálogo recusadas: %v", err)
	}
}

func TestValidateFiscalClassifications_RejectsUnknownCode(t *testing.T) {
	catalog := &fiscalCatalog{tenantID: 7, active: map[int64]bool{10: true}}
	err := validateFiscalClassifications(context.Background(), catalog, 7, fiscalItem("10", "99"))
	v, ok := errorsuc.AsValidation(err)
	if !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
	if !strings.Contains(v.Error(), "compra") {
		t.Fatalf("mensagem = %q, esperava citar a classificação de compra", v.Error())
	}
}

func TestValidateFiscalClassifications_RejectsInactiveCode(t *testing.T) {
	catalog := &fiscalCatalog{tenantID: 7, active: map[int64]bool{10: false}}
	err := validateFiscalClassifications(context.Background(), catalog, 7, fiscalItem("10", ""))
	v, ok := errorsuc.AsValidation(err)
	if !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
	if !strings.Contains(v.Error(), "inativa") {
		t.Fatalf("mensagem = %q", v.Error())
	}
}

func TestValidateFiscalClassifications_RejectsNonNumericCode(t *testing.T) {
	catalog := &fiscalCatalog{tenantID: 7, active: map[int64]bool{10: true}}
	if _, ok := errorsuc.AsValidation(validateFiscalClassifications(context.Background(), catalog, 7, fiscalItem("ABC", ""))); !ok {
		t.Fatal("código não numérico aceito")
	}
}

// Classificações em branco continuam opcionais, e sem catálogo nada é barrado.
func TestValidateFiscalClassifications_OptionalAndNilSafe(t *testing.T) {
	catalog := &fiscalCatalog{tenantID: 7, active: map[int64]bool{10: true}}
	if err := validateFiscalClassifications(context.Background(), catalog, 7, fiscalItem("", "")); err != nil {
		t.Fatalf("item sem classificações recusado: %v", err)
	}
	blank := "   "
	item := &entity.Item{Code: 300}
	item.Accounting.SaleFiscalClassificationCode = &blank
	if err := validateFiscalClassifications(context.Background(), catalog, 7, item); err != nil {
		t.Fatalf("classificação em branco recusada: %v", err)
	}
	if err := validateFiscalClassifications(context.Background(), nil, 7, fiscalItem("999", "")); err != nil {
		t.Fatalf("sem catálogo nada deveria ser barrado: %v", err)
	}
}

// O catálogo é por empresa: o código de outro tenant não vale.
func TestValidateFiscalClassifications_IsTenantScoped(t *testing.T) {
	catalog := &fiscalCatalog{tenantID: 7, active: map[int64]bool{10: true}}
	if _, ok := errorsuc.AsValidation(validateFiscalClassifications(context.Background(), catalog, 8, fiscalItem("10", ""))); !ok {
		t.Fatal("classificação de outra empresa aceita")
	}
}
