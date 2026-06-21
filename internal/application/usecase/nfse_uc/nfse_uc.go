// Package nfse_uc holds the NFS-e (service invoice) use cases: create a draft,
// authorize it at the city hall via Focus, consult, cancel and list.
package nfse_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/nfse/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
	"github.com/shopspring/decimal"
)

// ConfigProvider supplies the fiscal config (Focus token + prestador data).
// The fiscal repository satisfies it.
type ConfigProvider interface {
	GetFiscalConfig(ctx context.Context) (*fiscalentity.FiscalConfig, error)
}

// CreateNFSeUseCase creates an NFS-e draft and computes ISS / net value.
type CreateNFSeUseCase struct {
	Repo repository.NFSeRepository
	Auth ports.AuthService
}

func (uc *CreateNFSeUseCase) Execute(ctx context.Context, dto request.CreateNFSeDTO) (*response.NFSeResponse, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.ValorServicos <= 0 {
		return nil, fmt.Errorf("valor_servicos deve ser maior que zero")
	}
	if dto.ItemListaServico == "" {
		return nil, fmt.Errorf("item_lista_servico é obrigatório")
	}
	if dto.CodigoMunicipio == "" {
		return nil, fmt.Errorf("codigo_municipio (prestação do serviço) é obrigatório")
	}
	if dto.Discriminacao == "" {
		return nil, fmt.Errorf("discriminacao é obrigatória")
	}

	dataEmissao, err := time.Parse("2006-01-02", dto.DataEmissao)
	if err != nil {
		return nil, fmt.Errorf("data_emissao inválida (use AAAA-MM-DD): %w", err)
	}

	// Base de cálculo do ISS = valor dos serviços − deduções; ISS = base × alíquota.
	base := decimal.NewFromFloat(dto.ValorServicos).Sub(decimal.NewFromFloat(dto.ValorDeducoes))
	valorISS := base.Mul(decimal.NewFromFloat(dto.AliquotaISS)).Round(2)
	// Valor líquido: se ISS retido, é descontado do valor dos serviços.
	liquido := decimal.NewFromFloat(dto.ValorServicos)
	if dto.IssRetido {
		liquido = liquido.Sub(valorISS)
	}

	tipoRPS := dto.TipoRPS
	if tipoRPS == 0 {
		tipoRPS = 1
	}
	natOp := dto.NaturezaOperacao
	if natOp == 0 {
		natOp = 1
	}

	valorISSf, _ := valorISS.Float64()
	liquidof, _ := liquido.Float64()

	n := &entity.NFSe{
		NumeroRPS:                 dto.NumeroRPS,
		SerieRPS:                  dto.SerieRPS,
		TipoRPS:                   tipoRPS,
		DataEmissao:               dataEmissao,
		Status:                    entity.NFSeStatusRascunho,
		NaturezaOperacao:          natOp,
		OptanteSimples:            dto.OptanteSimples,
		IncentivadorCultural:      dto.IncentivadorCultural,
		TomadorCnpjCpf:            dto.TomadorCnpjCpf,
		TomadorRazaoSocial:        dto.TomadorRazaoSocial,
		TomadorEmail:              dto.TomadorEmail,
		TomadorLogradouro:         dto.TomadorLogradouro,
		TomadorNumero:             dto.TomadorNumero,
		TomadorComplemento:        dto.TomadorComplemento,
		TomadorBairro:             dto.TomadorBairro,
		TomadorCodigoMunicipio:    dto.TomadorCodigoMunicipio,
		TomadorUF:                 dto.TomadorUF,
		TomadorCEP:                dto.TomadorCEP,
		ItemListaServico:          dto.ItemListaServico,
		CodigoTributarioMunicipio: dto.CodigoTributarioMunicipio,
		Discriminacao:             dto.Discriminacao,
		CodigoMunicipio:           dto.CodigoMunicipio,
		ValorServicos:             dto.ValorServicos,
		ValorDeducoes:             dto.ValorDeducoes,
		AliquotaISS:               dto.AliquotaISS,
		IssRetido:                 dto.IssRetido,
		ValorISS:                  valorISSf,
		ValorLiquido:              liquidof,
		SalesOrderCode:            dto.SalesOrderCode,
		Notes:                     dto.Notes,
		CreatedBy:                 userID,
	}
	created, err := uc.Repo.Create(ctx, n)
	if err != nil {
		return nil, err
	}
	return toNFSeResponse(created), nil
}

// AuthorizeNFSeUseCase transmits a draft NFS-e to the city hall via Focus.
type AuthorizeNFSeUseCase struct {
	Repo   repository.NFSeRepository
	Config ConfigProvider
	Auth   ports.AuthService
}

