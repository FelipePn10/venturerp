package fiscal_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	enterpriseentity "github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type UpdateFiscalConfigUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *UpdateFiscalConfigUseCase) Execute(ctx context.Context, dto request.UpdateFiscalConfigDTO) (*response.FiscalConfigResponse, error) {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	atual, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil && !errorsuc.EhNaoEncontrado(err) {
		return nil, err
	}
	if err := entity.ConferirTrocaDoFocus(atual, dto.FocusNfeAmbiente, dto.FocusNfeToken); err != nil {
		return nil, err
	}
	// A identificação da empresa (CNPJ, razão social, IE, regime, endereço) é
	// mantida na VEMP0100. A VFIS0100 manda só os parâmetros fiscais: sem
	// CNPJ e razão social no corpo, vale o que está gravado. A tela antiga, que
	// ainda manda tudo, continua aceita.
	semIdentificacao := strings.TrimSpace(dto.CnpjEmpresa) == "" && strings.TrimSpace(dto.RazaoSocial) == ""
	if semIdentificacao {
		if atual == nil {
			return nil, errorsuc.NewValidationError("a empresa ainda não tem identificação: cadastre CNPJ, razão social e endereço na VEMP0100")
		}
		dto.CnpjEmpresa, dto.RazaoSocial, dto.IEEmpresa = atual.CnpjEmpresa, atual.RazaoSocial, atual.IEEmpresa
		dto.RegimeTributario, dto.UFEmpresa = atual.RegimeTributario, atual.UFEmpresa
		dto.Logradouro, dto.Numero, dto.Complemento, dto.Bairro = atual.Logradouro, atual.Numero, atual.Complemento, atual.Bairro
		dto.Municipio, dto.CodigoMunicipio, dto.CEP, dto.Telefone = atual.Municipio, atual.CodigoMunicipio, atual.CEP, atual.Telefone
	} else if msg := conferirCodigoIBGE(dto.CodigoMunicipio, dto.UFEmpresa); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}

	cfg := &entity.FiscalConfig{
		CnpjEmpresa:               dto.CnpjEmpresa,
		RazaoSocial:               dto.RazaoSocial,
		IEEmpresa:                 dto.IEEmpresa,
		RegimeTributario:          dto.RegimeTributario,
		UFEmpresa:                 dto.UFEmpresa,
		IcmsInternoAliquota:       dto.IcmsInternoAliquota,
		IcmsDiferimentoPercentual: dto.IcmsDiferimentoPercentual,
		FocusNfeToken:             dto.FocusNfeToken,
		FocusNfeAmbiente:          dto.FocusNfeAmbiente,
		JurosMes:                  dto.JurosMes,
		MultaAtraso:               dto.MultaAtraso,
		VencimentoIcmsDia:         dto.VencimentoIcmsDia,
		VencimentoIPIDia:          dto.VencimentoIPIDia,
		VencimentoPisCofinsDia:    dto.VencimentoPisCofinsDia,
		Logradouro:                dto.Logradouro,
		Numero:                    dto.Numero,
		Complemento:               dto.Complemento,
		Bairro:                    dto.Bairro,
		Municipio:                 dto.Municipio,
		CodigoMunicipio:           dto.CodigoMunicipio,
		CEP:                       dto.CEP,
		Telefone:                  dto.Telefone,
		UpdatedBy:                 userID,
	}
	if atual != nil {
		// Nome fantasia e e-mail não vêm nesta rota: antes cada gravação os apagava.
		cfg.TradeName, cfg.Email = atual.TradeName, atual.Email
	}

	if dto.AliquotaCreditoSimples != nil && (*dto.AliquotaCreditoSimples < 0 || *dto.AliquotaCreditoSimples > 100) {
		return nil, errorsuc.NewValidationError("a alíquota de crédito do Simples Nacional deve estar entre 0 e 100%")
	}
	if atual != nil && (dto.ExigirRegraICMSIPI == nil || dto.AliquotaCreditoSimples == nil || dto.CBenefDiferimento == nil) {
		cfg.ExigirRegraICMSIPI, cfg.AliquotaCreditoSimples = atual.ExigirRegraICMSIPI, atual.AliquotaCreditoSimples
		cfg.CBenefDiferimento = atual.CBenefDiferimento
	}
	if dto.ExigirRegraICMSIPI != nil {
		cfg.ExigirRegraICMSIPI = *dto.ExigirRegraICMSIPI
	}
	if dto.AliquotaCreditoSimples != nil {
		cfg.AliquotaCreditoSimples = *dto.AliquotaCreditoSimples
	}
	if dto.CBenefDiferimento != nil {
		v := strings.ToUpper(strings.TrimSpace(*dto.CBenefDiferimento))
		if v != "" && (len(v) < 8 || len(v) > 10) {
			return nil, errorsuc.NewValidationError("o código de benefício fiscal (cBenef) tem 8 a 10 caracteres, como PR830001")
		}
		cfg.CBenefDiferimento = &v
	}

	updated, err := uc.Repo.UpdateFiscalConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return toFiscalConfigResponse(updated), nil
}

// conferirCodigoIBGE: a regra é a da identificação da empresa (domínio).
func conferirCodigoIBGE(codigo, uf string) string {
	return enterpriseentity.ErroDoCodigoIBGE(codigo, uf)
}
