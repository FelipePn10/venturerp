package fiscal_params_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type CFOPUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *CFOPUseCase) Create(ctx context.Context, dto request.CreateCFOPDTO) (*response.CFOPResponse, error) {
	if dto.Description == "" {
		return nil, errors.New("description is required")
	}
	c := &entity.CFOP{
		Code:            dto.Code,
		Description:     dto.Description,
		DescriptionFull: dto.DescriptionFull,
		Utilization:     entity.CfopUtilization(dto.Utilization),
		OrigemClasIPI:   dto.OrigemClasIPI,
		IndOperacao:     entity.CfopIndOperacao(dto.IndOperacao),
		TipoUtilizacao:  entity.CfopTipoUtilizacao(dto.TipoUtilizacao),
		CodigoAnexoSN:   dto.CodigoAnexoSN,
		DIFAL:           dto.DIFAL,
		Doacao:          dto.Doacao,
		IsActive:        true,
	}
	if c.Utilization == "" {
		c.Utilization = entity.CfopUtilizationIndustrializacaoComercio
	}
	if c.IndOperacao == "" {
		c.IndOperacao = entity.CfopIndOperacaoNormal
	}
	if c.TipoUtilizacao == "" {
		c.TipoUtilizacao = entity.CfopTipoUtilizacaoNormal
	}
	created, err := uc.Repo.CreateCFOP(ctx, c)
	if err != nil {
		return nil, err
	}
	return toCFOPResponse(created), nil
}

func (uc *CFOPUseCase) Update(ctx context.Context, dto request.UpdateCFOPDTO) (*response.CFOPResponse, error) {
	c := &entity.CFOP{
		ID:              dto.ID,
		Code:            dto.Code,
		Description:     dto.Description,
		DescriptionFull: dto.DescriptionFull,
		Utilization:     entity.CfopUtilization(dto.Utilization),
		OrigemClasIPI:   dto.OrigemClasIPI,
		IndOperacao:     entity.CfopIndOperacao(dto.IndOperacao),
		TipoUtilizacao:  entity.CfopTipoUtilizacao(dto.TipoUtilizacao),
		CodigoAnexoSN:   dto.CodigoAnexoSN,
		DIFAL:           dto.DIFAL,
		Doacao:          dto.Doacao,
		IsActive:        dto.IsActive,
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
