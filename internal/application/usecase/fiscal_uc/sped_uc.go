package fiscal_uc

import (
	"context"
	"errors"
	"time"

	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped"
)

type SPEDUseCase struct {
	FiscalParamsRepo repository.FiscalParamsRepository
}

// SpedRequest carries parameters for a single EFD generation run.
type SpedRequest struct {
	Empresa     sped.EFDEmpresa
	DataInicial time.Time
	DataFinal   time.Time
	// IndicadorSituacaoEspecial: "0" regular, "1" retificadora
	IndicadorSituacao string
	// DocumentosFiscais, Inventario etc. would be populated from NF repository
	// (not wired to a live NF repo here — caller passes them directly for flexibility)
	DocumentosFiscais []sped.EFDDocumentoFiscal
	Participantes     []sped.EFDParticipante
	Unidades          []sped.EFDUnidade
	Itens             []sped.EFDItem
	Inventario        []sped.EFDInventarioItem
}

// GenerateEFD builds the EFD ICMS/IPI text using ICMS summary entries from the
// fiscal params repository as apuração data.
func (uc *SPEDUseCase) GenerateEFD(ctx context.Context, req SpedRequest) (string, error) {
	if req.Empresa.CNPJ == "" {
		return "", errors.New("empresa.cnpj is required")
	}
	if req.DataInicial.IsZero() || req.DataFinal.IsZero() {
		return "", errors.New("data_inicial and data_final are required")
	}
	if req.DataFinal.Before(req.DataInicial) {
		return "", errors.New("data_final must be after data_inicial")
	}
	if req.Empresa.RegimeTributario == "" {
		req.Empresa.RegimeTributario = "2"
	}
	if req.IndicadorSituacao == "" {
		req.IndicadorSituacao = "0"
	}

	period := req.DataInicial.Format("2006-01")
	summaries, err := uc.FiscalParamsRepo.ListICMSSummaryEntries(ctx, period, req.Empresa.UF)
	if err != nil {
		return "", err
	}

	apuracao := buildApuracaoFromSummaries(summaries)

	params := sped.EFDParams{
		Empresa: req.Empresa,
		Periodo: sped.EFDPeriodo{
			DataInicial:               req.DataInicial,
			DataFinal:                 req.DataFinal,
			IndicadorSituacaoEspecial: req.IndicadorSituacao,
		},
		Participantes:     req.Participantes,
		Unidades:          req.Unidades,
		Itens:             req.Itens,
		DocumentosFiscais: req.DocumentosFiscais,
		ApuracaoICMS:      apuracao,
		Inventario:        req.Inventario,
	}
	return sped.Generate(params), nil
}

func buildApuracaoFromSummaries(summaries []*fiscalEntity.ICMSSummaryEntry) *sped.EFDApuracaoICMS {
	if len(summaries) == 0 {
		return &sped.EFDApuracaoICMS{}
	}
	var totalBase, totalICMS float64
	for _, s := range summaries {
		totalBase += s.ICMSBase
		totalICMS += s.ICMSValue
	}
	return &sped.EFDApuracaoICMS{
		VlTotDebitos:   totalICMS,
		VlApuracao:     totalICMS,
		VlIcmsRecolher: totalICMS,
	}
}
