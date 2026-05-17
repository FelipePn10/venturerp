package financial_uc

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/shopspring/decimal"
)

type BaixarContaPagarUseCase struct {
	Repo       repository.FinancialRepository
	Auth       ports.AuthService
	FiscalRepo fiscalrepo.FiscalRepository
}

func (uc *BaixarContaPagarUseCase) Execute(ctx context.Context, id int64, dto request.BaixarContaPagarDTO) error {
	if !uc.Auth.CanBaixarContaPagar(ctx) {
		return errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}

	cp, err := uc.Repo.GetContaPagar(ctx, id)
	if err != nil {
		return err
	}

	if cp.Status != entity.ContaPagarStatusPendente && cp.Status != entity.ContaPagarStatusAprovado {
		return fmt.Errorf("conta a pagar deve estar PENDENTE ou APROVADO para baixa, status: %s", cp.Status)
	}

	dataPagamento, err := time.Parse("2006-01-02", dto.DataPagamento)
	if err != nil {
		return fmt.Errorf("data_pagamento inválida: %w", err)
	}

	valorPago := decimal.NewFromFloat(dto.ValorPago)
	valorOriginal := cp.ValorBruto.Sub(cp.ValorPago)

	jurosMes := 0.01
	multaAtraso := 0.02
	if fiscalCfg, err := uc.FiscalRepo.GetFiscalConfig(ctx); err == nil && fiscalCfg != nil {
		if fiscalCfg.JurosMes > 0 {
			jurosMes = fiscalCfg.JurosMes
		}
		if fiscalCfg.MultaAtraso > 0 {
			multaAtraso = fiscalCfg.MultaAtraso
		}
	}

	var jurosDec, multaDec decimal.Decimal
	if dataPagamento.After(cp.DataVencimento) {
		daysLate := int(math.Ceil(dataPagamento.Sub(cp.DataVencimento).Hours() / 24))
		monthsLate := float64(daysLate) / 30.0
		jurosDec = valorOriginal.Mul(decimal.NewFromFloat(jurosMes)).Mul(decimal.NewFromFloat(monthsLate))
		multaDec = valorOriginal.Mul(decimal.NewFromFloat(multaAtraso))
	}

	params := repository.BaixaParams{
		ContaBancariaID: dto.ContaBancariaID,
		ValorPago:       valorPago.InexactFloat64(),
		Juros:           jurosDec.InexactFloat64(),
		Multa:           multaDec.InexactFloat64(),
		Desconto:        0,
		DataPagamento:   dataPagamento,
		Observacao:      dto.Observacao,
		BaixadoPor:      userID,
	}

	totalFluxo := valorPago.Add(jurosDec).Add(multaDec)

	// All writes in a single atomic transaction
	return uc.Repo.BaixarContaPagarAtomico(ctx, id, params, entity.FluxoCaixa{
		Data:            dataPagamento,
		Tipo:            entity.FluxoCaixaTipoSaida,
		Valor:           totalFluxo,
		ContaBancariaID: &dto.ContaBancariaID,
		ContasPagarID:   &id,
		Descricao:       dto.Observacao,
		Conciliado:      false,
	}, valorOriginal, dto.ContaBancariaID)
}
