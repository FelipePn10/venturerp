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
		return fmt.Errorf("conta a receber deve estar PENDENTE ou APROVADO para baixa, status: %s", cr.Status)
	}

	dataRecebimento, err := time.Parse("2006-01-02", dto.DataRecebimento)
	if err != nil {
		return fmt.Errorf("data_recebimento inválida: %w", err)
	}

	valorRecebido := decimal.NewFromFloat(dto.ValorRecebido)
	valorOriginal := cr.ValorBruto.Sub(cr.ValorRecebido)

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
	if dataRecebimento.After(cr.DataVencimento) {
		daysLate := int(math.Ceil(dataRecebimento.Sub(cr.DataVencimento).Hours() / 24))
		monthsLate := float64(daysLate) / 30.0
		jurosDec = valorOriginal.Mul(decimal.NewFromFloat(jurosMes)).Mul(decimal.NewFromFloat(monthsLate))
		multaDec = valorOriginal.Mul(decimal.NewFromFloat(multaAtraso))
	}

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

	totalFluxo := valorRecebido.Add(jurosDec).Add(multaDec)

	return uc.Repo.BaixarContaReceberAtomico(ctx, id, params, entity.FluxoCaixa{
		Data:            dataRecebimento,
		Tipo:            entity.FluxoCaixaTipoEntrada,
		Valor:           totalFluxo,
		ContaBancariaID: &dto.ContaBancariaID,
		ContasReceberID: &id,
		Descricao:       dto.Observacao,
		Conciliado:      false,
	}, valorOriginal, dto.ContaBancariaID)
}
