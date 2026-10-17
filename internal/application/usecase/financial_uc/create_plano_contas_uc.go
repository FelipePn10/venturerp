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

type CreatePlanoContasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CreatePlanoContasUseCase) Execute(
	ctx context.Context, dto request.CreatePlanoContasDTO,
) (*entity.PlanoContas, error) {
	if !uc.Auth.CanCreatePlanoContas(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	p := &entity.PlanoContas{
		Codigo:     strings.TrimSpace(dto.Codigo),
		Descricao:  strings.TrimSpace(dto.Descricao),
		Tipo:       strings.ToUpper(strings.TrimSpace(dto.Tipo)),
		Natureza:   strings.ToUpper(strings.TrimSpace(dto.Natureza)),
		ParentCode: textoOuNulo(dto.ParentCode),
		IsActive:   true,
	}
	if p.Codigo == "" || p.Descricao == "" {
		return nil, errorsuc.NewValidationError("código e descrição são obrigatórios")
	}
	if !tiposPlanoContas[p.Tipo] {
		return nil, errorsuc.NewValidationError("tipo inválido: use RECEITA, DESPESA, ATIVO, PASSIVO ou PATRIMONIO")
	}
	if !naturezasPlanoConta[p.Natureza] {
		return nil, errorsuc.NewValidationError("natureza inválida: use DEBITO ou CREDITO")
	}
	// Conta-pai que não existe deixava a árvore com um galho solto: a conta
	// aparecia no nível certo, mas fora de qualquer grupo nos relatórios.
	if err := conferirContaPai(ctx, uc.Repo, p.Codigo, p.ParentCode); err != nil {
		return nil, err
	}
	p.Nivel = dto.Nivel
	if p.Nivel <= 0 {
		p.Nivel = nivelDoCodigo(p.Codigo)
	}

	p.CreatedAt = time.Now()

	created, err := uc.Repo.CreatePlanoContas(ctx, p)
	if err != nil {
		return nil, err
	}
	return created, nil
}
