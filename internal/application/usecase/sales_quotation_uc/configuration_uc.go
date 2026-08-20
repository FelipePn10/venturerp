package sales_quotation_uc

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	"github.com/shopspring/decimal"
)

func (uc *UseCase) GetParameters(ctx context.Context) (*entity.Parameters, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetParameters(ctx)
}

func (uc *UseCase) SaveParameters(ctx context.Context, dto request.SaveSalesQuotationParametersDTO) (*entity.Parameters, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	tenantID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	p := &entity.Parameters{EnterpriseCode: tenantID, PurchaseOrderPrompt: dto.PurchaseOrderPrompt, DeliveryAuthorizationPrompt: dto.DeliveryAuthorizationPrompt, FinalConsumerCustomerCode: dto.FinalConsumerCustomerCode, AllowServiceItemsNFCe: dto.AllowServiceItemsNFCe, DefaultNFCe: dto.DefaultNFCe, MinimumCIFFreight: dto.MinimumCIFFreight, AddRedeliveryToFreight: dto.AddRedeliveryToFreight}
	if err := p.Validate(); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	return uc.Repo.SaveParameters(ctx, p)
}

func (uc *UseCase) SaveCommissionPattern(ctx context.Context, dto request.SaveCommissionPatternDTO) (*entity.CommissionPattern, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p := &entity.CommissionPattern{Code: dto.Code, Description: dto.Description, CommissionPct: dto.CommissionPct, InvoicePct: dto.InvoicePct, PaymentPct: dto.PaymentPct}
	validationCode := p.Code
	if p.Code == 0 {
		p.Code = 1
	}
	if err := p.Validate(); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	p.Code = validationCode
	return uc.Repo.SaveCommissionPattern(ctx, p)
}
func (uc *UseCase) ListCommissionPatterns(ctx context.Context) ([]*entity.CommissionPattern, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListCommissionPatterns(ctx)
}
func (uc *UseCase) SaveCancellationReason(ctx context.Context, dto request.SaveCancellationReasonDTO) (*entity.CancellationReason, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	v := &entity.CancellationReason{Code: dto.Code, Description: dto.Description, AllowUncancel: dto.AllowUncancel, RequireComplement: dto.RequireComplement}
	validationCode := v.Code
	if v.Code == 0 {
		v.Code = 1
	}
	if err := v.Validate(); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	v.Code = validationCode
	return uc.Repo.SaveCancellationReason(ctx, v)
}
func (uc *UseCase) ListCancellationReasons(ctx context.Context) ([]*entity.CancellationReason, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListCancellationReasons(ctx)
}

func (uc *UseCase) GenerateDAV(ctx context.Context, code int64) (*response.SalesQuotationResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	q, err := uc.Repo.GenerateDAV(ctx, code)
	if err != nil {
		return nil, err
	}
	return toResponse(q), nil
}

func (uc *UseCase) CreateAttachment(ctx context.Context, dto request.CreateSalesQuotationAttachmentDTO) (*entity.Attachment, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	storageKey := fmt.Sprintf("db://sales-quotation/%d/%s", dto.SalesQuotationCode, path.Base(dto.FileName))
	return uc.Repo.CreateAttachment(ctx, &entity.Attachment{SalesQuotationCode: dto.SalesQuotationCode, FileName: path.Base(dto.FileName), ContentType: dto.ContentType, FileSize: dto.FileSize, StorageKey: storageKey, UploadedBy: userID, Content: dto.Content})
}
func (uc *UseCase) GetAttachment(ctx context.Context, code, id int64) (*entity.Attachment, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetAttachment(ctx, code, id)
}
func (uc *UseCase) ListAttachments(ctx context.Context, code int64) ([]*entity.Attachment, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListAttachments(ctx, code)
}
func (uc *UseCase) DeleteAttachment(ctx context.Context, code, id int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.DeleteAttachment(ctx, code, id)
}

func applyQuotationRules(q *entity.SalesQuotation, p *entity.Parameters) error {
	q.IsNFCe = q.IsNFCe || p.DefaultNFCe
	if q.DeliveryWithReceipt {
		q.IsNFCe = true
	}
	if q.CustomerCode != nil && p.FinalConsumerCustomerCode != nil && *q.CustomerCode == *p.FinalConsumerCustomerCode {
		address := strings.TrimSpace(value(q.Street))
		if number := strings.TrimSpace(value(q.StreetNumber)); number != "" {
			if address != "" {
				address += ", "
			}
			address += number
		}
		if address != "" {
			q.ConsumerAddress = &address
		}
	} else if q.ForeignDocument != nil {
		return errorsuc.NewValidationError("foreign_document is only allowed for the final consumer customer")
	}
	if !q.IsNFCe && q.ForeignDocument != nil {
		return errorsuc.NewValidationError("foreign_document requires NFC-e")
	}
	typeName := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value(q.FreightType)), "-", " "), "Ó", "O"))
	switch typeName {
	case "FOB CONTRAT.", "FOB PROPRIO", "CORTESIA", "RETIRA", "SEM FRETE", "TERCEIROS":
		q.FreightValue = decimal.Zero
		q.InsuranceValue = decimal.Zero
	case "CIF CONTRAT.", "CIF PROPRIO", "DAF":
		if q.VerifyFreight {
			minimum := p.MinimumCIFFreight
			if q.FreightValue.LessThan(minimum) {
				q.FreightValue = minimum
			}
		}
		if p.AddRedeliveryToFreight && q.RedeliveryFreightValue.IsPositive() {
			q.FreightValue = q.FreightValue.Add(q.RedeliveryFreightValue)
			q.RedeliveryFreightValue = decimal.Zero
		}
	case "CONVENIO":
		if p.AddRedeliveryToFreight && q.RedeliveryFreightValue.IsPositive() {
			q.FreightValue = q.FreightValue.Add(q.RedeliveryFreightValue)
			q.RedeliveryFreightValue = decimal.Zero
		}
	default:
		if typeName != "" {
			return errorsuc.NewValidationError("invalid freight_type")
		}
	}
	return nil
}

func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
