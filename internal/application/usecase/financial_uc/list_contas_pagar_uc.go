package financial_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ListContasPagarUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListContasPagarUseCase) Execute(
	ctx context.Context, dto request.ListContasPagarFilter,
) ([]*entity.ContaPagar, error) {
	if !uc.Auth.CanListContasPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	filters := repository.CPFilter{
		Status:          dto.Status,
		StatusAprovacao: dto.StatusAprovacao,
		FornecedorID:    dto.FornecedorID,
		PlanoContasID:   dto.PlanoContasID,
		CentroCustoID:   dto.CentroCustoID,
		TipoDocumento:   dto.TipoDocumento,
		Documento:       dto.Documento,
		ValorMinimo:     dto.ValorMinimo,
		ValorMaximo:     dto.ValorMaximo,
		DateField:       campoDeData(dto.DateField),
	}
	if dto.SomenteVencidos != nil {
		filters.SomenteVencidos = *dto.SomenteVencidos
	}
	inicio, err := dataDoFiltro(dto.StartDate, "data inicial")
	if err != nil {
		return nil, err
	}
	filters.StartDate = inicio
	fim, err := dataDoFiltro(dto.EndDate, "data final")
	if err != nil {
		return nil, err
	}
	filters.EndDate = fim
	if inicio != nil && fim != nil && fim.Before(*inicio) {
		return nil, errorsuc.NewValidationError("a data final do período é anterior à data inicial")
	}
	if filters.ValorMinimo != nil && filters.ValorMaximo != nil && *filters.ValorMaximo < *filters.ValorMinimo {
		return nil, errorsuc.NewValidationError("o valor máximo do filtro é menor que o valor mínimo")
	}

	return uc.Repo.ListContasPagar(ctx, filters)
}

// dataDoFiltro converte a data do filtro. Antes o erro de parse era descartado, e
// uma data digitada errada virava 01/01/0001 — um período que não devolve nada,
// sem dizer por quê. Recusar é o único jeito de quem usa descobrir o erro.
func dataDoFiltro(bruto *string, campo string) (*time.Time, error) {
	if bruto == nil || strings.TrimSpace(*bruto) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*bruto))
	if err != nil {
		return nil, errorsuc.NewValidationError(campo + " inválida: use o formato AAAA-MM-DD")
	}
	return &t, nil
}

// campoDeData fecha o domínio: qualquer coisa fora de EMISSAO cai em vencimento,
// que é o padrão histórico da consulta.
func campoDeData(bruto *string) repository.DateField {
	if bruto != nil && strings.EqualFold(strings.TrimSpace(*bruto), string(repository.DateFieldEmissao)) {
		return repository.DateFieldEmissao
	}
	return repository.DateFieldVencimento
}
