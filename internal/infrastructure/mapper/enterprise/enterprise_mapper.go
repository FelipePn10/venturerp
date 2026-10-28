package mapper

import (
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity"
)

func ToEnterpriseEntity(d request.CreateEnterpriseDTO) (*entity.Enterprise, error) {
	nome := strings.TrimSpace(d.Name)
	if nome == "" {
		nome = strings.TrimSpace(d.NomeFantasia)
	}
	if nome == "" {
		nome = strings.TrimSpace(d.RazaoSocial)
	}
	e, err := entity.NewEnterprise(d.Code, nome, d.CreatedBy)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(d.CNPJ) != "" {
		f := &entity.DadosFiscais{CNPJ: d.CNPJ, RazaoSocial: d.RazaoSocial, NomeFantasia: d.NomeFantasia, IE: d.InscricaoEstadual,
			RegimeTributario: d.RegimeTributario, UF: d.UF, Municipio: d.Municipio, CodigoMunicipio: d.CodigoMunicipio, CEP: d.CEP,
			Logradouro: d.Logradouro, Numero: d.Numero, Complemento: d.Complemento, Bairro: d.Bairro, Telefone: d.Telefone, Email: d.Email}
		if err := f.Conferir(); err != nil {
			return nil, err
		}
		e.Fiscal = f
	}
	if e.Name == "" {
		return nil, entity.ErrInvalidName
	}
	return e, nil
}
