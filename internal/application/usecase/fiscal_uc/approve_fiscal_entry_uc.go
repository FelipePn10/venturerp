package fiscal_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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

func (uc *ApproveFiscalEntryUseCase) Execute(ctx context.Context, dto request.ApproveFiscalEntryDTO) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanApproveFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
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

	// Accumulate tax credits per imposto
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
		if valor <= 0 {
			continue
		}
		ta := &financialEntity.TaxAssessment{
			Imposto:     imposto,
			Competencia: competencia,
			Creditos:    decimal.NewFromFloat(valor),
			Debitos:     decimal.Zero,
			Status:      financialEntity.TaxStatusApurar,
		}
		if err := uc.FinancialRepo.UpsertTaxAssessmentCredito(ctx, ta); err != nil {
			return nil, fmt.Errorf("registrando crédito %s: %w", imposto, err)
		}
	}

	// Auto-gerar Conta a Pagar para o fornecedor
	if entry.ValorTotal > 0 {
		nfNum := fmt.Sprintf("NF-%d/%s", entry.NumeroNF, entry.Serie)
		cp := &financialEntity.ContaPagar{
			NumeroDocumento: nfNum,
			TipoDocumento:   "NFE",
			FiscalEntryID:   &entry.ID,
			DataLancamento:  time.Now(),
			DataEmissao:     entry.DataEmissao,
			DataVencimento:  entry.DataEmissao.AddDate(0, 0, 30), // padrão 30 dias; usuário pode ajustar
			ValorBruto:      decimal.NewFromFloat(entry.ValorTotal),
			Desconto:        decimal.Zero,
			Juros:           decimal.Zero,
			Multa:           decimal.Zero,
			ValorPago:       decimal.Zero,
			ParcelaNumero:   1,
			ParcelaTotal:    1,
			StatusAprovacao: financialEntity.AprovacaoPendente,
			Status:          financialEntity.ContaPagarStatusPendente,
			IsActive:        true,
			CriadoPor:       userID,
		}
		if _, err := uc.FinancialRepo.CreateContaPagar(ctx, cp); err != nil {
			return nil, fmt.Errorf("gerando conta a pagar: %w", err)
		}
	}

	return toFiscalEntryResponse(entry), nil
}
