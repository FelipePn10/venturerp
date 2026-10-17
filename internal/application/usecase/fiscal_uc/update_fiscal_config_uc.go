package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

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
	if msg := conferirCodigoIBGE(dto.CodigoMunicipio, dto.UFEmpresa); msg != "" {
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

	updated, err := uc.Repo.UpdateFiscalConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return toFiscalConfigResponse(updated), nil
}

// prefixoIBGEDaUF: os dois primeiros dígitos do código IBGE do município são o
// código da UF. Código de outra UF passa na NF-e (a Focus resolve pelo nome) mas
// sai errado no CT-e, na NFS-e e no SPED.
var prefixoIBGEDaUF = map[string]string{
	"RO": "11", "AC": "12", "AM": "13", "RR": "14", "PA": "15", "AP": "16", "TO": "17",
	"MA": "21", "PI": "22", "CE": "23", "RN": "24", "PB": "25", "PE": "26", "AL": "27", "SE": "28", "BA": "29",
	"MG": "31", "ES": "32", "RJ": "33", "SP": "35", "PR": "41", "SC": "42", "RS": "43",
	"MS": "50", "MT": "51", "GO": "52", "DF": "53",
}

// conferirCodigoIBGE: vazio é aceito (a prévia avisa); preenchido tem de ter 7
// dígitos e ser da UF da empresa.
func conferirCodigoIBGE(codigo, uf string) string {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" || codigo == "0" || codigo == "0000000" {
		return ""
	}
	if len(codigo) != 7 || strings.Trim(codigo, "0123456789") != "" {
		return fmt.Sprintf("código IBGE do município inválido (%q): são 7 dígitos, ex.: 4106902 para Curitiba", codigo)
	}
	if p, ok := prefixoIBGEDaUF[strings.ToUpper(strings.TrimSpace(uf))]; ok && !strings.HasPrefix(codigo, p) {
		return fmt.Sprintf("o código IBGE %s não é de município do %s (os do %s começam com %s)", codigo, strings.ToUpper(uf), strings.ToUpper(uf), p)
	}
	return ""
}
