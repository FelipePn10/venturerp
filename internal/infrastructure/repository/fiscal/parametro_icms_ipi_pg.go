package fiscal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ParametroICMSIPIPG acha, no cadastro de parâmetros de ICMS/IPI (VFIS0320), a
// regra que vale para uma linha de nota. Só regras ativas da operação (SAIDA ou
// AMBAS na saída; ENTRADA ou AMBAS na entrada) e da UF da outra ponta. Da mais
// específica para a mais geral:
//
//  1. regra do item vence regra de NCM; entre regras de NCM, a de prefixo mais
//     longo (o NCM completo vence o capítulo "7308");
//  2. regra do tipo de nota da saída vence a sem tipo;
//  3. regra do cliente vence a do segmento de mercado do cliente, que vence a
//     da tributação do cliente, que vence a geral;
//  4. regra da finalidade da compra (revenda, industrialização, uso e consumo)
//     vence a sem finalidade;
//  5. regra de optante do Simples vence a comum, na empresa do Simples;
//  6. regra marcada como preferencial desempata; depois, a mais antiga.
//
// Regra restrita a um tipo de nota, a uma finalidade, a uma tributação ou a um
// segmento só vale quando a nota tem esse mesmo dado. Regras restritas a
// máscara de item configurado, a estabelecimento ou a fornecedor (regra de
// entrada) não entram: a consulta não carrega esses dados. Regra "optante do Simples" só vale quando a empresa é
// do Simples Nacional.
type ParametroICMSIPIPG struct {
	pool *pgxpool.Pool
}

func NewParametroICMSIPIPG(pool *pgxpool.Pool) *ParametroICMSIPIPG {
	return &ParametroICMSIPIPG{pool: pool}
}

