package sales_division_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity"
)

// toSalesDivisionResponse converts a domain SalesDivision into its API DTO.
func toSalesDivisionResponse(sd *entity.SalesDivision) *response.SalesDivisionResponse {
	if sd == nil {
		return nil
	}
	return &response.SalesDivisionResponse{
		ID:                      sd.ID,
		Code:                    sd.Code,
		Description:             sd.Description,
		CommercialAnalysis:      string(sd.CommercialAnalysis),
		FinancialAnalysis:       string(sd.FinancialAnalysis),
		IsTechnicalAssistance:   sd.IsTechnicalAssistance,
		ConsiderDeliveryPromise: sd.ConsiderDeliveryPromise,
		ConsiderMRP:             sd.ConsiderMRP,
		AllowOutsideLimits:      sd.AllowOutsideLimits,
		MinimumDeliveryDays:     sd.MinimumDeliveryDays,
		FinancialDelayDays:      sd.FinancialDelayDays,
		PISPercentage:           sd.PISPercentage,
		CofinsPercentage:        sd.CofinsPercentage,
		ParentDivisionID:        sd.ParentDivisionID,
		IsActive:                sd.IsActive,
		CreatedAt:               sd.CreatedAt,
		UpdatedAt:               sd.UpdatedAt,
		CreatedBy:               sd.CreatedBy,
	}
}

// toSalesDivisionResponses maps a slice of domain divisions to response DTOs.
func toSalesDivisionResponses(divisions []*entity.SalesDivision) []*response.SalesDivisionResponse {
	out := make([]*response.SalesDivisionResponse, 0, len(divisions))
	for _, sd := range divisions {
		out = append(out, toSalesDivisionResponse(sd))
	}
	return out
}
