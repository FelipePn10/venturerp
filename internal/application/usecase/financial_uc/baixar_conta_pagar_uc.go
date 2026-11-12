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
	// Contabil, quando presente, contabiliza o pagamento na mesma transação.
	Contabil Contabil
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

	// Título vencido continua em aberto: é justamente o que mais precisa ser
	// pago. Antes só PENDENTE e APROVADO passavam e o vencido não tinha baixa.
	if cp.Status != entity.ContaPagarStatusPendente && cp.Status != entity.ContaPagarStatusAprovado && cp.Status != entity.ContaPagarStatusVencido {
		return errorsuc.NewValidationError(fmt.Sprintf("só título em aberto (pendente, aprovado ou vencido) pode ser pago; este está %s", cp.Status))
	}

	dataPagamento, err := time.Parse("2006-01-02", dto.DataPagamento)
	if err != nil {
		return errorsuc.NewValidationError("data de pagamento inválida: use o formato AAAA-MM-DD")
	}
	if err := conferirContaBancaria(ctx, uc.Repo, dto.ContaBancariaID); err != nil {
		return err
	}

	valorPago := dto.ValorPago.Round(2)
	desconto := dto.Desconto.Round(2)
	// O que ainda se deve: o bruto menos o já pago, o desconto já dado e o que
	// um adiantamento abateu.
	valorOriginal := cp.ValorBruto.Sub(cp.ValorPago).Sub(cp.Desconto).Sub(cp.ValorAdiantamentoAbatido)
	if !valorPago.IsPositive() {
		return errorsuc.NewValidationError("informe o valor pago")
	}
	if desconto.IsNegative() {
		return errorsuc.NewValidationError("o desconto não pode ser negativo")
	}
	if valorPago.Add(desconto).GreaterThan(valorOriginal.Add(decimal.NewFromFloat(0.005))) {
		return errorsuc.NewValidationError(fmt.Sprintf("valor pago + desconto (%s) passa do saldo do título (%s)", valorPago.Add(desconto).StringFixed(2), valorOriginal.StringFixed(2)))
	}

	jurosMes := decimal.New(1, -2)
	multaAtraso := decimal.New(2, -2)
	if fiscalCfg, err := uc.FiscalRepo.GetFiscalConfig(ctx); err == nil && fiscalCfg != nil {
		if fiscalCfg.JurosMes.IsPositive() {
			jurosMes = fiscalCfg.JurosMes
		}
		if fiscalCfg.MultaAtraso.IsPositive() {
			multaAtraso = fiscalCfg.MultaAtraso
		}
	}

	var jurosDec, multaDec decimal.Decimal
	if dataPagamento.After(cp.DataVencimento) {
		daysLate := int(math.Ceil(dataPagamento.Sub(cp.DataVencimento).Hours() / 24))
		monthsLate := float64(daysLate) / 30.0
		// Juros e multa sobre o que está sendo quitado agora: na baixa parcial o
		// saldo vira outro título e paga os seus quando for quitado. Calcular
		// sobre o saldo inteiro cobrava a multa duas vezes sobre o mesmo valor.
		base := valorPago.Add(desconto)
		jurosDec = base.Mul(jurosMes).Mul(decimal.NewFromFloat(monthsLate))
		multaDec = base.Mul(multaAtraso)
	}

	params := repository.BaixaParams{
		ContaBancariaID: dto.ContaBancariaID,
		ValorPago:       valorPago,
		Juros:           jurosDec,
		Multa:           multaDec,
		Desconto:        desconto,
		DataPagamento:   dataPagamento,
		Observacao:      dto.Observacao,
		BaixadoPor:      userID,
	}

	jurosDec, multaDec = jurosDec.Round(2), multaDec.Round(2)
	params.Juros, params.Multa = jurosDec, multaDec
	totalFluxo := valorPago.Add(jurosDec).Add(multaDec)

	lote, err := lotePagamento(ctx, uc.Contabil, cp, dto.ContaBancariaID, valorPago, jurosDec.Add(multaDec), desconto, dataPagamento)
	if err != nil {
		return err
	}

	// All writes in a single atomic transaction
	return uc.Repo.BaixarContaPagarAtomico(ctx, id, params, entity.FluxoCaixa{
		Data:            dataPagamento,
		Tipo:            entity.FluxoCaixaTipoSaida,
		Valor:           totalFluxo,
		ContaBancariaID: &dto.ContaBancariaID,
		ContasPagarID:   &id,
		Descricao:       dto.Observacao,
		Conciliado:      false,
	}, valorOriginal, dto.ContaBancariaID, lote)
}
