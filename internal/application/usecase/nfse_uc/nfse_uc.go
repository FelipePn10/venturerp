// Package nfse_uc holds the NFS-e (service invoice) use cases: create a draft,
// authorize it at the city hall via Focus, consult, cancel and list.
package nfse_uc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	financialentity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
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

// LeitorDeCliente confere o cliente do título da NFS-e (o repositório de
// clientes, filtrado pela empresa da sessão, satisfaz).
type LeitorDeCliente interface {
	GetCustomerByCode(ctx context.Context, code int64) (*customerentity.Customer, error)
}

// TitulosDaNFSe: o título a receber que a NFS-e autorizada gera e que o
// cancelamento da nota cancela. O repositório financeiro satisfaz.
type TitulosDaNFSe interface {
	TitulosDaNFSe(ctx context.Context, nfseID int64) ([]*financialentity.ContaReceber, error)
	GerarTituloDaNFSe(ctx context.Context, nfseID int64, c *financialentity.ContaReceber) (bool, error)
	CancelarTitulosDaNFSe(ctx context.Context, nfseID int64) error
}

// CreateNFSeUseCase creates an NFS-e draft and computes ISS / net value.
type CreateNFSeUseCase struct {
	Repo repository.NFSeRepository
	Auth ports.AuthService
	// Clientes confere o cliente do título. Opcional: sem ele, o cliente
	// informado é recusado (não dá para garantir que é desta empresa).
	Clientes LeitorDeCliente
}

func (uc *CreateNFSeUseCase) Execute(ctx context.Context, dto request.CreateNFSeDTO) (*response.NFSeResponse, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if !dto.ValorServicos.IsPositive() {
		return nil, errorsuc.NewValidationError("valor_servicos deve ser maior que zero")
	}
	if dto.ItemListaServico == "" {
		return nil, errorsuc.NewValidationError("informe o item da lista de serviços")
	}
	if dto.CodigoMunicipio == "" {
		return nil, errorsuc.NewValidationError("informe o código do município onde o serviço foi prestado")
	}
	if dto.Discriminacao == "" {
		return nil, errorsuc.NewValidationError("discriminacao é obrigatória")
	}

	if err := conferirAliquotaEDeducoes(dto.AliquotaISS, dto.ValorServicos, dto.ValorDeducoes); err != nil {
		return nil, err
	}

	dataEmissao, err := time.Parse("2006-01-02", dto.DataEmissao)
	if err != nil {
		return nil, errorsuc.NewValidationError("data de emissão inválida: use AAAA-MM-DD")
	}

	vencimento, err := vencimentoDaNFSe(dto.DataVencimento, dataEmissao)
	if err != nil {
		return nil, err
	}
	if dto.CustomerCode != nil {
		if err := uc.conferirCliente(ctx, &dto); err != nil {
			return nil, err
		}
	}

	// Base de cálculo do ISS = valor dos serviços − deduções; ISS = base × alíquota.
	base := dto.ValorServicos.Sub(dto.ValorDeducoes)
	valorISS := base.Mul(dto.AliquotaISS).Round(2)
	// Valor líquido: se ISS retido, é descontado do valor dos serviços.
	liquido := dto.ValorServicos
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
		ValorISS:                  valorISS,
		ValorLiquido:              liquido,
		SalesOrderCode:            dto.SalesOrderCode,
		CustomerCode:              dto.CustomerCode,
		DataVencimento:            vencimento,
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
	// Titulos gera o título a receber da nota com cliente. Opcional.
	Titulos TitulosDaNFSe
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
		return nil, errorsuc.NewValidationError("só NFS-e em rascunho ou rejeitada pode ser autorizada")
	}

	cfg, err := uc.Config.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, errorsuc.NewValidationError("o token da Focus NF-e não está configurado: informe-o nos Parâmetros Fiscais (VFIS0100)")
	}

	payload := payloadDaNFSe(n, cfg)

	cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	cli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		if errLog := uc.Repo.SaveFocusLog(ctx, endpoint, method, reqBody, respBody, statusCode, durationMs); errLog != nil {
			// A chamada à Focus já aconteceu: não dá para desfazê-la, mas a falha
			// de registrar a comunicação não pode sumir.
			slog.Error("falha ao gravar o registro da comunicação com a Focus", "endpoint", endpoint, "erro", errLog)
		}
	})
	ref := fmt.Sprintf("nfse%d%d", id, time.Now().UnixNano()%1000000)
	if len(ref) > 50 {
		ref = ref[:50]
	}

	resp, err := cli.EmitirNFSe(ctx, ref, payload)
	if err != nil {
		if _, errSit := uc.Repo.UpdateStatus(ctx, id, entity.NFSeStatusRejeitada); errSit != nil {
			slog.Error("NFS-e recusada não pôde ser marcada como rejeitada", "documento", id, "erro", errSit)
		}
		return nil, fiscal_uc.ErroDaEmissora(ctx, "a prefeitura", "a NFS-e", id, err)
	}
	authorized, err := uc.Repo.UpdateAuthorization(ctx, id, resp.NumeroNFSe, resp.CodigoVerificacao, resp.URL, ref)
	if err != nil {
		return nil, err
	}
	// O título vem DEPOIS de gravar a autorização: se ele falhar, a nota já
	// está autorizada na prefeitura e não pode ser transmitida de novo.
	if uc.Titulos != nil && authorized.CustomerCode != nil {
		userID, _ := uc.Auth.UserID(ctx)
		if _, err := uc.Titulos.GerarTituloDaNFSe(ctx, id, tituloDaNFSe(authorized, userID, time.Now())); err != nil {
			slog.Error("NFS-e autorizada sem título a receber", "documento", id, "erro", err)
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"a NFS-e %s foi autorizada, mas o título a receber não foi gerado: lance-o no contas a receber (%v)",
				deref(authorized.NumeroNFSe), err))
		}
	}
	return toNFSeResponse(authorized), nil
}

