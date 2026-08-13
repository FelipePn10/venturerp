package fiscal_classification_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity"
	"github.com/google/uuid"
)

type fiscalAuth struct{ ports.AuthService }

func (fiscalAuth) EnterpriseID(context.Context) (int64, error) { return 19, nil }

type fiscalRepoStub struct {
	saved      *entity.FiscalClassification
	nextTenant int64
}

func (r *fiscalRepoStub) Create(_ context.Context, c *entity.FiscalClassification) (*entity.FiscalClassification, error) {
	r.saved = c
	return c, nil
}
func (*fiscalRepoStub) Update(context.Context, *entity.FiscalClassification) (*entity.FiscalClassification, error) {
	return nil, nil
}
func (*fiscalRepoStub) GetByCode(context.Context, int64, int64) (*entity.FiscalClassification, error) {
	return nil, nil
}
func (*fiscalRepoStub) List(context.Context, int64, bool) ([]*entity.FiscalClassification, error) {
	return nil, nil
}
func (r *fiscalRepoStub) NextCode(_ context.Context, e int64) (int64, error) {
	r.nextTenant = e
	return 7, nil
}
func (*fiscalRepoStub) AddLanguage(context.Context, *entity.FiscalClassificationLanguage) (*entity.FiscalClassificationLanguage, error) {
	return nil, nil
}
func (*fiscalRepoStub) ListLanguages(context.Context, int64) ([]*entity.FiscalClassificationLanguage, error) {
	return nil, nil
}
func (*fiscalRepoStub) DeleteLanguage(context.Context, int64) error { return nil }
func (*fiscalRepoStub) AddExportAttribute(context.Context, *entity.FiscalClassificationExportAttribute) (*entity.FiscalClassificationExportAttribute, error) {
	return nil, nil
}
func (*fiscalRepoStub) ListExportAttributes(context.Context, int64) ([]*entity.FiscalClassificationExportAttribute, error) {
	return nil, nil
}
func (*fiscalRepoStub) DeleteExportAttribute(context.Context, int64) error { return nil }

func TestCreateFiscalClassificationUsesAuthenticatedEnterpriseAndDefaults(t *testing.T) {
	repo := &fiscalRepoStub{}
	uc := NewFiscalClassificationUseCase(repo, fiscalAuth{})
	origin := "0"
	ncm := "73269090"
	result, err := uc.Create(context.Background(), request.CreateFiscalClassificationDTO{FiscalClassificationFields: request.FiscalClassificationFields{Description: "COMPONENTES METALICOS", NCM: &ncm, DefaultOrigin: &origin, DefaultICMSRate: 18, DefaultCalculatePISCOFINS: true}, CreatedBy: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Code != 7 || repo.nextTenant != 19 || repo.saved.EnterpriseID != 19 || repo.saved.DefaultICMSRate != 18 || !repo.saved.DefaultCalculatePISCOFINS {
		t.Fatalf("classificacao incorreta: %#v", repo.saved)
	}
}
