package shipping_carrier_uc

import (
	"context"

	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/repository"
)

// Cadastros de apoio da transportadora: tipos de veículo e tipos de ocorrência.

func (uc *UseCase) ListarTiposVeiculo(ctx context.Context) ([]response.TipoVeiculoResponse, error) {
	lista, err := uc.Repo.ListarTiposVeiculo(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]response.TipoVeiculoResponse, 0, len(lista))
	for _, t := range lista {
		out = append(out, response.TipoVeiculoResponse{
			ID: t.ID, Code: t.Code, Description: t.Description, WheelType: t.WheelType, BodyType: t.BodyType,
			Axles: t.Axles, CapacityKg: t.CapacityKg, CapacityM3: t.CapacityM3, IsActive: t.IsActive, EmUso: t.EmUso,
		})
	}
	return out, nil
}

func (uc *UseCase) SalvarTipoVeiculo(ctx context.Context, id int64, dto *request.SalvarTipoVeiculoDTO) (*response.TipoVeiculoResponse, error) {
	if dto == nil {
		return nil, errorsuc.NewValidationError("informe o tipo de veículo")
	}
	t := &domrepo.TipoVeiculo{
		ID: id, Code: strings.ToUpper(strings.TrimSpace(dto.Code)), Description: strings.TrimSpace(dto.Description),
		WheelType: vazioNulo(dto.WheelType), BodyType: vazioNulo(dto.BodyType), Axles: dto.Axles,
		CapacityKg: num(dto.CapacityKg), CapacityM3: num(dto.CapacityM3), IsActive: boolOu(dto.IsActive, true),
	}
	switch {
	case id == 0 && (t.Code == "" || len(t.Code) > 20):
		return nil, errorsuc.NewValidationError("informe o código do tipo de veículo (até 20 caracteres)")
	case t.Description == "":
		return nil, errorsuc.NewValidationError("informe a descrição do tipo de veículo")
	case t.WheelType != nil && !entity.RodadoValido(*t.WheelType):
		return nil, errorsuc.NewValidationError("tipo de rodado inválido: use os códigos do MDF-e (01 a 06)")
	case t.BodyType != nil && !entity.CarroceriaValida(*t.BodyType):
		return nil, errorsuc.NewValidationError("tipo de carroceria inválido: use os códigos do MDF-e (00 a 05)")
	case t.Axles != nil && (*t.Axles < 2 || *t.Axles > 12):
		return nil, errorsuc.NewValidationError("o número de eixos deve estar entre 2 e 12")
	case t.CapacityKg.IsNegative() || t.CapacityM3.IsNegative():
		return nil, errorsuc.NewValidationError("a capacidade não pode ser negativa")
	}
	salvo, err := uc.Repo.SalvarTipoVeiculo(ctx, t)
	if err != nil {
		return nil, err
	}
	return &response.TipoVeiculoResponse{
		ID: salvo.ID, Code: salvo.Code, Description: salvo.Description, WheelType: salvo.WheelType,
		BodyType: salvo.BodyType, Axles: salvo.Axles, CapacityKg: salvo.CapacityKg, CapacityM3: salvo.CapacityM3,
		IsActive: salvo.IsActive,
	}, nil
}

func (uc *UseCase) ExcluirTipoVeiculo(ctx context.Context, id int64) error {
	return uc.Repo.ExcluirTipoVeiculo(ctx, id)
}

func (uc *UseCase) ListarTiposOcorrencia(ctx context.Context) ([]response.TipoOcorrenciaResponse, error) {
	lista, err := uc.Repo.ListarTiposOcorrencia(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]response.TipoOcorrenciaResponse, 0, len(lista))
	for _, t := range lista {
		out = append(out, response.TipoOcorrenciaResponse{
			ID: t.ID, Code: t.Code, Description: t.Description, CountsAsFailure: t.CountsAsFailure,
			IsActive: t.IsActive, EmUso: t.EmUso,
		})
	}
	return out, nil
}

func (uc *UseCase) SalvarTipoOcorrencia(ctx context.Context, id int64, dto *request.SalvarTipoOcorrenciaDTO) (*response.TipoOcorrenciaResponse, error) {
	if dto == nil {
		return nil, errorsuc.NewValidationError("informe o tipo de ocorrência")
	}
	codigo := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(dto.Code), " ", "_"))
	t := &domrepo.TipoOcorrencia{
		ID: id, Code: codigo, Description: strings.TrimSpace(dto.Description),
		CountsAsFailure: boolOu(dto.CountsAsFailure, true), IsActive: boolOu(dto.IsActive, true),
	}
	if id == 0 && (t.Code == "" || len(t.Code) > 40) {
		return nil, errorsuc.NewValidationError("informe o código do tipo de ocorrência (até 40 caracteres)")
	}
	if t.Description == "" {
		return nil, errorsuc.NewValidationError("informe a descrição do tipo de ocorrência")
	}
	salvo, err := uc.Repo.SalvarTipoOcorrencia(ctx, t)
	if err != nil {
		return nil, err
	}
	return &response.TipoOcorrenciaResponse{
		ID: salvo.ID, Code: salvo.Code, Description: salvo.Description,
		CountsAsFailure: salvo.CountsAsFailure, IsActive: salvo.IsActive,
	}, nil
}

func (uc *UseCase) ExcluirTipoOcorrencia(ctx context.Context, id int64) error {
	return uc.Repo.ExcluirTipoOcorrencia(ctx, id)
}

func vazioNulo(v *string) *string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	s := strings.TrimSpace(*v)
	return &s
}
