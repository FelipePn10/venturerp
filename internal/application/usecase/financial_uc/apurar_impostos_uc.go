package financial_uc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/reforma"
	"github.com/shopspring/decimal"
)

type ApurarImpostosUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ApurarImpostosUseCase) Execute(ctx context.Context, dto request.ApurarImpostosDTO) ([]*response.TaxAssessmentResponse, error) {
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

	type aApurar struct {
		imposto         string
		debito, credito decimal.Decimal
	}
	var lista []aApurar
	for _, imposto := range impostosDaCompetencia(dto.Competencia) {
		lista = append(lista, aApurar{imposto, debits[imposto], credits[imposto]})
	}
	// ICMS-ST, DIFAL e FCP devidos a cada estado de destino: um registro e um
	// título por UF (é o que vira a guia — GNRE — de cada estado).
	porUF, err := uc.Repo.GetDebitosPorUF(ctx, dto.Competencia)
	if err != nil {
		return nil, err
	}
	for _, d := range porUF {
		lista = append(lista, aApurar{d.Imposto + "-" + d.UF, d.Valor, decimal.Zero})
	}

	var results []*entity.TaxAssessment
	for _, ap := range lista {
		imposto := ap.imposto
		existing, errApuracao := uc.Repo.GetTaxAssessment(ctx, imposto, dto.Competencia)
		if errApuracao != nil && !errorsuc.EhNaoEncontrado(errApuracao) {
			return nil, fmt.Errorf("lendo a apuração existente: %w", errApuracao)
		}
		if existing != nil {
			results = append(results, existing)
			continue
		}

		debito := ap.debito
		credito := ap.credito
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
			// Sem o título a apuração não pode ficar gravada como apurada: o
			// imposto sumiria do contas a pagar. Antes o erro era ignorado.
			created, err := uc.Repo.CreateContaPagar(ctx, cp)
			if err != nil {
				return nil, fmt.Errorf("criando o título do %s da competência %s: %w", imposto, dto.Competencia, err)
			}
			cpID = &created.ID
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

	return toTaxAssessmentResponses(results), nil
}

// impostosDaCompetencia lista o que se apura no mês. IBS e CBS entram a partir
// de 2027: em 2026 o destaque é informativo e não há saldo a recolher.
func impostosDaCompetencia(competencia string) []string {
	impostos := []string{"ICMS", "IPI", "PIS", "COFINS"}
	if reforma.RecolhivelNaCompetencia(competencia) {
		impostos = append(impostos, "IBS", "CBS")
	}
	return impostos
}

func calculateVencimento(competencia, imposto string, config *fiscalEntity.FiscalConfig) time.Time {
	parts := strings.Split(competencia, "/")
	if len(parts) != 2 {
		return time.Now().AddDate(0, 1, 15)
	}
	month, _ := strconv.Atoi(parts[0])
	year, _ := strconv.Atoi(parts[1])

	var day int
	switch {
	case imposto == "ICMS" || strings.HasPrefix(imposto, "ST-") || strings.HasPrefix(imposto, "DIFAL-") || strings.HasPrefix(imposto, "FCP-"):
		day = config.VencimentoIcmsDia
	case imposto == "IPI":
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