func (uc *AuthorizeNFSeUseCase) Execute(ctx context.Context, id int64) (*response.NFSeResponse, error) {
	if !uc.Auth.CanAuthorizeFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	n, err := uc.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if n.Status != entity.NFSeStatusRascunho && n.Status != entity.NFSeStatusRejeitada {
		return nil, fmt.Errorf("NFS-e deve estar em rascunho para autorizar, status atual: %s", n.Status)
	}

	cfg, err := uc.Config.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, fmt.Errorf("token Focus NF-e não configurado — acesse Configurações Fiscais")
	}

	payload := focusnfe.NFSePayload{
		DataEmissao:            n.DataEmissao.Format("2006-01-02T15:04:05-03:00"),
		NaturezaOperacao:       n.NaturezaOperacao,
		OptanteSimplesNacional: n.OptanteSimples,
		IncentivadorCultural:   n.IncentivadorCultural,
		Prestador: focusnfe.NFSePrestador{
			CNPJ:            cfg.CnpjEmpresa,
			CodigoMunicipio: cfg.CodigoMunicipio,
		},
		Tomador: focusnfe.NFSeTomador{
			RazaoSocial:     deref(n.TomadorRazaoSocial),
			Email:           deref(n.TomadorEmail),
			Logradouro:      deref(n.TomadorLogradouro),
			Numero:          deref(n.TomadorNumero),
			Complemento:     deref(n.TomadorComplemento),
			Bairro:          deref(n.TomadorBairro),
			CodigoMunicipio: deref(n.TomadorCodigoMunicipio),
			UF:              deref(n.TomadorUF),
			CEP:             deref(n.TomadorCEP),
		},
		ItemListaServico:          n.ItemListaServico,
		CodigoTributarioMunicipio: deref(n.CodigoTributarioMunicipio),
		Discriminacao:             n.Discriminacao,
		CodigoMunicipio:           n.CodigoMunicipio,
		ValorServicos:             n.ValorServicos,
		ValorDeducoes:             n.ValorDeducoes,
		AliquotaISS:               n.AliquotaISS,
		IssRetido:                 n.IssRetido,
		ValorIss:                  n.ValorISS,
	}
	// Tomador document: CNPJ (14) or CPF (11).
	if n.TomadorCnpjCpf != nil {
		doc := *n.TomadorCnpjCpf
		if len(doc) > 11 {
			payload.Tomador.CNPJ = doc
		} else {
			payload.Tomador.CPF = doc
		}
	}

	cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	cli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		_ = uc.Repo.SaveFocusLog(ctx, endpoint, method, reqBody, respBody, statusCode, durationMs)
	})
	ref := fmt.Sprintf("nfse%d%d", id, time.Now().UnixNano()%1000000)
	if len(ref) > 50 {
		ref = ref[:50]
	}

	resp, err := cli.EmitirNFSe(ctx, ref, payload)
	if err != nil {
		_, _ = uc.Repo.UpdateStatus(ctx, id, entity.NFSeStatusRejeitada)
		return nil, fmt.Errorf("Focus NFS-e: %w", err)
	}
	authorized, err := uc.Repo.UpdateAuthorization(ctx, id, resp.NumeroNFSe, resp.CodigoVerificacao, resp.URL, ref)
	if err != nil {
		return nil, err
	}
	return toNFSeResponse(authorized), nil
}

// CancelNFSeUseCase cancels an authorized NFS-e at the city hall.
type CancelNFSeUseCase struct {
	Repo   repository.NFSeRepository
	Config ConfigProvider
	Auth   ports.AuthService
}

func (uc *CancelNFSeUseCase) Execute(ctx context.Context, id int64, dto request.CancelNFSeDTO) (*response.NFSeResponse, error) {
	if !uc.Auth.CanAuthorizeFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len(dto.Justificativa) < 15 {
		return nil, fmt.Errorf("justificativa deve ter no mínimo 15 caracteres")
	}
	n, err := uc.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if n.Status != entity.NFSeStatusAutorizada {
		return nil, fmt.Errorf("apenas NFS-e autorizada pode ser cancelada")
	}
	if n.FocusRef == nil || *n.FocusRef == "" {
		return nil, fmt.Errorf("NFS-e %d não possui referência Focus", id)
	}
	cfg, err := uc.Config.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, fmt.Errorf("token Focus NF-e não configurado")
	}
	cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	cli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		_ = uc.Repo.SaveFocusLog(ctx, endpoint, method, reqBody, respBody, statusCode, durationMs)
	})
	if _, err := cli.CancelarNFSe(ctx, *n.FocusRef, dto.Justificativa); err != nil {
		return nil, fmt.Errorf("Focus NFS-e cancelamento: %w", err)
	}
	cancelled, err := uc.Repo.UpdateStatus(ctx, id, entity.NFSeStatusCancelada)
	if err != nil {
		return nil, err
	}
	return toNFSeResponse(cancelled), nil
}

// ListNFSeUseCase lists service invoices.
type ListNFSeUseCase struct {
	Repo repository.NFSeRepository
	Auth ports.AuthService
}

func (uc *ListNFSeUseCase) Execute(ctx context.Context) ([]*response.NFSeResponse, error) {
	if !uc.Auth.CanGetFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toNFSeResponses(list), nil
}

// GetNFSeUseCase returns one NFS-e by ID.
type GetNFSeUseCase struct {
	Repo repository.NFSeRepository
	Auth ports.AuthService
}

func (uc *GetNFSeUseCase) Execute(ctx context.Context, id int64) (*response.NFSeResponse, error) {
	if !uc.Auth.CanGetFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	n, err := uc.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toNFSeResponse(n), nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
