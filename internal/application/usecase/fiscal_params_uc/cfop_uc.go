package fiscal_params_uc

import (
	"context"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/shared/ptrutil"
)

type CFOPUseCase struct {
	Repo repository.FiscalParamsRepository
	Auth catalogAuth
}

func (uc *CFOPUseCase) Create(ctx context.Context, dto request.CreateCFOPDTO) (*response.CFOPResponse, error) {
	if err := podeAlterarCatalogo(ctx, uc.Auth); err != nil {
		return nil, err
	}
	if dto.Description == "" {
		return nil, errorsuc.NewValidationError("informe a descrição")
	}
	c := &entity.CFOP{
		Code:            dto.Code,
		Description:     dto.Description,
		DescriptionFull: dto.DescriptionFull,
		OrigemClasIPI:   dto.OrigemClasIPI,
		CodigoAnexoSN:   dto.CodigoAnexoSN,
		IsActive:        true,
	}
	if err := normalizaCFOP(c); err != nil {
		return nil, err
	}
	created, err := uc.Repo.CreateCFOP(ctx, c)
	if err != nil {
		return nil, err
	}
	return toCFOPResponse(created), nil
}

func (uc *CFOPUseCase) Update(ctx context.Context, dto request.UpdateCFOPDTO) (*response.CFOPResponse, error) {
	if err := podeAlterarCatalogo(ctx, uc.Auth); err != nil {
		return nil, err
	}
	c := &entity.CFOP{
		ID:              dto.ID,
		Code:            dto.Code,
		Description:     dto.Description,
		DescriptionFull: dto.DescriptionFull,
		OrigemClasIPI:   dto.OrigemClasIPI,
		CodigoAnexoSN:   dto.CodigoAnexoSN,
		IsActive:        ptrutil.BoolOrTrue(dto.IsActive),
	}
	if dto.ID <= 0 {
		return nil, errorsuc.NewValidationError("informe o CFOP a alterar")
	}
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("informe a descrição")
	}
	// Antes o campo de lista vazio ia direto para o banco e voltava "valor
	// inválido para enum"; agora vale o mesmo padrão do cadastro.
	if err := normalizaCFOP(c); err != nil {
		return nil, err
	}
	// O que não veio no corpo fica como está: a nota explicativa da tabela
	// oficial e os campos que a tela não mostra não se apagam numa edição.
	if atual, err := uc.Repo.GetCFOPByCode(ctx, dto.Code); err == nil && atual != nil && atual.ID == dto.ID {
		if c.DescriptionFull == nil {
			c.DescriptionFull = atual.DescriptionFull
		}
		if c.OrigemClasIPI == nil {
			c.OrigemClasIPI = atual.OrigemClasIPI
		}
		if c.CodigoAnexoSN == nil {
			c.CodigoAnexoSN = atual.CodigoAnexoSN
		}
	}
	updated, err := uc.Repo.UpdateCFOP(ctx, c)
	if err != nil {
		return nil, err
	}
	return toCFOPResponse(updated), nil
}

func (uc *CFOPUseCase) GetByCode(ctx context.Context, code int32) (*response.CFOPResponse, error) {
	c, err := uc.Repo.GetCFOPByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toCFOPResponse(c), nil
}

func (uc *CFOPUseCase) List(ctx context.Context, onlyActive bool) ([]*response.CFOPResponse, error) {
	list, err := uc.Repo.ListCFOPs(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toCFOPResponses(list), nil
}

func (uc *CFOPUseCase) ListByDirection(ctx context.Context, direction string, onlyActive bool) ([]*response.CFOPResponse, error) {
	list, err := uc.Repo.ListCFOPsByDirection(ctx, direction, onlyActive)
	if err != nil {
		return nil, err
	}
	return toCFOPResponses(list), nil
}

// normalizaCFOP aplica o padrão aos campos de lista vazios e recusa valor fora
// da lista dizendo o campo e o que é aceito.
func normalizaCFOP(c *entity.CFOP) error {
	if c.Code <= 0 {
		return errorsuc.NewValidationError("informe o código do CFOP")
	}
	if c.OrigemClasIPI != nil && *c.OrigemClasIPI == "" {
		c.OrigemClasIPI = nil
	}
	return nil
}
