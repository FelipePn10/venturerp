package cnpj

import (
	"context"
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
)

// cnpjwsProvider adapts the CNPJ.ws public API (publica.cnpj.ws/cnpj/{cnpj}).
// Unlike the Receita-only endpoints, it also exposes the Inscrições Estaduais
// (`inscricoes_estaduais`, sourced from SINTEGRA) which the cadastro screens
// need to pre-fill the Inscrição Estadual field.
type cnpjwsProvider struct {
	base string
	http *http.Client
}

type cnpjwsResponse struct {
	RazaoSocial string `json:"razao_social"`
	Porte       struct {
		Descricao string `json:"descricao"`
	} `json:"porte"`
	NaturezaJuridica struct {
		Descricao string `json:"descricao"`
	} `json:"natureza_juridica"`
	Simples struct {
		Simples string `json:"simples"`
		MEI     string `json:"mei"`
	} `json:"simples"`
	Estabelecimento struct {
		CNPJ                string `json:"cnpj"`
		NomeFantasia        string `json:"nome_fantasia"`
		SituacaoCadastral   string `json:"situacao_cadastral"`
		DataInicioAtividade string `json:"data_inicio_atividade"`
		TipoLogradouro      string `json:"tipo_logradouro"`
		Logradouro          string `json:"logradouro"`
		Numero              string `json:"numero"`
		Complemento         string `json:"complemento"`
		Bairro              string `json:"bairro"`
		CEP                 string `json:"cep"`
		DDD1                string `json:"ddd1"`
		Telefone1           string `json:"telefone1"`
		Email               string `json:"email"`
		Cidade              struct {
			Nome string `json:"nome"`
		} `json:"cidade"`
		Estado struct {
			Sigla string `json:"sigla"`
		} `json:"estado"`
		AtividadePrincipal struct {
			ID        string `json:"id"`
			Descricao string `json:"descricao"`
		} `json:"atividade_principal"`
		AtividadesSecundarias []struct {
			ID        string `json:"id"`
			Descricao string `json:"descricao"`
		} `json:"atividades_secundarias"`
		InscricoesEstaduais []struct {
			InscricaoEstadual string `json:"inscricao_estadual"`
			Ativo             bool   `json:"ativo"`
			Estado            struct {
				Sigla string `json:"sigla"`
			} `json:"estado"`
		} `json:"inscricoes_estaduais"`
	} `json:"estabelecimento"`
}

func (p *cnpjwsProvider) Lookup(ctx context.Context, cnpj string) (*entity.Company, error) {
	digits := onlyDigits(cnpj)
	var r cnpjwsResponse
	if err := doGET(ctx, p.http, p.base+"/cnpj/"+digits, &r); err != nil {
		return nil, err
	}
	if r.Estabelecimento.CNPJ == "" && r.RazaoSocial == "" {
		return nil, service.ErrNotFound
	}

	e := r.Estabelecimento
	street := strings.TrimSpace(e.Logradouro)
	if tipo := strings.TrimSpace(e.TipoLogradouro); tipo != "" {
		street = tipo + " " + street
	}

	c := &entity.Company{
		CNPJ:               digits,
		LegalName:          strings.TrimSpace(r.RazaoSocial),
		TradeName:          strings.TrimSpace(e.NomeFantasia),
		RegistrationStatus: strings.ToUpper(strings.TrimSpace(e.SituacaoCadastral)),
		LegalNature:        strings.TrimSpace(r.NaturezaJuridica.Descricao),
		Size:               strings.TrimSpace(r.Porte.Descricao),
		OpeningDate:        e.DataInicioAtividade,
		SimplesOptant:      strings.EqualFold(strings.TrimSpace(r.Simples.Simples), "sim"),
		MEI:                strings.EqualFold(strings.TrimSpace(r.Simples.MEI), "sim"),
		Source:             "cnpj.ws",
		Address: entity.Address{
			ZipCode:      onlyDigits(e.CEP),
			Street:       street,
			Number:       strings.TrimSpace(e.Numero),
			Complement:   strings.TrimSpace(e.Complemento),
			Neighborhood: strings.TrimSpace(e.Bairro),
			City:         strings.TrimSpace(e.Cidade.Nome),
			UF:           strings.ToUpper(strings.TrimSpace(e.Estado.Sigla)),
		},
	}
	if e.Email != "" {
		c.Email = strings.TrimSpace(e.Email)
	}
	if e.Telefone1 != "" {
		c.Phone = strings.TrimSpace(e.DDD1 + e.Telefone1)
	}
	if e.AtividadePrincipal.ID != "" {
		c.MainActivity = entity.Activity{
			Code:        e.AtividadePrincipal.ID,
			Description: strings.TrimSpace(e.AtividadePrincipal.Descricao),
		}
	}
	for _, a := range e.AtividadesSecundarias {
		c.SecondaryActivities = append(c.SecondaryActivities, entity.Activity{
			Code:        a.ID,
			Description: strings.TrimSpace(a.Descricao),
		})
	}
	for _, ie := range e.InscricoesEstaduais {
		c.StateRegistrations = append(c.StateRegistrations, entity.StateRegistration{
			UF:      strings.ToUpper(strings.TrimSpace(ie.Estado.Sigla)),
			Number:  strings.TrimSpace(ie.InscricaoEstadual),
			Enabled: ie.Ativo,
		})
	}
	return c, nil
}
