package financial_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/shopspring/decimal"
)

// CreateAdiantamentoUseCase registers an advance and posts its cash movement.
type CreateAdiantamentoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CreateAdiantamentoUseCase) Execute(ctx context.Context, dto request.CreateAdiantamentoDTO) (*response.AdiantamentoResponse, error) {
	if !uc.Auth.CanBaixarContaPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	tipo := entity.AdiantamentoTipo(dto.Tipo)
	if tipo != entity.AdiantamentoTipoPagar && tipo != entity.AdiantamentoTipoReceber {
		return nil, fmt.Errorf("tipo inválido: use PAGAR ou RECEBER")
	}
	if dto.ValorOriginal <= 0 {
		return nil, fmt.Errorf("valor_original deve ser maior que zero")
	}
	if dto.ContaBancariaID == 0 {
		return nil, fmt.Errorf("conta_bancaria_id é obrigatória")
	}

	data, err := time.Parse("2006-01-02", dto.DataAdiantamento)
	if err != nil {
		return nil, fmt.Errorf("data_adiantamento inválida (use AAAA-MM-DD): %w", err)
	}

	adv := &entity.Adiantamento{
		Tipo:             tipo,
		ParceiroID:       dto.ParceiroID,
		ContaBancariaID:  dto.ContaBancariaID,
		NumeroDocumento:  dto.NumeroDocumento,
		DataAdiantamento: data,
		ValorOriginal:    decimal.NewFromFloat(dto.ValorOriginal),
		Descricao:        dto.Descricao,
		CreatedBy:        userID,
	}

	// Advance to supplier = cash OUT; advance from customer = cash IN.
	fcTipo := entity.FluxoCaixaTipoSaida
	if tipo == entity.AdiantamentoTipoReceber {
		fcTipo = entity.FluxoCaixaTipoEntrada
	}
	desc := "Adiantamento"
	if dto.Descricao != nil && *dto.Descricao != "" {
		desc = *dto.Descricao
	}
	contaID := dto.ContaBancariaID
	fc := entity.FluxoCaixa{
		Data:            data,
		Tipo:            fcTipo,
		Valor:           decimal.NewFromFloat(dto.ValorOriginal),
		ContaBancariaID: &contaID,
		Descricao:       &desc,
	}

	created, err := uc.Repo.CreateAdiantamentoAtomico(ctx, adv, fc)
	if err != nil {
		return nil, err
	}
	return toAdiantamentoResponse(created), nil
}

// ListAdiantamentosUseCase lists advances, optionally filtered by tipo/parceiro.
type ListAdiantamentosUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListAdiantamentosUseCase) Execute(ctx context.Context, tipo *string, parceiroID *int64) ([]*response.AdiantamentoResponse, error) {
	if !uc.Auth.CanGetContaPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListAdiantamentos(ctx, tipo, parceiroID)
	if err != nil {
		return nil, err
	}
	return toAdiantamentoResponses(list), nil
}

// GetAdiantamentoUseCase returns one advance with its applications.
type GetAdiantamentoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

type AdiantamentoDetail struct {
	*response.AdiantamentoResponse
	Aplicacoes []*response.AdiantamentoAplicacaoResponse `json:"aplicacoes"`
}

func (uc *GetAdiantamentoUseCase) Execute(ctx context.Context, id int64) (*AdiantamentoDetail, error) {
	if !uc.Auth.CanGetContaPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	adv, err := uc.Repo.GetAdiantamento(ctx, id)
	if err != nil {
		return nil, err
	}
	aplicacoes, err := uc.Repo.ListAplicacoesByAdiantamento(ctx, id)
	if err != nil {
		return nil, err
	}
	aps := make([]*response.AdiantamentoAplicacaoResponse, 0, len(aplicacoes))
	for _, a := range aplicacoes {
		aps = append(aps, toAdiantamentoAplicacaoResponse(a))
	}
	return &AdiantamentoDetail{AdiantamentoResponse: toAdiantamentoResponse(adv), Aplicacoes: aps}, nil
}

// AplicarAdiantamentoUseCase applies an advance balance onto a title.
type AplicarAdiantamentoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *AplicarAdiantamentoUseCase) Execute(ctx context.Context, advID int64, dto request.AplicarAdiantamentoDTO) (*response.AdiantamentoAplicacaoResponse, error) {
	contaTipo := dto.ContaTipo
	if contaTipo == string(entity.AdiantamentoTipoReceber) {
		if !uc.Auth.CanBaixarContaReceber(ctx) {
			return nil, errorsuc.ErrUnauthorized
		}
	} else if contaTipo == string(entity.AdiantamentoTipoPagar) {
		if !uc.Auth.CanBaixarContaPagar(ctx) {
			return nil, errorsuc.ErrUnauthorized
		}
	} else {
		return nil, fmt.Errorf("conta_tipo inválido: use PAGAR ou RECEBER")
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.Valor <= 0 {
		return nil, fmt.Errorf("valor deve ser maior que zero")
	}

	data := time.Now()
	if dto.DataAplicacao != nil && *dto.DataAplicacao != "" {
		data, err = time.Parse("2006-01-02", *dto.DataAplicacao)
		if err != nil {
			return nil, fmt.Errorf("data_aplicacao inválida (use AAAA-MM-DD): %w", err)
		}
	}

	created, err := uc.Repo.AplicarAdiantamentoAtomico(ctx, advID, contaTipo, dto.ContaID, decimal.NewFromFloat(dto.Valor), userID, data)
	if err != nil {
		return nil, err
	}
	return toAdiantamentoAplicacaoResponse(created), nil
}
