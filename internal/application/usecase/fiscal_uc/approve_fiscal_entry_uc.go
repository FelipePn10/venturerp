package fiscal_uc

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	financialEntity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type ApproveFiscalEntryUseCase struct {
	FiscalRepo    repository.FiscalRepository
	FinancialRepo financialrepo.FinancialRepository
	Auth          ports.AuthService
}

func (uc *ApproveFiscalEntryUseCase) Execute(ctx context.Context, dto request.ApproveFiscalEntryDTO) (*entity.FiscalEntry, error) {
	if !uc.Auth.CanApproveFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	entry, err := uc.FiscalRepo.UpdateEntryStatus(ctx, dto.ID, entity.EntryStatusApproved)
	if err != nil {
		return nil, err
	}

	items, err := uc.FiscalRepo.GetEntryItems(ctx, dto.ID)
	if err != nil {
		return nil, err
	}

	competencia := entry.DataEntrada.Format("01/2006")

	credits := map[string]float64{"ICMS": 0, "IPI": 0, "PIS": 0, "COFINS": 0}
	for _, item := range items {
		if item.GeraCreditoICMS {
			credits["ICMS"] += item.ValorICMS
		}
		if item.GeraCreditoIPI {
			credits["IPI"] += item.ValorIPI
		}
		if item.GeraCreditoPIS {
			credits["PIS"] += item.ValorPIS
		}
		if item.GeraCreditoCOFINS {
			credits["COFINS"] += item.ValorCOFINS
		}
	}

	for imposto, valor := range credits {
		if valor > 0 {
			ta := &financialEntity.TaxAssessment{
				Imposto:     imposto,
				Competencia: competencia,
				Creditos:    decimal.NewFromFloat(valor),
				Debitos:     decimal.Zero,
				Status:      financialEntity.TaxStatusApurar,
			}
			if _, err := uc.FinancialRepo.CreateTaxAssessment(ctx, ta); err != nil {
				return nil, err
			}
		}
	}

	return entry, nil
}
