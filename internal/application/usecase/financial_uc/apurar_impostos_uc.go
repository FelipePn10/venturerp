package financial_uc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/shopspring/decimal"
)

type ApurarImpostosUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ApurarImpostosUseCase) Execute(ctx context.Context, dto request.ApurarImpostosDTO) ([]*entity.TaxAssessment, error) {
	if !uc.Auth.CanApurarImpostos(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	config, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}

	debits, err := uc.Repo.GetFiscalDebits(ctx, dto.Competencia)
	if err != nil {
		return nil, err
	}

	credits, err := uc.Repo.GetFiscalCredits(ctx, dto.Competencia)
	if err != nil {
		return nil, err
	}

	impostos := []string{"ICMS", "IPI", "PIS", "COFINS"}
	var results []*entity.TaxAssessment

	for _, imposto := range impostos {
		existing, _ := uc.Repo.GetTaxAssessment(ctx, imposto, dto.Competencia)
		if existing != nil {
			results = append(results, existing)
			continue
		}

		debito := decimal.NewFromFloat(debits[imposto])
		credito := decimal.NewFromFloat(credits[imposto])
		saldo := debito.Sub(credito)

		var saldoDevedor, saldoCredor decimal.Decimal
		var cpID *int64
		var dataVenc *time.Time

		if saldo.GreaterThan(decimal.Zero) {
			saldoDevedor = saldo
			venc := calculateVencimento(dto.Competencia, imposto, config)
			dataVenc = &venc

			cp := &entity.ContaPagar{
				NumeroDocumento: fmt.Sprintf("APUR-%s-%s", imposto, strings.ReplaceAll(dto.Competencia, "/", "")),
				TipoDocumento:   "IMPOSTO",
				DataLancamento:  time.Now(),
				DataEmissao:     venc.AddDate(0, -1, 0),
				DataVencimento:  venc,
				ValorBruto:      saldo,
				Desconto:        decimal.Zero,
				Juros:           decimal.Zero,
				Multa:           decimal.Zero,
				ValorPago:       decimal.Zero,
				ParcelaNumero:   1,
				ParcelaTotal:    1,
				StatusAprovacao: entity.AprovacaoPendente,
				Status:          entity.ContaPagarStatusPendente,
				IsActive:        true,
				CriadoPor:       userID,
			}
			created, err := uc.Repo.CreateContaPagar(ctx, cp)
			if err == nil && created != nil {
				cpID = &created.ID
			}
		} else {
			saldoCredor = saldo.Neg()
		}

		ta := &entity.TaxAssessment{
			Imposto:        imposto,
			Competencia:    dto.Competencia,
			Debitos:        debito,
			Creditos:       credito,
			SaldoDevedor:   saldoDevedor,
			SaldoCredor:    saldoCredor,
			Status:         entity.TaxStatusApurado,
			CpID:           cpID,
			DataVencimento: dataVenc,
		}

		created, err := uc.Repo.CreateTaxAssessment(ctx, ta)
		if err != nil {
			return nil, err
		}
		results = append(results, created)
	}

	return results, nil
}

func calculateVencimento(competencia, imposto string, config *fiscalEntity.FiscalConfig) time.Time {
	parts := strings.Split(competencia, "/")
	if len(parts) != 2 {
		return time.Now().AddDate(0, 1, 15)
	}
	month, _ := strconv.Atoi(parts[0])
	year, _ := strconv.Atoi(parts[1])

	var day int
	switch imposto {
	case "ICMS":
		day = config.VencimentoIcmsDia
	case "IPI":
		day = config.VencimentoIPIDia
	default:
		day = config.VencimentoPisCofinsDia
	}

	if day <= 0 {
		day = 15
	}

	nextMonth := time.Month(month) + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}

	return time.Date(nextYear, nextMonth, day, 0, 0, 0, 0, time.UTC)
}
