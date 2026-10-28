-- Unifica o cadastro "Redução / Substituição / Diferimento de ICMS" (VFIS0330,
-- icms_reduction_substitutions) no de parâmetros de ICMS/IPI (VFIS0320,
-- icms_ipi_tax_params). Os dois descreviam a mesma coisa — tributação de ICMS/
-- IPI por UF, item/NCM, cliente e operação — mas só a VFIS0320 entra no cálculo
-- da nota: a regra gravada na VFIS0330 era ignorada sem aviso.
--
-- 1. Percentuais da VFIS0320 passam de 2 para 4 casas (a VFIS0330 tinha 4; um
--    diferimento de 33,3333% não pode virar 33,33% na cópia).
-- 2. Fornecedor (supplier_code) entra na VFIS0320: a VFIS0330 restringia regra
--    de entrada por fornecedor. Regra com fornecedor não vale na saída.
-- 3. Cada regra da VFIS0330 é copiada para a VFIS0320 da MESMA empresa, com item
--    e cliente convertidos de id para código. origem_reducao_id marca a cópia
--    (não copia duas vezes e permite desfazer). Regra cujo item, cliente,
--    fornecedor, tipo de nota ou segmento não exista na própria empresa não é
--    copiada — ficaria com o vínculo trocado ou mais ampla do que foi cadastrada.
-- O "estabelecimento" da VFIS0330 era o da própria empresa e já está no
-- enterprise_id da regra; não tem equivalente na VFIS0320 (que só tem o
-- estabelecimento do cliente) e não é copiado.
-- A tabela antiga fica como histórico, sem uso; a API passa a recusar gravação nela.

DO $$
DECLARE c record;
BEGIN
    FOR c IN SELECT column_name FROM information_schema.columns
              WHERE table_schema = 'public' AND table_name = 'icms_ipi_tax_params'
                AND data_type = 'numeric' AND numeric_precision = 5 AND numeric_scale = 2
    LOOP
        EXECUTE format('ALTER TABLE icms_ipi_tax_params ALTER COLUMN %I TYPE numeric(7,4)', c.column_name);
    END LOOP;
END $$;

ALTER TABLE icms_ipi_tax_params
    ADD COLUMN IF NOT EXISTS supplier_code     bigint,
    ADD COLUMN IF NOT EXISTS origem_reducao_id bigint;
CREATE UNIQUE INDEX IF NOT EXISTS uq_icms_ipi_tax_params_origem_reducao
    ON icms_ipi_tax_params (origem_reducao_id) WHERE origem_reducao_id IS NOT NULL;
COMMENT ON COLUMN icms_ipi_tax_params.supplier_code IS 'fornecedor (suppliers.code) — restringe regra de entrada; não vale na saída';
-- O fornecedor da regra tem de ser da empresa (conferência por gatilho, como
-- os demais vínculos).
SELECT instalar_conferencia_de_empresa();
COMMENT ON COLUMN icms_ipi_tax_params.origem_reducao_id IS 'icms_reduction_substitutions.id de onde a regra foi copiada (migração 000433)';

INSERT INTO icms_ipi_tax_params (
    enterprise_id, origem_reducao_id, ncm_code, item_code, item_config_mask, uf, operation_type,
    customer_code, supplier_code, market_segment_id, invoice_type_exit_id, invoice_type_entry_id,
    is_preferred, is_simples_optante, is_active,
    icms_pct_contrib, legal_device_icms_contrib_id, icms_pct_non_contrib, legal_device_icms_non_contrib_id,
    icms_red_pct_contrib, icms_red_target_contrib, legal_device_icms_red_contrib_id,
    icms_red_pct_non_contrib, icms_red_target_non_contrib, legal_device_icms_red_non_contrib_id,
    icms_deferral_pct, icms_deferral_target, legal_device_icms_deferral_id,
    icms_subst_pct_contrib, legal_device_icms_subst_contrib_id, icms_subst_pct_non_contrib, legal_device_icms_subst_non_contrib_id,
    icms_subst_pct_contrib_uc, icms_subst_red_pct, legal_device_icms_subst_red_id, icms_internal_pct, bc_icms_st_modality,
    icms_pct_for_st_contrib, icms_pct_for_st_non_contrib,
    cst_situation_b, csosn_icms, cst_icms_contrib, cst_icms_non_contrib, cst_icms_contrib_dev, cst_icms_non_contrib_dev,
    cod_beneficio_fiscal, cod_benef_contrib, cod_benef_non_contrib,
    ipi_red_pct_contrib, ipi_red_target_contrib, legal_device_ipi_contrib_id,
    ipi_red_pct_non_contrib, ipi_red_target_non_contrib, legal_device_ipi_non_contrib_id,
    cst_ipi_exit, cst_ipi_entry,
    icms_pct_origins_1238, calc_base_red_fci, icms_subst_pct_origins_1238, cst_icms_fci, uses_icms_zona_franca, dif_aliq_st_contrib_uc,
    icms_acres_pct_contrib, icms_acres_type_contrib, icms_acres_sum_contrib,
    icms_acres_pct_non_contrib, icms_acres_type_non_contrib, icms_acres_sum_non_contrib,
    icms_st_acres_pct_contrib, icms_st_acres_type_contrib, icms_st_acres_pct_non_contrib, icms_st_acres_type_non_contrib,
    fcp_st_partilha_pct, icms_difal_red_pct, icms_difal_type, difal_purchase_red_pct)
