package nfse_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
)

func toNFSeResponse(n *entity.NFSe) *response.NFSeResponse {
	if n == nil {
		return nil
	}
	return &response.NFSeResponse{
		ID:                        n.ID,
		NumeroRPS:                 n.NumeroRPS,
		SerieRPS:                  n.SerieRPS,
		TipoRPS:                   n.TipoRPS,
		DataEmissao:               n.DataEmissao,
		Status:                    string(n.Status),
		NaturezaOperacao:          n.NaturezaOperacao,
		OptanteSimples:            n.OptanteSimples,
		IncentivadorCultural:      n.IncentivadorCultural,
		TomadorCnpjCpf:            n.TomadorCnpjCpf,
		TomadorRazaoSocial:        n.TomadorRazaoSocial,
		TomadorEmail:              n.TomadorEmail,
		TomadorLogradouro:         n.TomadorLogradouro,
		TomadorNumero:             n.TomadorNumero,
		TomadorComplemento:        n.TomadorComplemento,
		TomadorBairro:             n.TomadorBairro,
		TomadorCodigoMunicipio:    n.TomadorCodigoMunicipio,
		TomadorUF:                 n.TomadorUF,
		TomadorCEP:                n.TomadorCEP,
		ItemListaServico:          n.ItemListaServico,
		CodigoTributarioMunicipio: n.CodigoTributarioMunicipio,
		Discriminacao:             n.Discriminacao,
		CodigoMunicipio:           n.CodigoMunicipio,
		ValorServicos:             n.ValorServicos.InexactFloat64(),
		ValorDeducoes:             n.ValorDeducoes.InexactFloat64(),
		AliquotaISS:               n.AliquotaISS.InexactFloat64(),
		IssRetido:                 n.IssRetido,
		ValorISS:                  n.ValorISS.InexactFloat64(),
		ValorLiquido:              n.ValorLiquido.InexactFloat64(),
		FocusRef:                  n.FocusRef,
		NumeroNFSe:                n.NumeroNFSe,
		CodigoVerificacao:         n.CodigoVerificacao,
		URL:                       n.URL,
		XMLPath:                   n.XMLPath,
		SalesOrderCode:            n.SalesOrderCode,
		CustomerCode:              n.CustomerCode,
		DataVencimento:            n.DataVencimento,
		Notes:                     n.Notes,
		IsActive:                  n.IsActive,
		CreatedBy:                 n.CreatedBy,
		CreatedAt:                 n.CreatedAt,
		UpdatedAt:                 n.UpdatedAt,
	}
}

func toNFSeResponses(list []*entity.NFSe) []*response.NFSeResponse {
	out := make([]*response.NFSeResponse, 0, len(list))
	for _, n := range list {
		out = append(out, toNFSeResponse(n))
	}
	return out
}
