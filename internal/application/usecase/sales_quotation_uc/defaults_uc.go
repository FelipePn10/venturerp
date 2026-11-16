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
		return errorsuc.NewValidationError("o cliente está inativo ou bloqueado")
	}
	if q.CarrierCode == nil && customer.CarrierID != nil {
		carrier, err := uc.Customers.GetCarrierByID(ctx, *customer.CarrierID)
		if err != nil {
			return err
		}
		q.CarrierCode = &carrier.Code
	}
	// Condição de pagamento e tabela de preço não vêm mais do cliente: a
	// condição é negociada no orçamento e a tabela é a do item.
	if q.PaymentTermCode != nil {
		if _, err := uc.Customers.GetPaymentConditionByCode(ctx, *q.PaymentTermCode); err != nil {
			return err
		}
	}
	return nil
}
