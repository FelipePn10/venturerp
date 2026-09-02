package sales_quotation_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	representativeentity "github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
	quotationentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quotationrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
	"github.com/google/uuid"
)

type quotationRepresentativeAuth struct{ ports.AuthService }

func (quotationRepresentativeAuth) CanCreateSalesOrder(context.Context) bool    { return true }
func (quotationRepresentativeAuth) CanUpdateSalesOrder(context.Context) bool    { return true }
func (quotationRepresentativeAuth) EnterpriseID(context.Context) (int64, error) { return 1, nil }
func (quotationRepresentativeAuth) UserID(context.Context) (uuid.UUID, error)   { return uuid.New(), nil }

type quotationRepresentativeRepo struct {
	quotationrepo.SalesQuotationRepository
	quotation *quotationentity.SalesQuotation
}

func (r quotationRepresentativeRepo) GetByCode(context.Context, int64) (*quotationentity.SalesQuotation, error) {
	return r.quotation, nil
}

type inactiveQuotationRepresentativeRepo struct{}

func (inactiveQuotationRepresentativeRepo) Get(_ context.Context, code int64) (*representativeentity.Representative, error) {
	return &representativeentity.Representative{Code: code, IsActive: false}, nil
}

func TestQuotationCreateUpdateAndConversionRejectInactiveRepresentative(t *testing.T) {
	code := int64(91)
	uc := &UseCase{Repo: quotationRepresentativeRepo{}, Auth: quotationRepresentativeAuth{}, Representatives: inactiveQuotationRepresentativeRepo{}}
	if _, err := uc.Create(context.Background(), request.CreateSalesQuotationDTO{EnterpriseCode: 1, RepresentativeCode: &code}); err == nil || !strings.Contains(err.Error(), "inativo") {
		t.Fatalf("criação deveria rejeitar representante inativo: %v", err)
	}
	if _, err := uc.Update(context.Background(), request.UpdateSalesQuotationDTO{Code: 1, RepresentativeCode: &code}); err == nil || !strings.Contains(err.Error(), "inativo") {
		t.Fatalf("alteração deveria rejeitar representante inativo: %v", err)
	}
	uc.Repo = quotationRepresentativeRepo{quotation: &quotationentity.SalesQuotation{Code: 1, RepresentativeCode: &code}}
	if _, err := (&ConvertUseCase{Quotes: uc}).Execute(context.Background(), request.ConvertSalesQuotationDTO{Code: 1}); err == nil || !strings.Contains(err.Error(), "inativo") {
		t.Fatalf("conversão deveria rejeitar representante inativo: %v", err)
	}
}
