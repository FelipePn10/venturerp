package sales_quotation_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type UseCase struct {
	Repo repository.SalesQuotationRepository
	Auth ports.AuthService
}

func (uc *UseCase) Create(ctx context.Context, dto request.CreateSalesQuotationDTO) (*response.SalesQuotationResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.EnterpriseCode == 0 {
		return nil, errorsuc.NewValidationError("enterprise_code is required")
	}
	number, err := uc.Repo.NextQuotationNumber(ctx, dto.EnterpriseCode)
	if err != nil {
		return nil, err
	}
	status := entity.SalesQuotationStatusERPBudget
	if dto.Status != "" {
		status = entity.SalesQuotationStatus(dto.Status)
	}
	if !validStatus(status) {
		return nil, errorsuc.NewValidationError("invalid quotation status")
	}
	quotationType := entity.SalesQuotationTypeSale
	if dto.QuotationType != "" {
		quotationType = entity.SalesQuotationType(dto.QuotationType)
	}
	if !validType(quotationType) {
		return nil, errorsuc.NewValidationError("invalid quotation_type")
	}
	releaseStatus := entity.SalesQuotationReleaseOK
	if dto.ReleaseStatus != "" {
		releaseStatus = entity.SalesQuotationReleaseStatus(dto.ReleaseStatus)
	}
	if !validReleaseStatus(releaseStatus) {
		return nil, errorsuc.NewValidationError("invalid release_status")
	}
	currency := "BRL"
	if dto.CurrencyCode != "" {
		currency = dto.CurrencyCode
	}
	emissionDate := datetime.ParseDateOrDefault(dto.EmissionDate, time.Now())
	q := &entity.SalesQuotation{
		QuotationNumber:        number,
		EnterpriseCode:         dto.EnterpriseCode,
		Status:                 status,
		QuotationType:          quotationType,
		EmissionDate:           emissionDate,
		DigitDate:              datetime.ParseDateOrDefault(dto.DigitDate, emissionDate),
		ValidUntil:             datetime.ParseDatePtr(dto.ValidUntil),
		DeliveryDate:           datetime.ParseDatePtr(dto.DeliveryDate),
		DeliveryDateFirm:       dto.DeliveryDateFirm,
		PurchaseOrderNumber:    dto.PurchaseOrderNumber,
		CustomerCode:           dto.CustomerCode,
		BillingAddressCode:     dto.BillingAddressCode,
		ShippingAddressCode:    dto.ShippingAddressCode,
		RepresentativeCode:     dto.RepresentativeCode,
		SalesDivisionCode:      dto.SalesDivisionCode,
		PriceTableCode:         dto.PriceTableCode,
		PaymentTermCode:        dto.PaymentTermCode,
		CurrencyCode:           currency,
		ProbabilityPct:         dto.ProbabilityPct,
		CommissionPct:          dto.CommissionPct,
		IsNFCe:                 dto.IsNFCe,
		Street:                 dto.Street,
		StreetNumber:           dto.StreetNumber,
		ForeignDocument:        dto.ForeignDocument,
		ReleaseStatus:          releaseStatus,
		CommercialBlocked:      dto.CommercialBlocked,
		CommercialBlockReason:  dto.CommercialBlockReason,
		CarrierCode:            dto.CarrierCode,
		FreightType:            dto.FreightType,
		VerifyFreight:          dto.VerifyFreight,
		FreightValue:           dto.FreightValue,
		RedeliveryFreightValue: dto.RedeliveryFreightValue,
		InsuranceValue:         dto.InsuranceValue,
		DiscountValue:          dto.DiscountValue,
		SurchargeValue:         dto.SurchargeValue,
		RetainedTaxValue:       dto.RetainedTaxValue,
		DeliveryAuthorization:  dto.DeliveryAuthorization,
		Notes:                  dto.Notes,
		ObsCustomer:            dto.ObsCustomer,
		CreatedBy:              dto.CreatedBy,
	}
	created, err := uc.Repo.Create(ctx, q)
	if err != nil {
		return nil, err
	}
	return toResponse(created), nil
}

