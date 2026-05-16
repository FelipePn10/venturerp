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

type BaixarContaReceberUseCase struct {
	Repo       repository.FinancialRepository
	Auth       ports.AuthService
	FiscalRepo fiscalrepo.FiscalRepository
}

func (uc *BaixarContaReceberUseCase) Execute(ctx context.Context, id int64, dto request.BaixarContaReceberDTO) error {
	if !uc.Auth.CanBaixarContaReceber(ctx) {
		return errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}

	cr, err := uc.Repo.GetContaReceber(ctx, id)
	if err != nil {
		return err
	}

	if cr.Status != entity.ContaReceberStatusPendente && cr.Status != entity.ContaReceberStatusAprovado {
		return fmt.Errorf("conta a receber deve estar PENDENTE ou APROVADO para baixa")
	}

	dataRecebimento, _ := time.Parse("2006-01-02", dto.DataRecebimento)
	valorRecebido := decimal.NewFromFloat(dto.ValorRecebido)
	valorOriginal := cr.ValorBruto.Sub(cr.ValorRecebido)
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

	if dataRecebimento.After(cr.DataVencimento) {
		daysLate := int(math.Ceil(dataRecebimento.Sub(cr.DataVencimento).Hours() / 24))
		monthsLate := float64(daysLate) / 30.0
		jurosVal := valorOriginal.Mul(decimal.NewFromFloat(jurosMes)).Mul(decimal.NewFromFloat(monthsLate))
		jurosDec = jurosVal
		multaDec = valorOriginal.Mul(decimal.NewFromFloat(multaAtraso))
	}

	// Handle partial payment
	if valorRecebido.LessThan(valorOriginal) {
		remaining := valorOriginal.Sub(valorRecebido)
		if err := uc.Repo.BaixarContaReceber(ctx, id, repository.BaixaParams{
			ContaBancariaID: dto.ContaBancariaID,
			ValorPago:       valorRecebido.InexactFloat64(),
			Juros:           jurosDec.InexactFloat64(),
			Multa:           multaDec.InexactFloat64(),
			Desconto:        0,
			DataPagamento:   dataRecebimento,
			Observacao:      dto.Observacao,
			BaixadoPor:      userID,
		}); err != nil {
			return err
		}

		newCR := &entity.ContaReceber{
			NumeroDocumento: func() *string { if cr.NumeroDocumento != nil { v := *cr.NumeroDocumento + "/P"; return &v }; return nil }(),
			ClienteID:       cr.ClienteID,
			FiscalExitID:    cr.FiscalExitID,
			SalesOrderID:    cr.SalesOrderID,
			DataLancamento:  time.Now(),
			DataEmissao:     cr.DataEmissao,
			DataVencimento:  cr.DataVencimento,
			ValorBruto:      remaining,
			Desconto:        decimal.Zero,
			Juros:           decimal.Zero,
			Multa:           decimal.Zero,
			ValorRecebido:   decimal.Zero,
			ParcelaNumero:   cr.ParcelaNumero + 1,
			ParcelaTotal:    cr.ParcelaTotal,
			FormaPagamento:  cr.FormaPagamento,
			PlanoContasID:   cr.PlanoContasID,
			CentroCustoID:   cr.CentroCustoID,
			Status:          entity.ContaReceberStatusPendente,
			IsActive:        true,
			CriadoPor:       cr.CriadoPor,
		}
		newCR.CreatedAt = time.Now()
		newCR.UpdatedAt = time.Now()

		if _, err := uc.Repo.CreateContaReceber(ctx, newCR); err != nil {
			return err
		}
	} else {
		params := repository.BaixaParams{
			ContaBancariaID: dto.ContaBancariaID,
			ValorPago:       valorRecebido.InexactFloat64(),
			Juros:           jurosDec.InexactFloat64(),
			Multa:           multaDec.InexactFloat64(),
			Desconto:        0,
			DataPagamento:   dataRecebimento,
			Observacao:      dto.Observacao,
			BaixadoPor:      userID,
		}
		if err := uc.Repo.BaixarContaReceber(ctx, id, params); err != nil {
			return err
		}
	}

	// Create FluxoCaixa entry
	totalFluxo := valorRecebido.Add(jurosDec).Add(multaDec)
	fc := &entity.FluxoCaixa{
		Data:             dataRecebimento,
		Tipo:             entity.FluxoCaixaTipoEntrada,
		Valor:            totalFluxo,
		ContaBancariaID:  &dto.ContaBancariaID,
		ContasReceberID:  &id,
		Descricao:        dto.Observacao,
		Conciliado:       false,
	}
	if _, err := uc.Repo.CreateFluxoCaixa(ctx, fc); err != nil {
		return err
	}

	currentSaldo, err := uc.Repo.GetSaldoConta(ctx, dto.ContaBancariaID)
	if err != nil {
		return err
	}
	novoSaldo := currentSaldo + totalFluxo.InexactFloat64()
	return uc.Repo.UpdateSaldo(ctx, dto.ContaBancariaID, novoSaldo)
}
