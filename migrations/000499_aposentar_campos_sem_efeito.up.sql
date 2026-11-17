-- Aposenta as colunas que nenhuma regra lê (a tela já não as mostrava desde a
-- v1.19.0). Conferido em 17/11/2026: nas bases de produção nenhuma delas tem
-- valor preenchido. Mesmo assim o que estiver gravado vai para
-- <tabela>_campos_retirados (o down devolve). Ficam as que o motor fiscal lê:
-- supplier_code, item_config_mask, customer_establishment_code e
-- invoice_type_entry_id (impedem regra de entrada de valer na venda) e
-- invoice_types.exige_suframa (conferida no faturamento).

CREATE TABLE icms_ipi_tax_params_campos_retirados AS SELECT id, bc_icms_st_modality, calc_base_red_fci, cst_icms_contrib_dev, cst_icms_fci, cst_icms_non_contrib_dev, cst_ipi_entry, cst_situation_b, dif_aliq_st_contrib_uc, difal_purchase_red_pct, difal_purchase_red_target, fcp_st_partilha_pct, icms_acres_pct_contrib, icms_acres_sum_contrib, icms_acres_sum_non_contrib, icms_acres_type_contrib, icms_deferral_target, icms_internal_pct, icms_st_acres_pct_contrib, icms_st_acres_pct_non_contrib, icms_st_acres_sum_contrib, icms_st_acres_sum_non_contrib, icms_st_acres_type_contrib, icms_st_acres_type_non_contrib, icms_subst_pct_contrib, icms_subst_pct_contrib_uc, icms_subst_pct_non_contrib, icms_subst_pct_origins_1238, legal_device_icms_subst_contrib_id, legal_device_icms_subst_non_contrib_id, legal_device_icms_subst_red_id FROM icms_ipi_tax_params;
ALTER TABLE icms_ipi_tax_params_campos_retirados ADD PRIMARY KEY (id);
ALTER TABLE icms_ipi_tax_params
    DROP COLUMN bc_icms_st_modality,
    DROP COLUMN calc_base_red_fci,
    DROP COLUMN cst_icms_contrib_dev,
    DROP COLUMN cst_icms_fci,
    DROP COLUMN cst_icms_non_contrib_dev,
    DROP COLUMN cst_ipi_entry,
    DROP COLUMN cst_situation_b,
    DROP COLUMN dif_aliq_st_contrib_uc,
    DROP COLUMN difal_purchase_red_pct,
    DROP COLUMN difal_purchase_red_target,
    DROP COLUMN fcp_st_partilha_pct,
    DROP COLUMN icms_acres_pct_contrib,
    DROP COLUMN icms_acres_sum_contrib,
    DROP COLUMN icms_acres_sum_non_contrib,
    DROP COLUMN icms_acres_type_contrib,
    DROP COLUMN icms_deferral_target,
    DROP COLUMN icms_internal_pct,
    DROP COLUMN icms_st_acres_pct_contrib,
    DROP COLUMN icms_st_acres_pct_non_contrib,
    DROP COLUMN icms_st_acres_sum_contrib,
    DROP COLUMN icms_st_acres_sum_non_contrib,
    DROP COLUMN icms_st_acres_type_contrib,
    DROP COLUMN icms_st_acres_type_non_contrib,
    DROP COLUMN icms_subst_pct_contrib,
    DROP COLUMN icms_subst_pct_contrib_uc,
    DROP COLUMN icms_subst_pct_non_contrib,
    DROP COLUMN icms_subst_pct_origins_1238,
    DROP COLUMN legal_device_icms_subst_contrib_id,
    DROP COLUMN legal_device_icms_subst_non_contrib_id,
    DROP COLUMN legal_device_icms_subst_red_id;

CREATE TABLE cfops_campos_retirados AS SELECT id, difal, doacao, ind_operacao, tipo_utilizacao, utilization FROM cfops;
ALTER TABLE cfops_campos_retirados ADD PRIMARY KEY (id);
ALTER TABLE cfops
    DROP COLUMN difal,
    DROP COLUMN doacao,
    DROP COLUMN ind_operacao,
    DROP COLUMN tipo_utilizacao,
    DROP COLUMN utilization;

