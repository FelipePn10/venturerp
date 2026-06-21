package cnpj

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
)

// brasilAPIProvider adapts brasilapi.com.br/api/cnpj/v1/{cnpj}.
type brasilAPIProvider struct {
	base string
	http *http.Client
}

// brasilAPIResponse mirrors the subset of fields we consume.
type brasilAPIResponse struct {
	CNPJ                       string `json:"cnpj"`
	RazaoSocial                string `json:"razao_social"`
	NomeFantasia               string `json:"nome_fantasia"`
	DescricaoSituacaoCadastral string `json:"descricao_situacao_cadastral"`
	NaturezaJuridica           string `json:"natureza_juridica"`
	Porte                      string `json:"porte"`
	DataInicioAtividade        string `json:"data_inicio_atividade"`
	CEP                        string `json:"cep"`
	Logradouro                 string `json:"logradouro"`
	Numero                     string `json:"numero"`
	Complemento                string `json:"complemento"`
	Bairro                     string `json:"bairro"`
	Municipio                  string `json:"municipio"`
	UF                         string `json:"uf"`
	DDDTelefone1               string `json:"ddd_telefone_1"`
	Email                      string `json:"email"`
	CNAEFiscal                 int64  `json:"cnae_fiscal"`
	CNAEFiscalDescricao        string `json:"cnae_fiscal_descricao"`
	OpcaoPeloSimples           *bool  `json:"opcao_pelo_simples"`
	OpcaoPeloMEI               *bool  `json:"opcao_pelo_mei"`
	CNAEsSecundarios           []struct {
		Codigo    int64  `json:"codigo"`
		Descricao string `json:"descricao"`
	} `json:"cnaes_secundarios"`
}

func (p *brasilAPIProvider) Lookup(ctx context.Context, cnpj string) (*entity.Company, error) {
	digits := onlyDigits(cnpj)
	var r brasilAPIResponse
	if err := doGET(ctx, p.http, p.base+"/"+digits, &r); err != nil {
		return nil, err
	}
	if r.CNPJ == "" && r.RazaoSocial == "" {
		return nil, service.ErrNotFound
	}

	c := &entity.Company{
		CNPJ:               digits,
		LegalName:          strings.TrimSpace(r.RazaoSocial),
		TradeName:          strings.TrimSpace(r.NomeFantasia),
		RegistrationStatus: strings.ToUpper(strings.TrimSpace(r.DescricaoSituacaoCadastral)),
		LegalNature:        strings.TrimSpace(r.NaturezaJuridica),
		Size:               strings.TrimSpace(r.Porte),
		OpeningDate:        r.DataInicioAtividade,
		Email:              strings.TrimSpace(r.Email),
		Phone:              strings.TrimSpace(r.DDDTelefone1),
		Source:             "brasilapi",
		Address: entity.Address{
			ZipCode:      onlyDigits(r.CEP),
			Street:       strings.TrimSpace(r.Logradouro),
			Number:       strings.TrimSpace(r.Numero),
			Complement:   strings.TrimSpace(r.Complemento),
			Neighborhood: strings.TrimSpace(r.Bairro),
			City:         strings.TrimSpace(r.Municipio),
			UF:           strings.ToUpper(strings.TrimSpace(r.UF)),
		},
	}
	if r.OpcaoPeloSimples != nil {
		c.SimplesOptant = *r.OpcaoPeloSimples
	}
	if r.OpcaoPeloMEI != nil {
		c.MEI = *r.OpcaoPeloMEI
	}
	if r.CNAEFiscal != 0 {
		c.MainActivity = entity.Activity{
			Code:        formatCNAE(r.CNAEFiscal),
			Description: strings.TrimSpace(r.CNAEFiscalDescricao),
		}
	}
	for _, s := range r.CNAEsSecundarios {
		c.SecondaryActivities = append(c.SecondaryActivities, entity.Activity{
			Code:        formatCNAE(s.Codigo),
			Description: strings.TrimSpace(s.Descricao),
		})
	}
	return c, nil
}

// formatCNAE renders a numeric CNAE as a zero-padded 7-digit string.
func formatCNAE(code int64) string {
	s := strconv.FormatInt(code, 10)
	for len(s) < 7 {
		s = "0" + s
	}
	return s
}
