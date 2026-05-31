package fiscal_params_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type CFOPUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *CFOPUseCase) Create(ctx context.Context, dto request.CreateCFOPDTO) (*entity.CFOP, error) {
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
	return uc.Repo.CreateCFOP(ctx, c)
}

func (uc *CFOPUseCase) Update(ctx context.Context, dto request.UpdateCFOPDTO) (*entity.CFOP, error) {
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
	return uc.Repo.UpdateCFOP(ctx, c)
}

func (uc *CFOPUseCase) GetByCode(ctx context.Context, code int32) (*entity.CFOP, error) {
	return uc.Repo.GetCFOPByCode(ctx, code)
}

func (uc *CFOPUseCase) List(ctx context.Context, onlyActive bool) ([]*entity.CFOP, error) {
	return uc.Repo.ListCFOPs(ctx, onlyActive)
}

func (uc *CFOPUseCase) ListByDirection(ctx context.Context, direction string, onlyActive bool) ([]*entity.CFOP, error) {
	return uc.Repo.ListCFOPsByDirection(ctx, direction, onlyActive)
}
