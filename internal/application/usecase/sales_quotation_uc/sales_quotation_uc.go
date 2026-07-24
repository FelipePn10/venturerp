package sales_quotation_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	divisionrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type UseCase struct {
	Repo      repository.SalesQuotationRepository
	Auth      ports.AuthService
	Customers customerrepo.CustomerRepository
	Divisions divisionrepo.SalesDivisionRepository
	Items     itemrepo.ItemRepository
}

func (uc *UseCase) Create(ctx context.Context, dto request.CreateSalesQuotationDTO) (*response.SalesQuotationResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	tenantID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.EnterpriseCode != 0 && dto.EnterpriseCode != tenantID {
		return nil, errorsuc.NewValidationError("enterprise_code does not match authenticated tenant")
	}
	dto.EnterpriseCode = tenantID
	number := dto.QuotationNumber
	if number <= 0 {
		number, err = uc.Repo.NextQuotationNumber(ctx, tenantID)
		if err != nil {
			return nil, err
		}
	}
	status := entity.SalesQuotationStatusVentureBudget
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
		DeliveryWithReceipt:    dto.DeliveryWithReceipt,
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
	if err := uc.applyCustomerAndPaymentDefaults(ctx, q); err != nil {
		return nil, err
	}
	parameters, err := uc.Repo.GetParameters(ctx)
	if err != nil {
		return nil, err
	}
	if err := applyQuotationRules(q, parameters); err != nil {
		return nil, err
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
	if status != current.Status {
		return nil, errorsuc.NewValidationError("use the status endpoint to change quotation status")
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
	if releaseStatus != current.ReleaseStatus {
		return nil, errorsuc.NewValidationError("use the release endpoint to change quotation release")
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
	current.DeliveryWithReceipt = dto.DeliveryWithReceipt
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
	if err := uc.applyCustomerAndPaymentDefaults(ctx, current); err != nil {
		return nil, err
	}
	parameters, err := uc.Repo.GetParameters(ctx)
	if err != nil {
		return nil, err
	}
	if err := applyQuotationRules(current, parameters); err != nil {
		return nil, err
	}
	updated, err := uc.Repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.RecalculateTotals(ctx, updated.Code); err != nil {
		return nil, err
	}
	if err := uc.applyCommercialPolicies(ctx, updated.Code); err != nil {
		return nil, err
	}
	updated, err = uc.Repo.GetByCode(ctx, updated.Code)
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
	items, err := uc.Repo.ListItems(ctx, code)
	if err != nil {
		return nil, err
	}
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
	reason, err := uc.Repo.GetCancellationReason(ctx, dto.ReasonCode)
	if err != nil {
		return err
	}
	if reason.RequireComplement && (dto.Complement == nil || strings.TrimSpace(*dto.Complement) == "") {
		return errorsuc.NewValidationError("complement is required for the selected cancellation reason")
	}
	return uc.Repo.Cancel(ctx, dto.Code, reason.Code, reason.Description, dto.Complement)
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
	reason, err := uc.Repo.GetCancellationReason(ctx, dto.ReasonCode)
	if err != nil {
		return err
	}
	if !reason.AllowUncancel {
		return errorsuc.NewValidationError("selected reason does not allow uncancellation")
	}
	if reason.RequireComplement && (dto.Complement == nil || strings.TrimSpace(*dto.Complement) == "") {
		return errorsuc.NewValidationError("complement is required for the selected cancellation reason")
	}
	return uc.Repo.Uncancel(ctx, dto.Code, reason.Code, reason.Description, dto.Complement)
}

func (uc *UseCase) ChangeStatus(ctx context.Context, dto request.ChangeSalesQuotationStatusDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	status := entity.SalesQuotationStatus(dto.Status)
	if !validStatus(status) {
		return errorsuc.NewValidationError("invalid quotation status")
	}
	current, err := uc.Repo.GetByCode(ctx, dto.Code)
	if err != nil {
		return err
	}
	if !validManualTransition(current.Status, status) {
		return errorsuc.NewValidationError("invalid quotation status transition")
	}
	return uc.Repo.ChangeStatus(ctx, dto.Code, status)
}

func (uc *UseCase) ChangeRelease(ctx context.Context, dto request.ChangeSalesQuotationReleaseDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	status := entity.SalesQuotationReleaseStatus(dto.ReleaseStatus)
	if !validReleaseStatus(status) {
		return errorsuc.NewValidationError("invalid release_status")
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return errorsuc.NewValidationError("reason is required")
	}
	return uc.Repo.ChangeRelease(ctx, dto.Code, status, strings.TrimSpace(dto.Reason))
}
func (uc *UseCase) ListEvents(ctx context.Context, code int64) ([]*entity.Event, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListEvents(ctx, code)
}

func validManualTransition(from, to entity.SalesQuotationStatus) bool {
	if from == to {
		return true
	}
	if to == entity.SalesQuotationStatusCancelled || to == entity.SalesQuotationStatusAttended || to == entity.SalesQuotationStatusExpired {
		return false
	}
	switch from {
	case entity.SalesQuotationStatusDraft:
		return to == entity.SalesQuotationStatusAnalysis || to == entity.SalesQuotationStatusBudgetAnalysis || to == entity.SalesQuotationStatusVentureOrder || to == entity.SalesQuotationStatusVentureBudget
	case entity.SalesQuotationStatusAnalysis, entity.SalesQuotationStatusBudgetAnalysis:
		return to == entity.SalesQuotationStatusDraft || to == entity.SalesQuotationStatusVentureOrder || to == entity.SalesQuotationStatusVentureBudget
	case entity.SalesQuotationStatusERPOrder, entity.SalesQuotationStatusERPBudget, entity.SalesQuotationStatusVentureOrder, entity.SalesQuotationStatusVentureBudget:
		return to == entity.SalesQuotationStatusAnalysis || to == entity.SalesQuotationStatusBudgetAnalysis
	default:
		return false
	}
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
		entity.SalesQuotationStatusVentureOrder, entity.SalesQuotationStatusVentureBudget,
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
