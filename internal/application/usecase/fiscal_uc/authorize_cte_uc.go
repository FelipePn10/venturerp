package fiscal_uc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// AuthorizeCTeUseCase transmits a locally registered CT-e to SEFAZ via Focus.
// The CT-e must carry emission_data (the structured detail of parties, modal and
// municipalities); the emitente is filled from the fiscal config.
type AuthorizeCTeUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *AuthorizeCTeUseCase) Execute(ctx context.Context, id int64) (*entity.FiscalCTe, error) {
	if !uc.Auth.CanAuthorizeFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	cte, err := uc.Repo.GetCTeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cte.Status == "AUTORIZADO" {
		return nil, fmt.Errorf("CT-e %d já está autorizado", id)
	}
	if cte.EmissionData == nil || *cte.EmissionData == "" {
		return nil, fmt.Errorf("CT-e %d não possui emission_data — informe os dados de emissão (partes, modal, municípios) para autorizar", id)
	}

	var payload focusnfe.CTePayload
	if err := json.Unmarshal([]byte(*cte.EmissionData), &payload); err != nil {
		return nil, fmt.Errorf("emission_data inválido: %w", err)
	}

	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, fmt.Errorf("token Focus NF-e não configurado — acesse Configurações Fiscais")
	}

	// Defaults and emitente from the fiscal config (the company is the emitter).
	if payload.Modal == "" {
		payload.Modal = "01" // rodoviário
	}
	if payload.DataEmissao == "" {
		payload.DataEmissao = cte.DataEmissao.Format("2006-01-02T15:04:05-03:00")
	}
	if payload.NaturezaOperacao == "" {
		payload.NaturezaOperacao = "Prestação de serviço de transporte"
	}
	if payload.ValorTotalPrestacao == 0 {
		payload.ValorTotalPrestacao = cte.ValorTotal
	}
	if payload.ValorReceber == 0 {
		payload.ValorReceber = cte.ValorTotal
	}
	if payload.ICMS.SituacaoTributaria == "" {
		st := "90"
		if cte.CstICMS != nil && *cte.CstICMS != "" {
			st = *cte.CstICMS
		}
		payload.ICMS = focusnfe.CTeICMS{
			SituacaoTributaria: st,
			BaseCalculo:        cte.BaseICMS,
			Aliquota:           cte.AliqICMS * 100,
			Valor:              cte.ValorICMS,
		}
	}
	payload.Emitente = focusnfe.CTeParte{
		CNPJ:            cfg.CnpjEmpresa,
		IE:              derefStr(cfg.IEEmpresa),
		Nome:            cfg.RazaoSocial,
		Logradouro:      cfg.Logradouro,
		Numero:          cfg.Numero,
		Bairro:          cfg.Bairro,
		Municipio:       cfg.Municipio,
		CodigoMunicipio: cfg.CodigoMunicipio,
		UF:              cfg.UFEmpresa,
		CEP:             cfg.CEP,
		Telefone:        derefStr(cfg.Telefone),
	}

	focusCli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	focusCli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		_ = uc.Repo.SaveFocusLog(ctx, id, endpoint, method, reqBody, respBody, statusCode, durationMs)
	})

	ref := fmt.Sprintf("cte%d%d", id, time.Now().UnixNano()%1000000)
	if len(ref) > 50 {
		ref = ref[:50]
	}

	resp, err := focusCli.AutorizarCTe(ctx, ref, payload)
	if err != nil {
		_, _ = uc.Repo.UpdateCTeStatus(ctx, id, "REJEITADO")
		return nil, fmt.Errorf("Focus CT-e: %w", err)
	}

	return uc.Repo.UpdateCTeAuthorization(ctx, id, resp.ChaveCTe, resp.Protocolo, ref)
}
