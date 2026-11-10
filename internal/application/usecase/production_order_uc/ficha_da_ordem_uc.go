package production_order_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/google/uuid"
)

// LeitorDaFicha lê a ordem como ela vai impressa.
type LeitorDaFicha interface {
	FichaDaOrdem(ctx context.Context, id int64) (*entity.FichaDaOrdem, error)
}

// LeitorDaMarca dá o cabeçalho da empresa (razão social, CNPJ, logo, cor).
type LeitorDaMarca interface {
	GetFiscalConfig(ctx context.Context) (*fiscalentity.FiscalConfig, error)
}

// EmpresaDaFicha é o timbre do documento. Vazio quando a empresa ainda não
// configurou o fiscal — a ficha sai mesmo assim.
type EmpresaDaFicha struct {
	Nome, CNPJ, IE, Endereco, Telefone, Email string
	Logo                                      []byte
	Cor                                       string
}

type FichaImpressa struct {
	Ficha   *entity.FichaDaOrdem
	Empresa EmpresaDaFicha
	// EmitidaPor: quem imprimiu (vai no rodapé e no campo do PCP).
	EmitidaPor string
}

type nomeDoUsuario interface {
	NomeDoUsuario(ctx context.Context, id uuid.UUID) (string, error)
}

// FichaDaOrdemUseCase junta o que a ordem impressa precisa. Lê as mesmas
// permissões da consulta da ordem.
type FichaDaOrdemUseCase struct {
	Repo  LeitorDaFicha
	Marca LeitorDaMarca
	Auth  ports.AuthService
}

func (uc *FichaDaOrdemUseCase) Execute(ctx context.Context, id int64) (*FichaImpressa, error) {
	// O posto (OPERATOR) lê a ordem em que vai apontar: sem isso a VPRO0900,
	// uma das três telas do perfil, não abria nenhuma OF.
	if !uc.Auth.CanListOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if id <= 0 {
		return nil, errorsuc.NewValidationError("informe a ordem")
	}
	ficha, err := uc.Repo.FichaDaOrdem(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &FichaImpressa{Ficha: ficha}
	if leitor, ok := uc.Repo.(nomeDoUsuario); ok {
		if id, err := uc.Auth.UserID(ctx); err == nil {
			nome, err := leitor.NomeDoUsuario(ctx, id)
			if err != nil {
				return nil, err
			}
			out.EmitidaPor = nome
		}
	}
	if uc.Marca != nil {
		// Sem configuração fiscal a ordem ainda imprime — só sem timbre.
		if cfg, err := uc.Marca.GetFiscalConfig(ctx); err == nil && cfg != nil {
			out.Empresa = empresaDaConfig(cfg)
		}
	}
	return out, nil
}

func empresaDaConfig(cfg *fiscalentity.FiscalConfig) EmpresaDaFicha {
	e := EmpresaDaFicha{Nome: cfg.RazaoSocial, CNPJ: cfg.CnpjEmpresa, Email: cfg.Email, Logo: cfg.Logo}
	if cfg.IEEmpresa != nil {
		e.IE = *cfg.IEEmpresa
	}
	if cfg.Telefone != nil {
		e.Telefone = *cfg.Telefone
	}
	if cfg.BrandColor != nil {
		e.Cor = *cfg.BrandColor
	}
	var partes []string
	rua := strings.TrimSpace(strings.Join([]string{cfg.Logradouro, cfg.Numero}, ", "))
	if strings.Trim(rua, ", ") != "" {
		partes = append(partes, strings.Trim(rua, ", "))
	}
	if cfg.Bairro != "" {
		partes = append(partes, cfg.Bairro)
	}
	if cfg.Municipio != "" {
		partes = append(partes, strings.TrimSpace(cfg.Municipio+" "+cfg.UFEmpresa))
	}
	e.Endereco = strings.Join(partes, " · ")
	return e
}

// LeitorDoPosto dá ao posto as listas mínimas para apontar.
type LeitorDoPosto interface {
	CadastrosDoPosto(ctx context.Context) (*entity.CadastrosDoPosto, error)
}

// CadastrosDoPostoUseCase: máquinas, operadores e depósitos (código e nome)
// para quem aponta produção.
type CadastrosDoPostoUseCase struct {
	Repo LeitorDoPosto
	Auth ports.AuthService
}

func (uc *CadastrosDoPostoUseCase) Execute(ctx context.Context) (*entity.CadastrosDoPosto, error) {
	if !uc.Auth.CanReportProduction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.CadastrosDoPosto(ctx)
}
