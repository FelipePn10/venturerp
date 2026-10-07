package fiscal_uc

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"strings"

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
	tipo := normalizarTipoManifestacao(dto.Tipo)
	if !tiposManifestacao[tipo] {
		return nil, errorsuc.NewValidationError("tipo de manifestação inválido: ciencia, confirmacao, desconhecimento ou nao_realizada")
	}
	if len(soDigitos(dto.ChaveNFe)) != 44 {
		return nil, errorsuc.NewValidationError("a chave de acesso deve ter 44 dígitos")
	}
	if (tipo == "desconhecimento" || tipo == "nao_realizada") && len([]rune(strings.TrimSpace(dto.Justificativa))) < 15 {
		return nil, errorsuc.NewValidationError("informe a justificativa (pelo menos 15 caracteres)")
	}
	cli, cfg, err := newFocusFromConfig(ctx, uc.Repo)
	if err != nil {
		return nil, err
	}
	return cli.ManifestarDestinatario(ctx, focusnfe.ManifestacaoPayload{
		CNPJ:          cfg.CnpjEmpresa,
		ChaveNFe:      dto.ChaveNFe,
		Tipo:          tipo,
		Justificativa: dto.Justificativa,
	})
}

// normalizarTipoManifestacao aceita a grafia das telas antigas (CIENCIA,
// OPERACAO_NAO_REALIZADA...) e devolve a que a Focus NF-e espera.
func normalizarTipoManifestacao(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "operacao_nao_realizada", "nao-realizada", "naorealizada":
		return "nao_realizada"
	case "ciência":
		return "ciencia"
	case "confirmação":
		return "confirmacao"
	}
	return t
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
		return nil, errorsuc.NewValidationError("o número final deve ser maior ou igual ao número inicial")
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
		return nil, nil, errorsuc.NewValidationError("o token da Focus NF-e não está configurado — acesse Configurações Fiscais")
	}
	return focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente), cfg, nil
}
