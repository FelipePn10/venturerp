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
		return fmt.Errorf("conta a pagar deve estar PENDENTE ou APROVADO para baixa")
	}

	dataPagamento, _ := time.Parse("2006-01-02", dto.DataPagamento)
	valorPago := decimal.NewFromFloat(dto.ValorPago)
	valorOriginal := cp.ValorBruto.Sub(cp.ValorPago)
	var jurosDec, multaDec decimal.Decimal

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

	if dataPagamento.After(cp.DataVencimento) {
		daysLate := int(math.Ceil(dataPagamento.Sub(cp.DataVencimento).Hours() / 24))
		monthsLate := float64(daysLate) / 30.0
		jurosVal := valorOriginal.Mul(decimal.NewFromFloat(jurosMes)).Mul(decimal.NewFromFloat(monthsLate))
		jurosDec = jurosVal
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

	// Handle partial payment
	if valorPago.LessThan(valorOriginal) {
		remaining := valorOriginal.Sub(valorPago)
		if err := uc.Repo.BaixarContaPagar(ctx, id, repository.BaixaParams{
			ContaBancariaID: dto.ContaBancariaID,
			ValorPago:       valorPago.InexactFloat64(),
			Juros:           jurosDec.InexactFloat64(),
			Multa:           multaDec.InexactFloat64(),
			Desconto:        0,
			DataPagamento:   dataPagamento,
			Observacao:      dto.Observacao,
			BaixadoPor:      userID,
		}); err != nil {
			return err
		}

		// Create new CP for remaining
		newCP := &entity.ContaPagar{
			NumeroDocumento: cp.NumeroDocumento + "/P",
			TipoDocumento:   cp.TipoDocumento,
			FornecedorID:    cp.FornecedorID,
			FiscalEntryID:   cp.FiscalEntryID,
			PurchaseOrderID: cp.PurchaseOrderID,
			DataLancamento:  time.Now(),
			DataEmissao:     cp.DataEmissao,
			DataVencimento:  cp.DataVencimento,
			ValorBruto:      remaining,
			Desconto:        decimal.Zero,
			Juros:           decimal.Zero,
			Multa:           decimal.Zero,
			ValorPago:       decimal.Zero,
			ParcelaNumero:   cp.ParcelaNumero + 1,
			ParcelaTotal:    cp.ParcelaTotal,
			FormaPagamento:  cp.FormaPagamento,
			PlanoContasID:   cp.PlanoContasID,
			CentroCustoID:   cp.CentroCustoID,
			StatusAprovacao: entity.AprovacaoAprovado,
			Status:          entity.ContaPagarStatusPendente,
			IsActive:        true,
			CriadoPor:       cp.CriadoPor,
		}
		newCP.CreatedAt = time.Now()
		newCP.UpdatedAt = time.Now()

		if _, err := uc.Repo.CreateContaPagar(ctx, newCP); err != nil {
			return err
		}
	} else {
		if err := uc.Repo.BaixarContaPagar(ctx, id, params); err != nil {
			return err
		}
	}

	// Create FluxoCaixa entry
	totalFluxo := valorPago.Add(jurosDec).Add(multaDec)
	fc := &entity.FluxoCaixa{
		Data:            dataPagamento,
		Tipo:            entity.FluxoCaixaTipoSaida,
		Valor:           totalFluxo,
		ContaBancariaID: &dto.ContaBancariaID,
		ContasPagarID:   &id,
		Descricao:       dto.Observacao,
		Conciliado:      false,
	}
	if _, err := uc.Repo.CreateFluxoCaixa(ctx, fc); err != nil {
		return err
	}

	// Update saldo conta bancaria
	currentSaldo, err := uc.Repo.GetSaldoConta(ctx, dto.ContaBancariaID)
	if err != nil {
		return err
	}
	novoSaldo := currentSaldo - totalFluxo.InexactFloat64()
	return uc.Repo.UpdateSaldo(ctx, dto.ContaBancariaID, novoSaldo)
}