CREATE TABLE fiscal_classifications_campos_retirados AS SELECT id, apuracao FROM fiscal_classifications;
ALTER TABLE fiscal_classifications_campos_retirados ADD PRIMARY KEY (id);
ALTER TABLE fiscal_classifications
    DROP COLUMN apuracao;

CREATE TABLE entry_operation_types_campos_retirados AS SELECT id, classification_code, classification_type, supplier_type_code FROM entry_operation_types;
ALTER TABLE entry_operation_types_campos_retirados ADD PRIMARY KEY (id);
ALTER TABLE entry_operation_types
    DROP COLUMN classification_code,
    DROP COLUMN classification_type,
    DROP COLUMN supplier_type_code;

CREATE TABLE special_adjustment_notes_campos_retirados AS SELECT id, auto_generate_summary, cfop_id, total_value FROM special_adjustment_notes;
ALTER TABLE special_adjustment_notes_campos_retirados ADD PRIMARY KEY (id);
ALTER TABLE special_adjustment_notes
    DROP COLUMN auto_generate_summary,
    DROP COLUMN cfop_id,
    DROP COLUMN total_value;

CREATE TABLE invoice_types_campos_retirados AS SELECT id, icms_pct, ipi_pct, pis_pct, cofins_pct, issqn_pct, ir_pct, csll_pct, inss_pct, considers_goals, lists_fiscal_books, gera_titulo_dev, model_nf, cst_icms, csosn_icms, cst_ipi, cst_pis, cst_cofins, ir_pct_presumption, csll_pct_presumption, description_nf, impostos_nfe, lista_valor_contabil, lista_registro_saida, lista_icms_ipi, sintegra_sped_fiscal, sisdeclara, calc_reducao, calc_imp_ibpt, cred_presumido_icms, desc_icms_licitacoes, vlr_agregado_base_subst, icms_st_ult_entrada, comp_ress_ret_st, ciap, calc_fomentar, excecao_fomentar, busca_tipo_nf, complemento_itens, somente_consulta_lotes, contrato_facon, ipi_transfer_sales_table_id, dispositivo_legal_icms_id, hierarchy_icms, dispositivo_legal_icms_st_id, hierarchy_icms_st, dispositivo_legal_ipi_id, hierarchy_ipi, dispositivo_legal_pis_id, hierarchy_pis, dispositivo_legal_cofins_id, hierarchy_cofins, cod_beneficio_fiscal, cod_motivo_rest_comp_icms_st, cfop_id, cod_clas_trib, cod_clas_trib_trib_reg FROM invoice_types;
ALTER TABLE invoice_types_campos_retirados ADD PRIMARY KEY (id);
ALTER TABLE invoice_types
    DROP COLUMN icms_pct,
    DROP COLUMN ipi_pct,
    DROP COLUMN pis_pct,
    DROP COLUMN cofins_pct,
    DROP COLUMN issqn_pct,
    DROP COLUMN ir_pct,
    DROP COLUMN csll_pct,
    DROP COLUMN inss_pct,
    DROP COLUMN considers_goals,
    DROP COLUMN lists_fiscal_books,
    DROP COLUMN gera_titulo_dev,
    DROP COLUMN model_nf,
    DROP COLUMN cst_icms,
    DROP COLUMN csosn_icms,
    DROP COLUMN cst_ipi,
    DROP COLUMN cst_pis,
    DROP COLUMN cst_cofins,
    DROP COLUMN ir_pct_presumption,
    DROP COLUMN csll_pct_presumption,
    DROP COLUMN description_nf,
    DROP COLUMN impostos_nfe,
    DROP COLUMN lista_valor_contabil,
    DROP COLUMN lista_registro_saida,
    DROP COLUMN lista_icms_ipi,
    DROP COLUMN sintegra_sped_fiscal,
    DROP COLUMN sisdeclara,
    DROP COLUMN calc_reducao,
    DROP COLUMN calc_imp_ibpt,
    DROP COLUMN cred_presumido_icms,
    DROP COLUMN desc_icms_licitacoes,
    DROP COLUMN vlr_agregado_base_subst,
    DROP COLUMN icms_st_ult_entrada,
    DROP COLUMN comp_ress_ret_st,
    DROP COLUMN ciap,
    DROP COLUMN calc_fomentar,
    DROP COLUMN excecao_fomentar,
    DROP COLUMN busca_tipo_nf,
    DROP COLUMN complemento_itens,
    DROP COLUMN somente_consulta_lotes,
    DROP COLUMN contrato_facon,
    DROP COLUMN ipi_transfer_sales_table_id,
    DROP COLUMN dispositivo_legal_icms_id,
    DROP COLUMN hierarchy_icms,
    DROP COLUMN dispositivo_legal_icms_st_id,
    DROP COLUMN hierarchy_icms_st,
    DROP COLUMN dispositivo_legal_ipi_id,
    DROP COLUMN hierarchy_ipi,
    DROP COLUMN dispositivo_legal_pis_id,
    DROP COLUMN hierarchy_pis,
    DROP COLUMN dispositivo_legal_cofins_id,
    DROP COLUMN hierarchy_cofins,
    DROP COLUMN cod_beneficio_fiscal,
    DROP COLUMN cod_motivo_rest_comp_icms_st,
    DROP COLUMN cfop_id,
    DROP COLUMN cod_clas_trib,
    DROP COLUMN cod_clas_trib_trib_reg;

