package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDescriptionRequired       = domainerr.NewValidation("informe a descrição")
	ErrCodeInvalid               = domainerr.NewValidation("o código deve ser maior que zero")
	ErrInvalidCommercialAnalysis = domainerr.NewValidation("análise comercial inválida: use livre, bloquear sempre ou analisar sempre")
	ErrInvalidFinancialAnalysis  = domainerr.NewValidation("análise financeira inválida: use livre, bloquear sempre ou analisar sempre")
)

func isValidAnalysis(a SalesDivisionAnalysis) bool {
	return a == AnalysisFree || a == AnalysisBlockAlways || a == AnalysisAlwaysAnalyze
}

func NewSalesDivision(
	code int64,
	description string,
	commercialAnalysis SalesDivisionAnalysis,
	financialAnalysis SalesDivisionAnalysis,
	isTechnicalAssistance bool,
	considerDeliveryPromise bool,
	considerMRP bool,
	allowOutsideLimits bool,
	minimumDeliveryDays int,
	financialDelayDays int,
	pisPercentage float64,
	cofinsPercentage float64,
	parentDivisionID *int64,
	createdBy uuid.UUID,
) (*SalesDivision, error) {
	if code <= 0 {
		return nil, ErrCodeInvalid
	}
	if description == "" {
		return nil, ErrDescriptionRequired
	}
	// Empty analysis fields default to FREE, matching the column default, so the
	// caller may omit them.
	if commercialAnalysis == "" {
		commercialAnalysis = AnalysisFree
	}
	if financialAnalysis == "" {
		financialAnalysis = AnalysisFree
	}
	if !isValidAnalysis(commercialAnalysis) {
		return nil, ErrInvalidCommercialAnalysis
	}
	if !isValidAnalysis(financialAnalysis) {
		return nil, ErrInvalidFinancialAnalysis
	}

	now := time.Now()
	return &SalesDivision{
		Code:                    code,
		Description:             description,
		CommercialAnalysis:      commercialAnalysis,
		FinancialAnalysis:       financialAnalysis,
		IsTechnicalAssistance:   isTechnicalAssistance,
		ConsiderDeliveryPromise: considerDeliveryPromise,
		ConsiderMRP:             considerMRP,
		AllowOutsideLimits:      allowOutsideLimits,
		MinimumDeliveryDays:     minimumDeliveryDays,
		FinancialDelayDays:      financialDelayDays,
		PISPercentage:           pisPercentage,
		CofinsPercentage:        cofinsPercentage,
		ParentDivisionID:        parentDivisionID,
		IsActive:                true,
		CreatedAt:               now,
		UpdatedAt:               now,
		CreatedBy:               createdBy,
	}, nil
}
