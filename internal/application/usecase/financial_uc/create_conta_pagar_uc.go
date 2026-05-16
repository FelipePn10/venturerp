package financial_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/shopspring/decimal"
)

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

	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	created, err := uc.Repo.CreateContaPagar(ctx, c)
	if err != nil {
		return nil, err
	}
	return created, nil
}