CREATE TABLE carriers_campos_retirados AS SELECT id, billing_type, uses_credit_limit, consider_available, postpone_due_date, receipt_days, payment_days FROM carriers;
ALTER TABLE carriers_campos_retirados ADD PRIMARY KEY (id);
ALTER TABLE carriers
    DROP COLUMN billing_type,
    DROP COLUMN uses_credit_limit,
    DROP COLUMN consider_available,
    DROP COLUMN postpone_due_date,
    DROP COLUMN receipt_days,
    DROP COLUMN payment_days;

CREATE TABLE tax_types_campos_retirados AS SELECT id, ipi_base_total_items, ipi_base_subtract_discount, ipi_base_add_freight, ipi_base_add_expenses, icms_base_total_items, icms_base_subtract_discount, icms_base_add_freight, icms_base_add_ipi, icms_base_add_expenses, pis_cofins_base_total_items, pis_cofins_base_subtract_discount, pis_cofins_base_add_freight, pis_cofins_base_add_insurance, pis_cofins_base_add_expenses, csll_base_total_items, csll_base_subtract_discount, csll_base_add_freight, ir_base_total_items, ir_base_subtract_discount, ir_base_add_freight FROM tax_types;
ALTER TABLE tax_types_campos_retirados ADD PRIMARY KEY (id);
ALTER TABLE tax_types
    DROP COLUMN ipi_base_total_items,
    DROP COLUMN ipi_base_subtract_discount,
    DROP COLUMN ipi_base_add_freight,
    DROP COLUMN ipi_base_add_expenses,
    DROP COLUMN icms_base_total_items,
    DROP COLUMN icms_base_subtract_discount,
    DROP COLUMN icms_base_add_freight,
    DROP COLUMN icms_base_add_ipi,
    DROP COLUMN icms_base_add_expenses,
    DROP COLUMN pis_cofins_base_total_items,
    DROP COLUMN pis_cofins_base_subtract_discount,
    DROP COLUMN pis_cofins_base_add_freight,
    DROP COLUMN pis_cofins_base_add_insurance,
    DROP COLUMN pis_cofins_base_add_expenses,
    DROP COLUMN csll_base_total_items,
    DROP COLUMN csll_base_subtract_discount,
    DROP COLUMN csll_base_add_freight,
    DROP COLUMN ir_base_total_items,
    DROP COLUMN ir_base_subtract_discount,
    DROP COLUMN ir_base_add_freight;

-- Itens da nota especial de ajuste: o SPED (E111) leva só o valor da nota.
CREATE TABLE special_adjustment_note_items_retirados AS SELECT * FROM special_adjustment_note_items;
DROP TABLE special_adjustment_note_items;

SELECT instalar_conferencia_de_empresa();
