package fiscal_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// ManifestarDestinatarioUseCase registers the recipient's manifestation about an
// incoming NF-e (ciência/confirmação/desconhecimento/não realizada) at SEFAZ.
type ManifestarDestinatarioUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

type ManifestarDestinatarioDTO struct {
	ChaveNFe      string `json:"chave_nfe"`
	Tipo          string `json:"tipo"`
	Justificativa string `json:"justificativa,omitempty"`
}

func (uc *ManifestarDestinatarioUseCase) Execute(ctx context.Context, dto ManifestarDestinatarioDTO) (map[string]interface{}, error) {
	if !uc.Auth.CanAuthorizeFiscalExit(ctx) {
		return nil, fmt.Errorf("não autorizado")
	}
	cli, cfg, err := newFocusFromConfig(ctx, uc.Repo)
	if err != nil {
		return nil, err
	}
	return cli.ManifestarDestinatario(ctx, focusnfe.ManifestacaoPayload{
		CNPJ:          cfg.CnpjEmpresa,
		ChaveNFe:      dto.ChaveNFe,
		Tipo:          dto.Tipo,
		Justificativa: dto.Justificativa,
	})
}

// InutilizarNumeracaoUseCase invalidates a range of unused NF-e numbers at SEFAZ.
type InutilizarNumeracaoUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

type InutilizarNumeracaoDTO struct {
	Serie         int    `json:"serie"`
	NumeroInicial int    `json:"numero_inicial"`
	NumeroFinal   int    `json:"numero_final"`
	Justificativa string `json:"justificativa"`
}

func (uc *InutilizarNumeracaoUseCase) Execute(ctx context.Context, dto InutilizarNumeracaoDTO) (map[string]interface{}, error) {
	if !uc.Auth.CanAuthorizeFiscalExit(ctx) {
		return nil, fmt.Errorf("não autorizado")
	}
	if dto.NumeroFinal < dto.NumeroInicial {
		return nil, fmt.Errorf("numero_final deve ser >= numero_inicial")
	}
	cli, cfg, err := newFocusFromConfig(ctx, uc.Repo)
	if err != nil {
		return nil, err
	}
	return cli.InutilizarNumeracao(ctx, focusnfe.InutilizacaoPayload{
		CNPJ:          cfg.CnpjEmpresa,
		Serie:         dto.Serie,
		NumeroInicial: dto.NumeroInicial,
		NumeroFinal:   dto.NumeroFinal,
		Justificativa: dto.Justificativa,
	})
}

// newFocusFromConfig builds a Focus client from the stored fiscal configuration.
func newFocusFromConfig(ctx context.Context, repo repository.FiscalRepository) (*focusnfe.Client, *entity.FiscalConfig, error) {
	cfg, err := repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("reading fiscal config: %w", err)
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, nil, fmt.Errorf("token Focus NF-e não configurado")
	}
	return focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente), cfg, nil
}
