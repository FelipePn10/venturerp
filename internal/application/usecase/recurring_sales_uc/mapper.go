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
	missing := recurringPreconditions(v)
	out := &response.RecurringSaleResponse{
		Code: v.Code, EnterpriseCode: v.EnterpriseCode, CustomerCode: v.CustomerCode,
		EstablishmentCode: v.EstablishmentCode, ItemCode: v.ItemCode, ItemMask: v.ItemMask,
		SalesPlanCode: v.SalesPlanCode, MovementType: string(v.MovementType), TermType: string(v.TermType),
		SaleDate: v.SaleDate, NextAdjustmentDate: v.NextAdjustmentDate, MonthsQuantity: v.MonthsQuantity,
		PaymentsQuantity: v.PaymentsQuantity, GraceMonths: v.GraceMonths, PaymentValue: v.PaymentValue,
		Quantity: v.Quantity, UnitValue: v.UnitValue, MonthlyValue: monthlyValue(v),
		Reason: v.Reason, GeneratedOrderCode: v.GeneratedOrderCode, GeneratedOrderAt: v.GeneratedOrderAt,
		SourceRecurringSaleCode: v.SourceRecurringSaleCode, OriginalAdjustmentCode: v.OriginalAdjustmentCode,
		AdjustmentPercent: v.AdjustmentPercent, IsActive: v.IsActive, LifecycleStatus: string(v.LifecycleStatus),
		EffectiveFrom: v.EffectiveFrom, EffectiveUntil: v.EffectiveUntil,
		CancellationEffectiveDate: v.CancellationEffectiveDate, FutureOrdersPolicy: v.FutureOrdersPolicy, CreatedAt: v.CreatedAt,
		Frequency: v.Frequency, PriceTableCode: v.PriceTableCode, CurrencyCode: v.CurrencyCode,
		AdjustmentIndex: v.AdjustmentIndex, AdjustmentPeriodMonths: v.AdjustmentPeriodMonths,
		AdjustmentFloorPct: v.AdjustmentFloorPct, AdjustmentCapPct: v.AdjustmentCapPct,
		BillingPolicy: v.BillingPolicy, DeliveryPolicy: v.DeliveryPolicy, TaxPolicy: v.TaxPolicy,
		CostCenterCode: v.CostCenterCode, RenewalPolicy: v.RenewalPolicy,
		MissingPreconditions: missing, CanGenerateOrder: len(missing) == 0,
		CanCancel:      v.IsActive,
		CanAdjust:      v.IsActive && v.TermType == entity.TermIndefinite && v.GeneratedOrderCode != nil,
		AllowedActions: make([]string, 0),
	}
	if out.CanGenerateOrder {
		out.AllowedActions = append(out.AllowedActions, "GERAR_PEDIDO")
	}
	if out.CanAdjust {
		out.AllowedActions = append(out.AllowedActions, "REAJUSTAR")
	}
	if out.CanCancel {
		out.AllowedActions = append(out.AllowedActions, "CANCELAR")
	}
	if v.IsActive {
		out.AllowedActions = append(out.AllowedActions, "ATUALIZAR", "SUSPENDER")
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

func recurringPreconditions(v *entity.RecurringSale) []string {
	missing := make([]string, 0)
	if !v.IsActive {
		missing = append(missing, "a recorrência está inativa")
	}
	if v.GeneratedOrderCode != nil {
		missing = append(missing, "a recorrência já possui pedido gerado")
	}
	if primaryRepresentative(v) == nil {
		missing = append(missing, "representante principal")
	}
	if v.SalesPlanCode == nil {
		missing = append(missing, "plano de vendas")
	}
	if v.TermType == entity.TermIndefinite && v.NextAdjustmentDate == nil {
		missing = append(missing, "próxima data de reajuste")
	}
	return missing
}
