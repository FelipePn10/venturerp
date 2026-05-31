package supplier_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/supplier/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// unsupportedSefazUFs are states without the SEFAZ cadastral query service.
var unsupportedSefazUFs = map[string]bool{
	"AL": true, "AP": true, "DF": true, "MA": true, "PA": true,
	"PI": true, "RJ": true, "RO": true, "RR": true, "SE": true, "TO": true,
}

type SefazQueryResult struct {
	SupplierCode      int64  `json:"supplier_code"`
	Situation         string `json:"billing_receipt_status"`
	Nome              string `json:"nome,omitempty"`
	SituacaoCadastral string `json:"situacao_cadastral,omitempty"`
	UF                string `json:"uf,omitempty"`
	QueriedAt         string `json:"queried_at"`
}

// ConsultSupplierSefazUseCase queries a supplier's cadastral situation on SEFAZ
// via FocusNFE and records the snapshot on the supplier.
type ConsultSupplierSefazUseCase struct {
	Repo       repository.SupplierRepository
	FiscalRepo fiscalrepo.FiscalRepository
	Auth       ports.AuthService
}

func (uc *ConsultSupplierSefazUseCase) Execute(ctx context.Context, supplierCode int64) (*SefazQueryResult, error) {
	s, err := uc.Repo.GetSupplierByCode(ctx, supplierCode)
	if err != nil {
		return nil, err
	}

	// Determine the supplier UF from its default (or first) address.
	uf := ""
	if addrs, aerr := uc.Repo.ListAddresses(ctx, s.ID); aerr == nil {
		for _, a := range addrs {
			if a.IsDefault && a.UF != nil && *a.UF != "" {
				uf = strings.ToUpper(*a.UF)
				break
			}
		}
		if uf == "" {
			for _, a := range addrs {
				if a.UF != nil && *a.UF != "" {
					uf = strings.ToUpper(*a.UF)
					break
				}
			}
		}
	}
	if uf != "" && unsupportedSefazUFs[uf] {
		return nil, fmt.Errorf("o estado %s não é contemplado pelo processo de consulta cadastral junto à SEFAZ", uf)
	}

	cfg, err := uc.FiscalRepo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading fiscal config: %w", err)
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, fmt.Errorf("token Focus NF-e não configurado")
	}

	cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	resp, err := cli.ConsultarCadastro(ctx, s.DocumentNumber)
	if err != nil {
		return nil, err
	}

	situation := "BLOQUEADO"
	if resp.Habilitado || strings.Contains(strings.ToUpper(resp.SituacaoCadastral), "ATIV") {
		situation = "LIBERADO"
	}

	user := ""
	if uid, uerr := uc.Auth.UserID(ctx); uerr == nil {
		user = uid.String()
	}

	if err := uc.Repo.UpdateSefazSnapshot(ctx, supplierCode, situation, user); err != nil {
		return nil, err
	}

	return &SefazQueryResult{
		SupplierCode:      supplierCode,
		Situation:         situation,
		Nome:              resp.Nome,
		SituacaoCadastral: resp.SituacaoCadastral,
		UF:                uf,
		QueriedAt:         time.Now().Format("2006-01-02"),
	}, nil
}