// CancelNFSeUseCase cancels an authorized NFS-e at the city hall.
type CancelNFSeUseCase struct {
	Repo   repository.NFSeRepository
	Config ConfigProvider
	Auth   ports.AuthService
	// Titulos: título recebido impede o cancelamento; o em aberto é cancelado
	// junto com a nota. Opcional.
	Titulos TitulosDaNFSe
}

func (uc *CancelNFSeUseCase) Execute(ctx context.Context, id int64, dto request.CancelNFSeDTO) (*response.NFSeResponse, error) {
	if !uc.Auth.CanAuthorizeFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len(dto.Justificativa) < 15 {
		return nil, errorsuc.NewValidationError("justificativa deve ter no mínimo 15 caracteres")
	}
	n, err := uc.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if n.Status != entity.NFSeStatusAutorizada {
		return nil, errorsuc.NewValidationError("apenas NFS-e autorizada pode ser cancelada")
	}
	if n.FocusRef == nil || *n.FocusRef == "" {
		return nil, errorsuc.NewValidationError("esta NFS-e não foi transmitida à prefeitura por este sistema, então não há o que cancelar")
	}
	if uc.Titulos != nil {
		titulos, err := uc.Titulos.TitulosDaNFSe(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := recusarComTituloRecebido(titulos); err != nil {
			return nil, err
		}
	}
	cfg, err := uc.Config.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, errorsuc.NewValidationError("o token da Focus NF-e não está configurado: informe-o nos Parâmetros Fiscais (VFIS0100)")
	}
	cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	cli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		if errLog := uc.Repo.SaveFocusLog(ctx, endpoint, method, reqBody, respBody, statusCode, durationMs); errLog != nil {
			// A chamada à Focus já aconteceu: não dá para desfazê-la, mas a falha
			// de registrar a comunicação não pode sumir.
			slog.Error("falha ao gravar o registro da comunicação com a Focus", "endpoint", endpoint, "erro", errLog)
		}
	})
	if _, err := cli.CancelarNFSe(ctx, *n.FocusRef, dto.Justificativa); err != nil {
		return nil, fiscal_uc.ErroDaEmissora(ctx, "a prefeitura", "o cancelamento", id, err)
	}
	cancelled, err := uc.Repo.UpdateStatus(ctx, id, entity.NFSeStatusCancelada)
	if err != nil {
		return nil, err
	}
	if uc.Titulos != nil {
		if err := uc.Titulos.CancelarTitulosDaNFSe(ctx, id); err != nil {
			slog.Error("NFS-e cancelada com título em aberto", "documento", id, "erro", err)
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"a NFS-e foi cancelada na prefeitura, mas o título a receber dela não foi cancelado: cancele-o no contas a receber (%v)", err))
		}
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

