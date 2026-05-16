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

type CreateContaBancariaUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CreateContaBancariaUseCase) Execute(
	ctx context.Context, dto request.CreateContaBancariaDTO,
) (*entity.ContaBancaria, error) {
	if !uc.Auth.CanCreateContaBancaria(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	c := &entity.ContaBancaria{
		Banco:        dto.Banco,
		Agencia:      dto.Agencia,
		Conta:        dto.Conta,
		Digito:       dto.Digito,
		Descricao:    dto.Descricao,
		Titular:      dto.Titular,
		SaldoInicial: decimal.NewFromFloat(dto.SaldoInicial),
		ChavePix:     dto.ChavePix,
		TipoChavePix: dto.TipoChavePix,
		IsActive:     true,
		CreatedBy:    userID,
	}

	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	created, err := uc.Repo.CreateContaBancaria(ctx, c)
	if err != nil {
		return nil, err
	}
	return created, nil
}
