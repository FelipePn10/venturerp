package recurring_sales_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
)

func toParametersResponse(v *entity.Parameters) *response.RecurringSalesParametersResponse {
	return &response.RecurringSalesParametersResponse{
		EnterpriseCode: v.EnterpriseCode, CurrentMonthBillingLimitDay: v.CurrentMonthBillingLimitDay,
		GroupOrderItemTotal: v.GroupOrderItemTotal, IndefiniteDeliveryDay: v.IndefiniteDeliveryDay,
		FixedTermDeliveryDay: v.FixedTermDeliveryDay, ConsiderDiscountsAdditions: v.ConsiderDiscountsAdditions,
		GenericRepresentativeCode: v.GenericRepresentativeCode, GenericSalesPlanCode: v.GenericSalesPlanCode,
		UpdatedAt: v.UpdatedAt,
	}
}

func toAdjustmentDateResponse(v *entity.AdjustmentDate) *response.RecurringSalesAdjustmentDateResponse {
	return &response.RecurringSalesAdjustmentDateResponse{
		Code: v.Code, EnterpriseCode: v.EnterpriseCode, CustomerCode: v.CustomerCode,
		EstablishmentCode: v.EstablishmentCode, AdjustmentDate: v.AdjustmentDate,
		Notes: v.Notes, CreatedAt: v.CreatedAt,
	}
}

func toRecurringSaleResponse(v *entity.RecurringSale) *response.RecurringSaleResponse {
	out := &response.RecurringSaleResponse{
		Code: v.Code, EnterpriseCode: v.EnterpriseCode, CustomerCode: v.CustomerCode,
		EstablishmentCode: v.EstablishmentCode, ItemCode: v.ItemCode, ItemMask: v.ItemMask,
		SalesPlanCode: v.SalesPlanCode, MovementType: string(v.MovementType), TermType: string(v.TermType),
		SaleDate: v.SaleDate, NextAdjustmentDate: v.NextAdjustmentDate, MonthsQuantity: v.MonthsQuantity,
		PaymentsQuantity: v.PaymentsQuantity, GraceMonths: v.GraceMonths, PaymentValue: v.PaymentValue,
		Quantity: v.Quantity, UnitValue: v.UnitValue, MonthlyValue: monthlyValue(v),
		Reason: v.Reason, GeneratedOrderCode: v.GeneratedOrderCode, GeneratedOrderAt: v.GeneratedOrderAt,
		SourceRecurringSaleCode: v.SourceRecurringSaleCode, OriginalAdjustmentCode: v.OriginalAdjustmentCode,
		AdjustmentPercent: v.AdjustmentPercent, IsActive: v.IsActive, CreatedAt: v.CreatedAt,
	}
	for _, rep := range v.Representatives {
		out.Representatives = append(out.Representatives, response.RecurringSaleRepresentativeResponse{
			Code: rep.Code, RepresentativeCode: rep.RepresentativeCode, IsPrimary: rep.IsPrimary,
			CommissionPercent: rep.CommissionPercent, CommissionBase: string(rep.CommissionBase),
			IsLifetime: rep.IsLifetime, CommissionInstallments: rep.CommissionInstallments,
		})
	}
	return out
}