// dataHoraBrasilia: a data de emissão às 00:00 de Brasília. O layout usa o
// marcador de fuso do Go ("-07:00"); "-03:00" literal saía como "-12:00".
func dataHoraBrasilia(d time.Time) string {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.FixedZone("BRT", -3*60*60)).Format("2006-01-02T15:04:05-07:00")
}

// aliquotaMaximaISS: a LC 116/2003 (art. 8º, II) limita o ISS a 5%.
var aliquotaMaximaISS = decimal.RequireFromString("0.05")

// conferirAliquotaEDeducoes: a alíquota vai em fração (0,05 = 5%). Quem digita
// 5 querendo dizer 5% ganharia um ISS de 500% — antes passava e a nota saía com
// o imposto maior que o serviço.
func conferirAliquotaEDeducoes(aliquota, servicos, deducoes decimal.Decimal) error {
	if aliquota.IsNegative() || aliquota.GreaterThan(aliquotaMaximaISS) {
		return errorsuc.NewValidationError(fmt.Sprintf("alíquota do ISS %s fora do limite: informe em fração, de 0 a 0,05 (5%%) — por exemplo, 0,03 para 3%%", aliquota.String()))
	}
	if deducoes.IsNegative() || deducoes.GreaterThan(servicos) {
		return errorsuc.NewValidationError("as deduções não podem ser negativas nem maiores que o valor dos serviços")
	}
	return nil
}

// enderecoDoTomador monta o "tomador.endereco"; sem nenhum campo, não vai.
func enderecoDoTomador(n *entity.NFSe) *focusnfe.NFSeEndereco {
	e := &focusnfe.NFSeEndereco{
		Logradouro:      deref(n.TomadorLogradouro),
		Numero:          deref(n.TomadorNumero),
		Complemento:     deref(n.TomadorComplemento),
		Bairro:          deref(n.TomadorBairro),
		CodigoMunicipio: deref(n.TomadorCodigoMunicipio),
		UF:              deref(n.TomadorUF),
		CEP:             deref(n.TomadorCEP),
	}
	if *e == (focusnfe.NFSeEndereco{}) {
		return nil
	}
	return e
}

// payloadDaNFSe monta o corpo que a Focus recebe para emitir a NFS-e.
func payloadDaNFSe(n *entity.NFSe, cfg *fiscalentity.FiscalConfig) focusnfe.NFSePayload {
	payload := focusnfe.NFSePayload{
		DataEmissao:            dataHoraBrasilia(n.DataEmissao),
		NaturezaOperacao:       n.NaturezaOperacao,
		OptanteSimplesNacional: n.OptanteSimples,
		IncentivadorCultural:   n.IncentivadorCultural,
		Prestador: focusnfe.NFSePrestador{
			CNPJ:               cfg.CnpjEmpresa,
			InscricaoMunicipal: deref(cfg.InscricaoMunicipal),
			CodigoMunicipio:    cfg.CodigoMunicipio,
		},
		Tomador: focusnfe.NFSeTomador{
			RazaoSocial: deref(n.TomadorRazaoSocial),
			Email:       deref(n.TomadorEmail),
			Endereco:    enderecoDoTomador(n),
		},
		Servico: focusnfe.NFSeServico{
			// Gravada em fração (0,05 = 5%); a Focus recebe o percentual.
			Aliquota:                  n.AliquotaISS.Mul(decimal.NewFromInt(100)).Round(4).InexactFloat64(),
			Discriminacao:             n.Discriminacao,
			IssRetido:                 n.IssRetido,
			ItemListaServico:          n.ItemListaServico,
			CodigoTributarioMunicipio: deref(n.CodigoTributarioMunicipio),
			CodigoMunicipio:           n.CodigoMunicipio,
			ValorServicos:             n.ValorServicos.InexactFloat64(),
			ValorDeducoes:             n.ValorDeducoes.InexactFloat64(),
			ValorIss:                  n.ValorISS.InexactFloat64(),
		},
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
	return payload
}
