package item_supplier_uc

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository"
	"strings"
	"time"
)

type ItemSupplierUseCase struct {
	repo repository.ItemSupplierRepository
	auth ports.AuthService
}

func NewItemSupplierUseCase(repo repository.ItemSupplierRepository, auth ports.AuthService) *ItemSupplierUseCase {
	return &ItemSupplierUseCase{repo: repo, auth: auth}
}
func parseOptionalDate(s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	x, err := time.Parse("2006-01-02", strings.TrimSpace(*s))
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: use YYYY-MM-DD", *s)
	}
	return &x, nil
}
func (uc *ItemSupplierUseCase) Upsert(ctx context.Context, d request.UpsertItemPreferredSupplierDTO) (*response.ItemPreferredSupplierResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	by, err := uc.auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := entity.NewItemPreferredSupplier(e, d.ItemCode, d.SupplierCode, d.Mask, d.Ranking, by)
	if err != nil {
		return nil, err
	}
	s.SupplierItemCode = d.SupplierItemCode
	s.SupplierDescription = d.SupplierDescription
	s.UOM = d.UOM
	s.XMLUOM = d.XMLUOM
	s.ConversionFactor = d.ConversionFactor
	s.PackageQuantity = d.PackageQuantity
	// ranking=1 was the legacy preferred-supplier contract; keep it compatible
	// while exposing the explicit flag required by the richer register.
	s.IsPreferred = d.IsPreferred || d.Ranking == 1
	s.ClassificationID = d.ClassificationID
	s.ClassificationGrade = d.ClassificationGrade
	s.DirectBilling = d.DirectBilling
	s.ThirdPartyOrder = d.ThirdPartyOrder
	s.IgnoreAvgCostAddition = d.IgnoreAvgCostAddition
	s.Ecommerce = d.Ecommerce
	s.Barcode = d.Barcode
	s.Notes = d.Notes
	s.LeadTimeDays = d.LeadTimeDays
	if s.ClassificationDate, err = parseOptionalDate(d.ClassificationDate); err != nil {
		return nil, err
	}
	if s.ValidUntil, err = parseOptionalDate(d.ValidUntil); err != nil {
		return nil, err
	}
	if s.ConversionFactor != nil {
		ok, err := uc.repo.ItemAllowsConversionFactor(ctx, s.ItemCode)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("conversion_factor is allowed only for generic XML-import items")
		}
	}
	if err = s.Validate(); err != nil {
		return nil, err
	}
	saved, err := uc.repo.Upsert(ctx, s)
	if err != nil {
		return nil, err
	}
	return toItemPreferredSupplierResponse(saved), nil
}
func (uc *ItemSupplierUseCase) ListByItem(ctx context.Context, item int64) ([]*response.ItemPreferredSupplierResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.ListByItem(ctx, e, item)
	if err != nil {
		return nil, err
	}
	return toItemPreferredSupplierResponses(x), nil
}
func (uc *ItemSupplierUseCase) ListBySupplier(ctx context.Context, supplier int64) ([]*response.ItemPreferredSupplierResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.ListBySupplier(ctx, e, supplier)
	if err != nil {
		return nil, err
	}
	return toItemPreferredSupplierResponses(x), nil
}
func (uc *ItemSupplierUseCase) Delete(ctx context.Context, id int64) error {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return err
	}
	return uc.repo.Delete(ctx, e, id)
}
func (uc *ItemSupplierUseCase) GetPreferredSupplier(ctx context.Context, item int64) (int64, bool, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return 0, false, err
	}
	s, err := uc.repo.GetPreferred(ctx, e, item)
	if err != nil || s == nil {
		return 0, false, nil
	}
	return s.SupplierCode, true, nil
}
func (uc *ItemSupplierUseCase) CreateQualityReport(ctx context.Context, link int64, d request.CreateItemSupplierQualityReportDTO) (*response.ItemSupplierQualityReportResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	by, err := uc.auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	on, err := time.Parse("2006-01-02", d.RegisteredOn)
	if err != nil {
		return nil, fmt.Errorf("registered_on must use YYYY-MM-DD")
	}
	q, err := entity.NewQualityReport(e, link, on, d.Status, by)
	if err != nil {
		return nil, err
	}
	q.FileName = d.FileName
	q.ContentType = d.ContentType
	q.Content = d.Content
	if len(q.Content) > 10*1024*1024 {
		return nil, fmt.Errorf("quality report attachment exceeds 10 MiB")
	}
	q.Notes = d.Notes
	saved, err := uc.repo.CreateQualityReport(ctx, q)
	if err != nil {
		return nil, err
	}
	return toQualityResponse(saved), nil
}
func (uc *ItemSupplierUseCase) ListQualityReports(ctx context.Context, link int64) ([]response.ItemSupplierQualityReportResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.ListQualityReports(ctx, e, link)
	if err != nil {
		return nil, err
	}
	out := make([]response.ItemSupplierQualityReportResponse, 0, len(x))
	for _, q := range x {
		out = append(out, *toQualityResponse(q))
	}
	return out, nil
}