func (uc *UseCase) Update(ctx context.Context, dto request.UpdateSalesQuotationDTO) (*response.SalesQuotationResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	current, err := uc.Repo.GetByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	status := current.Status
	if dto.Status != "" {
		status = entity.SalesQuotationStatus(dto.Status)
	}
	if !validStatus(status) {
		return nil, errorsuc.NewValidationError("invalid quotation status")
	}
	quotationType := current.QuotationType
	if dto.QuotationType != "" {
		quotationType = entity.SalesQuotationType(dto.QuotationType)
	}
	if !validType(quotationType) {
		return nil, errorsuc.NewValidationError("invalid quotation_type")
	}
	releaseStatus := current.ReleaseStatus
	if dto.ReleaseStatus != "" {
		releaseStatus = entity.SalesQuotationReleaseStatus(dto.ReleaseStatus)
	}
	if !validReleaseStatus(releaseStatus) {
		return nil, errorsuc.NewValidationError("invalid release_status")
	}
	current.Status = status
	current.QuotationType = quotationType
	current.ValidUntil = datetime.ParseDatePtr(dto.ValidUntil)
	current.DeliveryDate = datetime.ParseDatePtr(dto.DeliveryDate)
	current.DeliveryDateFirm = dto.DeliveryDateFirm
	current.PurchaseOrderNumber = dto.PurchaseOrderNumber
	current.CustomerCode = dto.CustomerCode
	current.BillingAddressCode = dto.BillingAddressCode
	current.ShippingAddressCode = dto.ShippingAddressCode
	current.RepresentativeCode = dto.RepresentativeCode
	current.SalesDivisionCode = dto.SalesDivisionCode
	current.PriceTableCode = dto.PriceTableCode
	current.PaymentTermCode = dto.PaymentTermCode
	current.CurrencyCode = dto.CurrencyCode
	if current.CurrencyCode == "" {
		current.CurrencyCode = "BRL"
	}
	current.ProbabilityPct = dto.ProbabilityPct
	current.CommissionPct = dto.CommissionPct
	current.IsNFCe = dto.IsNFCe
	current.Street = dto.Street
	current.StreetNumber = dto.StreetNumber
	current.ForeignDocument = dto.ForeignDocument
	current.ReleaseStatus = releaseStatus
	current.CommercialBlocked = dto.CommercialBlocked
	current.CommercialBlockReason = dto.CommercialBlockReason
	current.CarrierCode = dto.CarrierCode
	current.FreightType = dto.FreightType
	current.VerifyFreight = dto.VerifyFreight
	current.FreightValue = dto.FreightValue
	current.RedeliveryFreightValue = dto.RedeliveryFreightValue
	current.InsuranceValue = dto.InsuranceValue
	current.DiscountValue = dto.DiscountValue
	current.SurchargeValue = dto.SurchargeValue
	current.RetainedTaxValue = dto.RetainedTaxValue
	current.DeliveryAuthorization = dto.DeliveryAuthorization
	current.Notes = dto.Notes
	current.ObsCustomer = dto.ObsCustomer
	updated, err := uc.Repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}
	return toResponse(updated), nil
}

func (uc *UseCase) Get(ctx context.Context, code int64) (*response.SalesQuotationResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	q, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	items, _ := uc.Repo.ListItems(ctx, code)
	q.Items = items
	return toResponse(q), nil
}

func (uc *UseCase) List(ctx context.Context, filter repository.SalesQuotationFilter) ([]*response.SalesQuotationResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	items, err := uc.Repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return toResponses(items), nil
}

func (uc *UseCase) Cancel(ctx context.Context, dto request.CancelSalesQuotationDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if dto.Reason == "" {
		return errorsuc.NewValidationError("reason is required")
	}
	return uc.Repo.Cancel(ctx, dto.Code, dto.Reason, dto.Complement)
}

func (uc *UseCase) Attend(ctx context.Context, dto request.AttendSalesQuotationDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if dto.Reason == "" {
		return errorsuc.NewValidationError("reason is required")
	}
	eventDate := datetime.ParseDateOrDefault(dto.EventDate, time.Now())
	return uc.Repo.Attend(ctx, dto.Code, dto.Reason, dto.Complement, eventDate)
}

func (uc *UseCase) Uncancel(ctx context.Context, dto request.UncancelSalesQuotationDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if dto.Reason == "" {
		return errorsuc.NewValidationError("reason is required")
	}
	return uc.Repo.Uncancel(ctx, dto.Code, dto.Reason, dto.Complement)
}

func (uc *UseCase) ChangeStatus(ctx context.Context, dto request.ChangeSalesQuotationStatusDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	status := entity.SalesQuotationStatus(dto.Status)
	if !validStatus(status) {
		return errorsuc.NewValidationError("invalid quotation status")
	}
	return uc.Repo.ChangeStatus(ctx, dto.Code, status)
}

func (uc *UseCase) Report(ctx context.Context, filter repository.SalesQuotationFilter) (*response.SalesQuotationReportResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	report, err := uc.Repo.Report(ctx, filter)
	if err != nil {
		return nil, err
	}
	return toReportResponse(report), nil
}

func validStatus(status entity.SalesQuotationStatus) bool {
	switch status {
	case entity.SalesQuotationStatusDraft, entity.SalesQuotationStatusWebOrder, entity.SalesQuotationStatusAnalysis,
		entity.SalesQuotationStatusBudgetAnalysis, entity.SalesQuotationStatusERPOrder, entity.SalesQuotationStatusERPBudget,
		entity.SalesQuotationStatusCancelled, entity.SalesQuotationStatusAttended, entity.SalesQuotationStatusExpired:
		return true
	default:
		return false
	}
}

func validType(v entity.SalesQuotationType) bool {
	switch v {
	case entity.SalesQuotationTypeThirdParty, entity.SalesQuotationTypeConsult, entity.SalesQuotationTypePortal,
		entity.SalesQuotationTypeImported, entity.SalesQuotationTypeNegotiation, entity.SalesQuotationTypeSale:
		return true
	default:
		return false
	}
}

func validReleaseStatus(v entity.SalesQuotationReleaseStatus) bool {
	switch v {
	case entity.SalesQuotationReleaseBlocked, entity.SalesQuotationReleaseManual, entity.SalesQuotationReleaseOK:
		return true
	default:
		return false
	}
}
