package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
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

	updated, err := uc.Repo.UpdateFiscalConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return toFiscalConfigResponse(updated), nil
}
