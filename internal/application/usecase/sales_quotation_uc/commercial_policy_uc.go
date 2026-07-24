package sales_quotation_uc

import (
	"context"
	"strconv"

	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/shopspring/decimal"
)

func (uc *UseCase) applyCommercialPolicies(ctx context.Context, quotationCode int64) error {
	if uc.Customers == nil {
		return nil
	}
	q, err := uc.Repo.GetByCode(ctx, quotationCode)
	if err != nil {
		return err
	}
	policies, err := uc.Customers.ListCommercialPolicies(ctx, true, nil)
	if err != nil {
		return err
	}
	for _, p := range policies {
		lines, err := uc.Customers.ListCommercialPolicyLines(ctx, p.Code)
		if err != nil {
			return err
		}
		p.Lines = lines
	}
	quantity := decimal.Zero
	items, err := uc.Repo.ListItems(ctx, quotationCode)
	if err != nil {
		return err
	}
	for _, item := range items {
		quantity = quantity.Add(item.Balance)
	}
	policyCtx := customerentity.CommercialPolicyContext{GrossValue: q.TotalGross.InexactFloat64(), Quantity: quantity.InexactFloat64(), CustomerCode: q.CustomerCode, CarrierID: q.CarrierCode}
	if q.CustomerCode != nil {
		customer, err := uc.Customers.GetCustomerByCode(ctx, *q.CustomerCode)
		if err != nil {
			return err
		}
		policyCtx.CustomerTypeID = customer.CustomerTypeID
		policyCtx.MarketSegmentID = customer.MarketSegmentID
		policyCtx.RegionID = customer.RegionID
	}
	if q.PaymentTermCode != nil {
		condition, err := uc.Customers.GetPaymentConditionByCode(ctx, *q.PaymentTermCode)
		if err != nil {
			return err
		}
		policyCtx.PaymentConditionID = &condition.ID
	}
	if q.PriceTableCode != nil {
		table, err := uc.Customers.GetSalesTableByCode(ctx, *q.PriceTableCode)
		if err != nil {
			return err
		}
		policyCtx.SalesTableID = &table.ID
	}
	if q.CarrierCode != nil {
		carrier, err := uc.Customers.GetCarrierByCode(ctx, *q.CarrierCode)
		if err != nil {
			return err
		}
		policyCtx.CarrierID = &carrier.ID
	}
	headerPolicies := make([]*customerentity.CommercialPolicy, 0, len(policies))
	itemPolicies := make([]*customerentity.CommercialPolicy, 0, len(policies))
	for _, policy := range policies {
		if policy.AppliesToItems || policy.ItemCode != nil || policy.ItemMask != nil || policy.ProductLineID != nil || policy.ItemClassification != nil {
			itemPolicies = append(itemPolicies, policy)
			continue
		}
		headerPolicies = append(headerPolicies, policy)
	}
	evaluation, err := customerentity.EvaluateCommercialPolicies(headerPolicies, policyCtx)
	if err != nil {
		return err
	}
	for _, item := range items {
		itemCode := strconv.FormatInt(item.ItemCode, 10)
		itemMask := item.Mask
		itemCtx := policyCtx
		itemCtx.GrossValue = item.TotalGross.InexactFloat64()
		itemCtx.Quantity = item.Balance.InexactFloat64()
		itemCtx.ItemCode = &itemCode
		itemCtx.ItemMask = &itemMask
		itemEvaluation, err := customerentity.EvaluateCommercialPolicies(itemPolicies, itemCtx)
		if err != nil {
			return err
		}
		evaluation.DiscountValue += itemEvaluation.DiscountValue
		evaluation.SurchargeValue += itemEvaluation.SurchargeValue
		evaluation.FreightValue += itemEvaluation.FreightValue
		evaluation.CommissionValue += itemEvaluation.CommissionValue
		evaluation.RequiresApproval = evaluation.RequiresApproval || itemEvaluation.RequiresApproval
	}
	changed := false
	if q.DiscountValue.IsZero() && evaluation.DiscountValue > 0 {
		q.DiscountValue = decimal.NewFromFloat(evaluation.DiscountValue)
		changed = true
	}
	if q.SurchargeValue.IsZero() && evaluation.SurchargeValue > 0 {
		q.SurchargeValue = decimal.NewFromFloat(evaluation.SurchargeValue)
		changed = true
	}
	if q.FreightValue.IsZero() && evaluation.FreightValue > 0 {
		q.FreightValue = decimal.NewFromFloat(evaluation.FreightValue)
		changed = true
	}
	if q.CommissionPct.IsZero() && evaluation.CommissionValue > 0 && q.TotalGross.IsPositive() {
		q.CommissionPct = decimal.NewFromFloat(evaluation.CommissionValue).Div(q.TotalGross).Mul(decimal.NewFromInt(100))
		changed = true
	}
	if evaluation.RequiresApproval {
		q.CommercialBlocked = true
		reason := "Política comercial exige aprovação"
		q.CommercialBlockReason = &reason
		q.ReleaseStatus = "BLOCKED"
		changed = true
	}
	if !changed {
		return nil
	}
	if _, err = uc.Repo.Update(ctx, q); err != nil {
		return err
	}
	return uc.Repo.RecalculateTotals(ctx, q.Code)
}
