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
	// Contabil, quando presente, contabiliza o recebimento na mesma transação.
	Contabil Contabil
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
		return errorsuc.NewValidationError(fmt.Sprintf("conta a receber deve estar PENDENTE ou APROVADO para baixa, status: %s", cr.Status))
	}

	dataRecebimento, err := time.Parse("2006-01-02", dto.DataRecebimento)
	if err != nil {
		return errorsuc.NewValidationError("data de recebimento inválida: use o formato AAAA-MM-DD")
	}

	valorRecebido := decimal.NewFromFloat(dto.ValorRecebido).Round(2)
	desconto := dto.Desconto.Round(2)
	valorOriginal := cr.ValorBruto.Sub(cr.ValorRecebido).Sub(cr.Desconto)
	if !valorRecebido.IsPositive() {
		return errorsuc.NewValidationError("informe o valor recebido")
	}
	if desconto.IsNegative() {
		return errorsuc.NewValidationError("o desconto não pode ser negativo")
	}
	if valorRecebido.Add(desconto).GreaterThan(valorOriginal.Add(decimal.NewFromFloat(0.005))) {
		return errorsuc.NewValidationError(fmt.Sprintf("valor recebido + desconto (%s) passa do saldo do título (%s)", valorRecebido.Add(desconto).StringFixed(2), valorOriginal.StringFixed(2)))
	}

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
		// Juros e multa sobre o que está sendo quitado agora: na baixa parcial o
		// saldo vira outro título e paga os seus quando for quitado. Calcular
		// sobre o saldo inteiro cobrava a multa duas vezes sobre o mesmo valor.
		base := valorRecebido.Add(desconto)
		jurosDec = base.Mul(decimal.NewFromFloat(jurosMes)).Mul(decimal.NewFromFloat(monthsLate))
		multaDec = base.Mul(decimal.NewFromFloat(multaAtraso))
	}

	params := repository.BaixaParams{
		ContaBancariaID: dto.ContaBancariaID,
		ValorPago:       valorRecebido.InexactFloat64(),
		Juros:           jurosDec.InexactFloat64(),
		Multa:           multaDec.InexactFloat64(),
		Desconto:        desconto.InexactFloat64(),
		DataPagamento:   dataRecebimento,
		Observacao:      dto.Observacao,
		BaixadoPor:      userID,
	}

	jurosDec, multaDec = jurosDec.Round(2), multaDec.Round(2)
	params.Juros, params.Multa = jurosDec.InexactFloat64(), multaDec.InexactFloat64()
	totalFluxo := valorRecebido.Add(jurosDec).Add(multaDec)

	lote, err := loteRecebimento(ctx, uc.Contabil, cr, dto.ContaBancariaID, valorRecebido, jurosDec.Add(multaDec), desconto, dataRecebimento)
	if err != nil {
		return err
	}

	return uc.Repo.BaixarContaReceberAtomico(ctx, id, params, entity.FluxoCaixa{
		Data:            dataRecebimento,
		Tipo:            entity.FluxoCaixaTipoEntrada,
		Valor:           totalFluxo,
		ContaBancariaID: &dto.ContaBancariaID,
		ContasReceberID: &id,
		Descricao:       dto.Observacao,
		Conciliado:      false,
	}, valorOriginal, dto.ContaBancariaID, lote)
}
