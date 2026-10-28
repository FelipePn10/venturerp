package enterprise_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/repository"
)

// IdentificacaoUseCase: a identificação da empresa da sessão (CNPJ, razão
// social, IE, regime, endereço) num lugar só — a VEMP0100. A configuração
// fiscal (VFIS0100) lê, mas não altera.
type IdentificacaoUseCase struct {
	Repo repository.EnterpriseRepository
	Auth ports.AuthService
}

func (uc *IdentificacaoUseCase) Obter(ctx context.Context) (*response.IdentificacaoEmpresaResponse, error) {
	d, err := uc.Repo.Identificacao(ctx)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errorsuc.NewNotFoundError("a empresa ainda não tem identificação: cadastre na VEMP0100")
	}
	return toIdentificacaoResponse(d), nil
}

func (uc *IdentificacaoUseCase) Atualizar(ctx context.Context, dto request.IdentificacaoEmpresaDTO) (*response.IdentificacaoEmpresaResponse, error) {
	if !uc.Auth.CanCreateEnterprise(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	por, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	d := &entity.DadosFiscais{
		CNPJ: dto.CNPJ, RazaoSocial: dto.RazaoSocial, NomeFantasia: dto.NomeFantasia, IE: dto.InscricaoEstadual,
		RegimeTributario: dto.RegimeTributario, UF: dto.UF, Municipio: dto.Municipio, CodigoMunicipio: dto.CodigoMunicipio,
		CEP: dto.CEP, Logradouro: dto.Logradouro, Numero: dto.Numero, Complemento: dto.Complemento, Bairro: dto.Bairro,
		Telefone: dto.Telefone, Email: dto.Email,
	}
	if err := d.Conferir(); err != nil {
		return nil, err
	}
	if err := uc.Repo.AtualizarIdentificacao(ctx, d, por); err != nil {
		return nil, err
	}
	return uc.Obter(ctx)
}

func toIdentificacaoResponse(d *entity.DadosFiscais) *response.IdentificacaoEmpresaResponse {
	return &response.IdentificacaoEmpresaResponse{
		CNPJ: d.CNPJ, RazaoSocial: d.RazaoSocial, NomeFantasia: d.NomeFantasia, InscricaoEstadual: d.IE,
		RegimeTributario: d.RegimeTributario, UF: d.UF, Municipio: d.Municipio, CodigoMunicipio: d.CodigoMunicipio,
		CEP: d.CEP, Logradouro: d.Logradouro, Numero: d.Numero, Complemento: d.Complemento, Bairro: d.Bairro,
		Telefone: d.Telefone, Email: d.Email,
	}
}