func (r *ParametroICMSIPIPG) ParametroICMSIPI(ctx context.Context, q engine.ConsultaParametroICMSIPI) (*engine.ParametroICMSIPI, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ncmDigitos := ""
	if q.NCM != nil {
		ncmDigitos = engine.NormalizarNCM(*q.NCM)
	}
	operacoes := []string{"SAIDA", "AMBAS"}
	if q.Entrada {
		operacoes = []string{"ENTRADA", "AMBAS"}
	}
	var finalidade *string
	if f := strings.TrimSpace(q.Finalidade); f != "" {
		finalidade = &f
	}
	var (
		p                                         engine.ParametroICMSIPI
		aliqC, redC, ipiC, aliqN, redN, ipiN      decimal.Decimal
		stC, stN, redST, mva, difRed              decimal.Decimal
		dif, imp                                  decimal.Decimal
		cstC, alvoC, alvoIPIC, benC               *string
		cstN, alvoN, alvoIPIN, benN, cstIPI, benG *string
		csosn, difTipo                            *string
		fcpProd                                   decimal.Decimal
	)
	err = r.pool.QueryRow(ctx, `
		WITH cli AS (
		    SELECT c.market_segment_id, c.tax_type_id FROM customers c
		     WHERE c.code = $5 AND c.enterprise_id = $1
		)
		SELECT p.id,
		       p.icms_pct_contrib, p.cst_icms_contrib, p.icms_red_pct_contrib, p.icms_red_target_contrib::text,
		       p.ipi_red_pct_contrib, p.ipi_red_target_contrib::text, p.cod_benef_contrib, p.icms_pct_for_st_contrib,
		       p.icms_pct_non_contrib, p.cst_icms_non_contrib, p.icms_red_pct_non_contrib, p.icms_red_target_non_contrib::text,
		       p.ipi_red_pct_non_contrib, p.ipi_red_target_non_contrib::text, p.cod_benef_non_contrib, p.icms_pct_for_st_non_contrib,
		       p.icms_deferral_pct, p.icms_pct_origins_1238, p.cst_ipi_exit,
		       COALESCE(NULLIF(btrim(p.cod_beneficio_fiscal), ''), NULLIF(btrim(p.cod_benef_rbc), '')),
		       NULLIF(btrim(p.csosn_icms), ''), p.mva_pct, p.mva_ajustar_interestadual, p.icms_subst_red_pct,
		       p.icms_difal_type::text, p.icms_difal_red_pct,
		       CASE WHEN p.icms_acres_type_non_contrib::text = 'FUNDO_COMBATE_POBREZA' THEN p.icms_acres_pct_non_contrib ELSE 0 END,
		       COALESCE(d1.description,''), COALESCE(d2.description,''), COALESCE(d3.description,''), COALESCE(d4.description,''),
		       COALESCE(d5.description,''), COALESCE(d6.description,''), COALESCE(d7.description,'')
		  FROM icms_ipi_tax_params p
		  LEFT JOIN legal_devices d1 ON d1.id = p.legal_device_icms_contrib_id AND d1.is_active
		  LEFT JOIN legal_devices d2 ON d2.id = p.legal_device_icms_non_contrib_id AND d2.is_active
		  LEFT JOIN legal_devices d3 ON d3.id = p.legal_device_icms_red_contrib_id AND d3.is_active
		  LEFT JOIN legal_devices d4 ON d4.id = p.legal_device_icms_red_non_contrib_id AND d4.is_active
		  LEFT JOIN legal_devices d5 ON d5.id = p.legal_device_icms_deferral_id AND d5.is_active
		  LEFT JOIN legal_devices d6 ON d6.id = p.legal_device_ipi_contrib_id AND d6.is_active
		  LEFT JOIN legal_devices d7 ON d7.id = p.legal_device_ipi_non_contrib_id AND d7.is_active
		 WHERE p.enterprise_id = $1 AND p.is_active
		   AND p.operation_type::text = ANY($8)
		   AND p.uf = upper($2)
		   AND ((p.item_code IS NOT NULL AND p.item_code = $3 AND NULLIF(btrim(p.item_config_mask), '') IS NULL)
		        OR (p.item_code IS NULL AND $4 <> ''
		            AND regexp_replace(COALESCE(p.ncm_code, ''), '[^0-9]', '', 'g') <> ''
		            AND $4 LIKE regexp_replace(p.ncm_code, '[^0-9]', '', 'g') || '%'))
		   AND (p.customer_code IS NULL OR p.customer_code = $5)
		   AND (p.market_segment_id IS NULL OR p.market_segment_id = (SELECT market_segment_id FROM cli))
		   AND (p.tax_type_id IS NULL OR p.tax_type_id = (SELECT tax_type_id FROM cli))
		   AND (p.finalidade_compra IS NULL OR p.finalidade_compra = $7)
		   AND (p.invoice_type_exit_id IS NULL OR (NOT $10 AND p.invoice_type_exit_id = $9))
		   AND (p.invoice_type_entry_id IS NULL OR NOT $10)
		   AND p.customer_establishment_code IS NULL
		   -- Regra restrita a fornecedor é de entrada; a consulta não traz fornecedor.
		   AND p.supplier_code IS NULL
		   AND (NOT p.is_simples_optante OR $6)
		 ORDER BY (p.item_code IS NOT NULL) DESC,
		          length(regexp_replace(COALESCE(p.ncm_code, ''), '[^0-9]', '', 'g')) DESC,
		          (p.invoice_type_exit_id IS NOT NULL) DESC,
		          (p.customer_code IS NOT NULL) DESC, (p.market_segment_id IS NOT NULL) DESC,
		          (p.tax_type_id IS NOT NULL) DESC, (p.finalidade_compra IS NOT NULL) DESC,
		          p.is_simples_optante DESC, p.is_preferred DESC, p.id
		 LIMIT 1`, empresa, strings.TrimSpace(q.UF), q.ItemCode, ncmDigitos, q.Cliente, q.Simples,
		finalidade, operacoes, q.TipoNotaID, q.Entrada).Scan(
		&p.ID,
		&aliqC, &cstC, &redC, &alvoC, &ipiC, &alvoIPIC, &benC, &stC,
		&aliqN, &cstN, &redN, &alvoN, &ipiN, &alvoIPIN, &benN, &stN,
		&dif, &imp, &cstIPI, &benG,
		&csosn, &mva, &p.MvaAjustar, &redST, &difTipo, &difRed, &fcpProd,
		&p.Dispositivos.ICMSContrib, &p.Dispositivos.ICMSNaoContrib, &p.Dispositivos.ReducaoContrib, &p.Dispositivos.ReducaoNaoContrib,
		&p.Dispositivos.Diferimento, &p.Dispositivos.IPIContrib, &p.Dispositivos.IPINaoContrib)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("buscando o parâmetro de ICMS/IPI: %w", err)
	}
	p.Contrib = engine.LadoParametroICMSIPI{
		AliqICMSPct: aliqC.InexactFloat64(), CSTICMS: texto(cstC), RedICMSPct: redC.InexactFloat64(), AlvoRedICMS: texto(alvoC),
		RedIPIPct: ipiC.InexactFloat64(), AlvoRedIPI: texto(alvoIPIC), CodBeneficio: texto(benC), AliqSTPct: stC.InexactFloat64(),
	}
	p.NaoContrib = engine.LadoParametroICMSIPI{
		AliqICMSPct: aliqN.InexactFloat64(), CSTICMS: texto(cstN), RedICMSPct: redN.InexactFloat64(), AlvoRedICMS: texto(alvoN),
		RedIPIPct: ipiN.InexactFloat64(), AlvoRedIPI: texto(alvoIPIN), CodBeneficio: texto(benN), AliqSTPct: stN.InexactFloat64(),
	}
	p.DiferimentoPct = dif.InexactFloat64()
	p.AliqImportadoPct = imp.InexactFloat64()
	p.CSTIPI = texto(cstIPI)
	p.CodBeneficio = texto(benG)
	p.CSOSN = texto(csosn)
	p.MvaPct = mva.InexactFloat64()
	p.RedBaseSTPct = redST.InexactFloat64()
	p.DifalTipo = texto(difTipo)
	p.DifalRedPct = difRed.InexactFloat64()
	p.FCPProdutoPct = fcpProd.InexactFloat64()
	return &p, nil
}

func texto(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
