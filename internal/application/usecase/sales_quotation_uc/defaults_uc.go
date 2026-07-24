package sales_quotation_uc

import (
	"context"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

func (uc *UseCase) applyCustomerAndPaymentDefaults(ctx context.Context, q *quoteentity.SalesQuotation) error {
	if q.CustomerCode == nil || uc.Customers == nil {
		return nil
	}
	customer, err := uc.Customers.GetCustomerByCode(ctx, *q.CustomerCode)
	if err != nil {
		return err
	}
	if !customer.IsActive || customer.Blocked {
		return errorsuc.NewValidationError("customer is inactive or blocked")
	}
	if q.CarrierCode == nil && customer.CarrierID != nil {
		carrier, err := uc.Customers.GetCarrierByID(ctx, *customer.CarrierID)
		if err != nil {
			return err
		}
		q.CarrierCode = &carrier.Code
	}
	if q.PriceTableCode == nil && customer.SalesTableID != nil {
		table, err := uc.Customers.GetSalesTableByID(ctx, *customer.SalesTableID)
		if err != nil {
			return err
		}
		q.PriceTableCode = &table.Code
	}
	var defaultPayment *int64
	if customer.PaymentConditionID != nil {
		condition, err := uc.Customers.GetPaymentConditionByID(ctx, *customer.PaymentConditionID)
		if err != nil {
			return err
		}
		defaultPayment = &condition.Code
	}
	if q.PaymentTermCode == nil {
		q.PaymentTermCode = defaultPayment
		return nil
	}
	if _, err := uc.Customers.GetPaymentConditionByCode(ctx, *q.PaymentTermCode); err != nil {
		return err
	}
	if defaultPayment == nil {
		return nil
	}
	if defaultPayment != nil && *q.PaymentTermCode == *defaultPayment {
		return nil
	}
	if q.SalesDivisionCode == nil || uc.Divisions == nil {
		return errorsuc.NewValidationError("a sales division with free payment terms is required to override the customer payment condition")
	}
	division, err := uc.Divisions.GetByCode(ctx, *q.SalesDivisionCode)
	if err != nil {
		return err
	}
	if !division.AllowFreePaymentTerms {
		return errorsuc.NewValidationError("sales division does not allow free payment terms")
	}
	return nil
}