SELECT
    r.enterprise_id, r.id, NULLIF(btrim(r.ncm_code), ''), i.code, NULLIF(btrim(r.item_mask), ''), upper(r.uf), r.operation_type::text::tax_param_operation_enum,
    c.code, s.code, r.market_segment_id, r.invoice_type_out_id, r.invoice_type_in_id,
    r.is_preferential, r.is_simples_optante, r.is_active,
    COALESCE(r.icms_pct_contrib, 0), r.legal_device_icms_contrib_id, COALESCE(r.icms_pct_non_contrib, 0), r.legal_device_icms_non_contrib_id,
    COALESCE(r.icms_red_pct_contrib, 0), CASE WHEN COALESCE(r.icms_red_pct_contrib, 0) > 0 THEN r.icms_red_target_contrib END, r.legal_device_icms_red_contrib_id,
    COALESCE(r.icms_red_pct_non_contrib, 0), CASE WHEN COALESCE(r.icms_red_pct_non_contrib, 0) > 0 THEN r.icms_red_target_non_contrib END, r.legal_device_icms_red_non_contrib_id,
    COALESCE(r.icms_deferral_pct, 0), CASE WHEN COALESCE(r.icms_deferral_pct, 0) > 0 THEN r.icms_deferral_target END, r.legal_device_icms_deferral_id,
    COALESCE(r.icms_subst_pct_contrib, 0), r.legal_device_icms_subst_contrib_id, COALESCE(r.icms_subst_pct_non_contrib, 0), r.legal_device_icms_subst_non_contrib_id,
    COALESCE(r.icms_subst_pct_contrib_uc, 0), COALESCE(r.icms_subst_red_pct, 0), r.legal_device_icms_subst_red_id, COALESCE(r.icms_internal_pct, 0), NULLIF(btrim(r.mod_bc_icms_st), ''),
    COALESCE(r.icms_pct_for_st_contrib, 0), COALESCE(r.icms_pct_for_st_non_contrib, 0),
    left(NULLIF(btrim(r.cst_sit_trib_b), ''), 2), NULLIF(btrim(r.csosn_icms), ''), NULLIF(btrim(r.cst_icms_contrib), ''), NULLIF(btrim(r.cst_icms_non_contrib), ''),
    NULLIF(btrim(r.cst_icms_contrib_dev), ''), NULLIF(btrim(r.cst_icms_non_contrib_dev), ''),
    -- O cBenef do diferimento só vale como geral quando a regra não tem um.
    COALESCE(NULLIF(btrim(r.fiscal_benefit_code), ''), NULLIF(btrim(r.icms_deferral_benefit_code), '')),
    NULLIF(btrim(r.fiscal_benefit_code_contrib), ''), NULLIF(btrim(r.fiscal_benefit_code_non_contrib), ''),
    COALESCE(r.ipi_red_pct_contrib, 0), CASE WHEN COALESCE(r.ipi_red_pct_contrib, 0) > 0 THEN r.ipi_red_target_contrib END, r.legal_device_ipi_contrib_id,
    COALESCE(r.ipi_red_pct_non_contrib, 0), CASE WHEN COALESCE(r.ipi_red_pct_non_contrib, 0) > 0 THEN r.ipi_red_target_non_contrib END, r.legal_device_ipi_non_contrib_id,
    left(NULLIF(btrim(r.cst_ipi_out), ''), 2), left(NULLIF(btrim(r.cst_ipi_in), ''), 2),
    COALESCE(r.fci_icms_pct, 0), r.fci_reduce_base, COALESCE(r.fci_icms_subst_pct, 0), NULLIF(btrim(r.fci_cst_icms), ''), r.fci_use_icms_zf, COALESCE(r.fci_difal_st_contrib_uc_pct, 0),
    COALESCE(r.icms_add_pct_contrib, 0),
    CASE WHEN r.icms_add_type_contrib IS NULL OR btrim(r.icms_add_type_contrib) = '' THEN NULL
         WHEN upper(r.icms_add_type_contrib) LIKE 'F%' THEN 'FUNDO_COMBATE_POBREZA' ELSE 'OUTROS' END::icms_acres_type_enum,
    r.icms_add_sum_aliq_contrib,
    COALESCE(r.icms_add_pct_non_contrib, 0),
    CASE WHEN r.icms_add_type_non_contrib IS NULL OR btrim(r.icms_add_type_non_contrib) = '' THEN NULL
         WHEN upper(r.icms_add_type_non_contrib) LIKE 'F%' THEN 'FUNDO_COMBATE_POBREZA' ELSE 'OUTROS' END::icms_acres_type_enum,
    r.icms_add_sum_aliq_non_contrib,
    COALESCE(r.icms_st_add_pct_contrib, 0),
    CASE WHEN r.icms_st_add_type_contrib IS NULL OR btrim(r.icms_st_add_type_contrib) = '' THEN NULL
         WHEN upper(r.icms_st_add_type_contrib) LIKE 'F%' THEN 'FUNDO_COMBATE_POBREZA' ELSE 'OUTROS' END::icms_acres_type_enum,
    COALESCE(r.icms_st_add_pct_non_contrib, 0),
    CASE WHEN r.icms_st_add_type_non_contrib IS NULL OR btrim(r.icms_st_add_type_non_contrib) = '' THEN NULL
         WHEN upper(r.icms_st_add_type_non_contrib) LIKE 'F%' THEN 'FUNDO_COMBATE_POBREZA' ELSE 'OUTROS' END::icms_acres_type_enum,
    COALESCE(r.fcp_partition_pct, 0), COALESCE(r.difal_icms_red_pct, 0),
    CASE WHEN upper(btrim(r.difal_icms_type)) IN ('TRIBUTADO', 'ISENTO_OUTRAS', 'NAO_CONSIDERA')
         THEN upper(btrim(r.difal_icms_type))::icms_difal_type_enum END,
    COALESCE(r.difal_purchase_red_pct, 0)
  FROM icms_reduction_substitutions r
  LEFT JOIN items i     ON i.id = r.item_id     AND i.enterprise_id = r.enterprise_id
  LEFT JOIN customers c ON c.id = r.customer_id AND c.enterprise_id = r.enterprise_id
  LEFT JOIN suppliers s ON s.id = r.supplier_id AND s.enterprise_id = r.enterprise_id
 WHERE (r.item_id IS NULL OR i.code IS NOT NULL)
   AND (r.customer_id IS NULL OR c.code IS NOT NULL)
   AND (r.supplier_id IS NULL OR s.code IS NOT NULL)
   AND (r.item_id IS NOT NULL OR NULLIF(btrim(r.ncm_code), '') IS NOT NULL)
   AND (r.market_segment_id IS NULL OR EXISTS (SELECT 1 FROM market_segments m WHERE m.id = r.market_segment_id AND m.enterprise_id = r.enterprise_id))
   AND (r.invoice_type_out_id IS NULL OR EXISTS (SELECT 1 FROM invoice_types t WHERE t.id = r.invoice_type_out_id AND t.enterprise_id = r.enterprise_id))
   AND (r.invoice_type_in_id IS NULL OR EXISTS (SELECT 1 FROM invoice_types t WHERE t.id = r.invoice_type_in_id AND t.enterprise_id = r.enterprise_id))
   AND NOT EXISTS (SELECT 1 FROM icms_ipi_tax_params p WHERE p.origem_reducao_id = r.id);

-- Regras que não puderam ser copiadas ficam visíveis no log da migração.
DO $$
DECLARE n bigint;
BEGIN
    SELECT count(*) INTO n FROM icms_reduction_substitutions r
     WHERE NOT EXISTS (SELECT 1 FROM icms_ipi_tax_params p WHERE p.origem_reducao_id = r.id);
    IF n > 0 THEN
        RAISE NOTICE '000433: % regra(s) da VFIS0330 não copiada(s) — vínculo de outra empresa ou sem item/NCM; ficam em icms_reduction_substitutions', n;
    END IF;
END $$;
