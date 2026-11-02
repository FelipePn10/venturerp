package financial_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

// CreateContaPagarParceladaUseCase cria as parcelas de um título numa operação
// só. As parcelas chegam com valor e vencimento: "30% na entrada e o resto em
// 3 vezes" ou "3 vezes e 40% no final" são a mesma coisa aqui.
type CreateContaPagarParceladaUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CreateContaPagarParceladaUseCase) Execute(ctx context.Context, dto request.CreateContaPagarParceladaDTO) ([]*entity.ContaPagar, error) {
	if !uc.Auth.CanCreateContaPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.NumeroDocumento) == "" {
		return nil, errorsuc.NewValidationError("informe o número do documento")
	}
	emissao, msg := datetime.Obrigatoria("data de emissão", dto.DataEmissao)
	if msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	if len(dto.Parcelas) == 0 {
		return nil, errorsuc.NewValidationError("informe as parcelas do título")
	}
	bruto := decimal.NewFromFloat(dto.ValorBruto).Round(2)
	desconto := decimal.NewFromFloat(dto.Desconto).Round(2)
	liquido := bruto.Sub(desconto)
	if !liquido.IsPositive() {
		return nil, errorsuc.NewValidationError("o valor líquido do título precisa ser maior que zero")
	}

	vencimentos := make([]time.Time, len(dto.Parcelas))
	soma := decimal.Zero
	for i, p := range dto.Parcelas {
		venc, msg := datetime.Obrigatoria(fmt.Sprintf("vencimento da parcela %d", i+1), p.DataVencimento)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		if venc.Before(emissao) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a parcela %d vence antes da emissão do título", i+1))
		}
		if !p.Valor.Round(2).IsPositive() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a parcela %d precisa de valor maior que zero", i+1))
		}
		vencimentos[i] = venc
		soma = soma.Add(p.Valor.Round(2))
	}
	if !soma.Equal(liquido) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("as parcelas somam %s e o valor líquido do título é %s", soma.StringFixed(2), liquido.StringFixed(2)))
	}

	titulos, err := desdobrarTitulo(dto, userID, emissao, vencimentos, bruto, desconto, liquido)
	if err != nil {
		return nil, err
	}
	criador := CreateContaPagarUseCase{Repo: uc.Repo, Auth: uc.Auth}
	if err := criador.conferirPlanosDaEmpresa(ctx, titulos[0]); err != nil {
		return nil, err
	}
	return uc.Repo.CreateContasPagar(ctx, titulos)
}

// desdobrarTitulo reparte desconto e rateio entre as parcelas. Quem arredonda não
// perde centavo: a última parcela leva a diferença de cada total e, nas demais, a
// última conta do rateio fecha o valor da parcela.
func desdobrarTitulo(
	dto request.CreateContaPagarParceladaDTO, userID uuid.UUID, emissao time.Time, vencimentos []time.Time,
	bruto, desconto, liquido decimal.Decimal,
) ([]*entity.ContaPagar, error) {
	total := len(dto.Parcelas)
	titulos := make([]*entity.ContaPagar, 0, total)
	descontoUsado := decimal.Zero
	rateioUsado := make([]decimal.Decimal, len(dto.Rateios))
	for i, p := range dto.Parcelas {
		ultima := i == total-1
		valor := p.Valor.Round(2)
		descontoParcela := desconto.Sub(descontoUsado)
		if !ultima {
			descontoParcela = desconto.Mul(valor).Div(liquido).Round(2)
		}
		descontoUsado = descontoUsado.Add(descontoParcela)
		brutoParcela := valor.Add(descontoParcela)

		parte := dto.CreateContaPagarDTO
		parte.NumeroDocumento = fmt.Sprintf("%s %d/%d", strings.TrimSpace(dto.NumeroDocumento), i+1, total)
		parte.ValorBruto = brutoParcela.InexactFloat64()
		parte.Desconto = descontoParcela.InexactFloat64()
		parte.ParcelaNumero, parte.ParcelaTotal = int32(i+1), int32(total)
		c := novoTitulo(parte, userID, emissao, vencimentos[i])

		rateios := make([]request.RateioContaPagarDTO, 0, len(dto.Rateios))
		somaLinha := decimal.Zero
		for r, rt := range dto.Rateios {
			var v decimal.Decimal
			switch {
			case ultima:
				v = rt.Valor.Round(2).Sub(rateioUsado[r])
			case r == len(dto.Rateios)-1:
				v = brutoParcela.Sub(somaLinha)
			default:
				v = rt.Valor.Mul(brutoParcela).Div(bruto).Round(2)
			}
			rateioUsado[r] = rateioUsado[r].Add(v)
			somaLinha = somaLinha.Add(v)
			if v.IsPositive() {
				rateios = append(rateios, request.RateioContaPagarDTO{PlanoContasID: rt.PlanoContasID, CentroCustoID: rt.CentroCustoID, Valor: v})
			}
		}
		if err := aplicarRateio(c, rateios); err != nil {
			return nil, err
		}
		titulos = append(titulos, c)
	}
	return titulos, nil
}
