package sales_quotation_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quotationrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
)

type maintenanceQuotationRepo struct {
	quotationrepo.SalesQuotationRepository
	reasonErr error
	reset     bool
}

func (r *maintenanceQuotationRepo) ResetParameters(context.Context) (*entity.Parameters, error) {
	r.reset = true
	return entity.DefaultParameters(1), nil
}
func (r *maintenanceQuotationRepo) SetCommissionPatternActive(context.Context, int64, bool) error {
	return nil
}
func (r *maintenanceQuotationRepo) SetCancellationReasonActive(context.Context, int64, bool) error {
	return r.reasonErr
}

func TestQuotationConfigurationMaintenance(t *testing.T) {
	repo := &maintenanceQuotationRepo{}
	uc := UseCase{Repo: repo, Auth: quotationRepresentativeAuth{}}
	if _, err := uc.ResetParameters(context.Background()); err != nil || !repo.reset {
		t.Fatalf("reset falhou: %v", err)
	}
	repo.reasonErr = quotationrepo.ErrConfigurationReferenced
	if err := uc.SetCancellationReasonActive(context.Background(), 7, false); err == nil || !strings.Contains(err.Error(), "descancelado") {
		t.Fatalf("referência não protegida: %v", err)
	}
}
