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

type CreateContaReceberUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CreateContaReceberUseCase) Execute(
	ctx context.Context, dto request.CreateContaReceberDTO,
) (*entity.ContaReceber, error) {
	if !uc.Auth.CanCreateContaReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	dataEmissao, _ := time.Parse("2006-01-02", dto.DataEmissao)
	dataVencimento, _ := time.Parse("2006-01-02", dto.DataVencimento)

	c := &entity.ContaReceber{
		NumeroDocumento: dto.NumeroDocumento,
		ClienteID:       dto.ClienteID,
		FiscalExitID:    dto.FiscalExitID,
		SalesOrderID:    dto.SalesOrderID,
		DataLancamento:  time.Now(),
		DataEmissao:     dataEmissao,
		DataVencimento:  dataVencimento,
		ValorBruto:      decimal.NewFromFloat(dto.ValorBruto),
		Desconto:        decimal.NewFromFloat(dto.Desconto),
		Juros:           decimal.Zero,
		Multa:           decimal.Zero,
		ValorRecebido:   decimal.Zero,
		ParcelaNumero:   dto.ParcelaNumero,
		ParcelaTotal:    dto.ParcelaTotal,
		FormaPagamento:  dto.FormaPagamento,
		PlanoContasID:   dto.PlanoContasID,
		CentroCustoID:   dto.CentroCustoID,
		Status:          entity.ContaReceberStatusPendente,
		IsActive:        true,
		CriadoPor:       userID,
	}

	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	created, err := uc.Repo.CreateContaReceber(ctx, c)
	if err != nil {
		return nil, err
	}
	return created, nil
}
