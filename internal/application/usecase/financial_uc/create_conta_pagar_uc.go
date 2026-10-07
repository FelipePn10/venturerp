package financial_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/shopspring/decimal"
)

// aplicarRateio valida a distribuição do título por plano de contas: cada
// parte com plano e valor positivo, e a soma fechando com o valor bruto. Com
// um plano só, o título leva o plano também na capa.
func aplicarRateio(c *entity.ContaPagar, in []request.RateioContaPagarDTO) error {
	if len(in) == 0 {
		return nil
	}
	soma := decimal.Zero
	vistos := map[[2]int64]bool{}
	for _, r := range in {
		if r.PlanoContasID <= 0 {
			return errorsuc.NewValidationError("informe o plano de contas de cada parte do rateio")
		}
		v := r.Valor.Round(2)
		if !v.IsPositive() {
			return errorsuc.NewValidationError("cada parte do rateio precisa de valor maior que zero")
		}
		cc := int64(0)
		if r.CentroCustoID != nil {
			cc = *r.CentroCustoID
		}
		chave := [2]int64{r.PlanoContasID, cc}
		if vistos[chave] {
			return errorsuc.NewValidationError("o mesmo plano de contas/centro de custo aparece duas vezes no rateio")
		}
		vistos[chave] = true
		soma = soma.Add(v)
		c.Rateios = append(c.Rateios, entity.RateioContaPagar{PlanoContasID: r.PlanoContasID, CentroCustoID: r.CentroCustoID, Valor: v})
	}
	if soma.Sub(c.ValorBruto.Round(2)).Abs().GreaterThan(decimal.NewFromFloat(0.009)) {
		return errorsuc.NewValidationError(fmt.Sprintf("o rateio soma %s e o título vale %s", soma.StringFixed(2), c.ValorBruto.StringFixed(2)))
	}
	if len(c.Rateios) == 1 {
		plano := c.Rateios[0].PlanoContasID
		c.PlanoContasID, c.CentroCustoID = &plano, c.Rateios[0].CentroCustoID
	} else {
		c.PlanoContasID, c.CentroCustoID = nil, nil
	}
	return nil
}

type CreateContaPagarUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CreateContaPagarUseCase) Execute(
	ctx context.Context, dto request.CreateContaPagarDTO,
) (*entity.ContaPagar, error) {
	if !uc.Auth.CanCreateContaPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	dataEmissao, _ := time.Parse("2006-01-02", dto.DataEmissao)
	dataVencimento, _ := time.Parse("2006-01-02", dto.DataVencimento)

	c := &entity.ContaPagar{
		NumeroDocumento: dto.NumeroDocumento,
		TipoDocumento:   dto.TipoDocumento,
		FornecedorID:    dto.FornecedorID,
		FiscalEntryID:   dto.FiscalEntryID,
		PurchaseOrderID: dto.PurchaseOrderID,
		DataLancamento:  time.Now(),
		DataEmissao:     dataEmissao,
		DataVencimento:  dataVencimento,
		ValorBruto:      decimal.NewFromFloat(dto.ValorBruto),
		Desconto:        decimal.NewFromFloat(dto.Desconto),
		Juros:           decimal.Zero,
		Multa:           decimal.Zero,
		ValorPago:       decimal.Zero,
		ParcelaNumero:   dto.ParcelaNumero,
		ParcelaTotal:    dto.ParcelaTotal,
		FormaPagamento:  dto.FormaPagamento,
		PlanoContasID:   dto.PlanoContasID,
		CentroCustoID:   dto.CentroCustoID,
		StatusAprovacao: entity.AprovacaoPendente,
		Status:          entity.ContaPagarStatusPendente,
		IsActive:        true,
		CriadoPor:       userID,
		Observacao:      dto.Observacao,
	}

	if err := aplicarRateio(c, dto.Rateios); err != nil {
		return nil, err
	}
	if err := uc.conferirPlanosDaEmpresa(ctx, c); err != nil {
		return nil, err
	}

	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	created, err := uc.Repo.CreateContaPagar(ctx, c)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// conferirPlanosDaEmpresa recusa plano de contas ou centro de custo de outra
// empresa: o id existe no banco (a chave estrangeira aceita), mas não é desta
// empresa, e o título apareceria no plano de contas de outro CNPJ.
func (uc *CreateContaPagarUseCase) conferirPlanosDaEmpresa(ctx context.Context, c *entity.ContaPagar) error {
	if len(c.Rateios) == 0 {
		return nil
	}
	planos, err := uc.Repo.ListPlanoContas(ctx)
	if err != nil {
		return err
	}
	centros, err := uc.Repo.ListCentrosCusto(ctx)
	if err != nil {
		return err
	}
	planoOK := map[int64]bool{}
	for _, p := range planos {
		planoOK[p.ID] = p.IsActive
	}
	centroOK := map[int64]bool{}
	for _, cc := range centros {
		centroOK[cc.ID] = cc.IsActive
	}
	for _, r := range c.Rateios {
		if !planoOK[r.PlanoContasID] {
			return errorsuc.NewValidationError(fmt.Sprintf("o plano de contas %d não existe nesta empresa ou está inativo", r.PlanoContasID))
		}
		if r.CentroCustoID != nil && !centroOK[*r.CentroCustoID] {
			return errorsuc.NewValidationError(fmt.Sprintf("o centro de custo %d não existe nesta empresa ou está inativo", *r.CentroCustoID))
		}
	}
	return nil
}
