package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDescriptionRequired       = errors.New("description is required")
	ErrCodeInvalid               = errors.New("code must be greater than zero")
	ErrInvalidCommercialAnalysis = errors.New("invalid commercial analysis value")
	ErrInvalidFinancialAnalysis  = errors.New("invalid financial analysis value")
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
