package fiscal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// BeneficioFiscalPG busca o código de benefício fiscal (cBenef) nos parâmetros
// de ICMS/IPI da empresa: parâmetro do item vence o do NCM, o do cliente vence o
// geral. Só parâmetros ativos de saída (SAIDA ou AMBAS) da UF de destino.
type BeneficioFiscalPG struct {
	pool *pgxpool.Pool
}

func NewBeneficioFiscalPG(pool *pgxpool.Pool) *BeneficioFiscalPG {
	return &BeneficioFiscalPG{pool: pool}
}

func (r *BeneficioFiscalPG) BeneficioFiscal(ctx context.Context, itemCode *int64, ncm *string, ufDestino string, cliente *int64) (*string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ncmDigitos := ""
	if ncm != nil {
		ncmDigitos = strings.Map(func(c rune) rune {
			if c >= '0' && c <= '9' {
				return c
			}
			return -1
		}, *ncm)
	}
	var codigo *string
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(btrim(cod_beneficio_fiscal), ''), NULLIF(btrim(cod_benef_rbc), ''))
		  FROM icms_ipi_tax_params
		 WHERE enterprise_id = $1 AND is_active
		   AND operation_type IN ('SAIDA', 'AMBAS')
		   AND uf = upper($4)
		   AND ((item_code IS NOT NULL AND item_code = $2)
		        OR (item_code IS NULL AND $3 <> '' AND regexp_replace(COALESCE(ncm_code, ''), '[^0-9]', '', 'g') = $3))
		   AND (customer_code IS NULL OR customer_code = $5)
		   AND COALESCE(NULLIF(btrim(cod_beneficio_fiscal), ''), NULLIF(btrim(cod_benef_rbc), '')) IS NOT NULL
		 ORDER BY (item_code IS NOT NULL) DESC, (customer_code IS NOT NULL) DESC, id
		 LIMIT 1`, empresa, itemCode, ncmDigitos, strings.TrimSpace(ufDestino), cliente).Scan(&codigo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("buscando o código de benefício fiscal: %w", err)
	}
	return codigo, nil
}
