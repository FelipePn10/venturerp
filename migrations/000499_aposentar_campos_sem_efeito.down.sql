-- Devolve as colunas aposentadas e o que estava gravado nelas.
CREATE TABLE special_adjustment_note_items (
    id bigserial PRIMARY KEY,
    note_id bigint NOT NULL REFERENCES special_adjustment_notes(id) ON DELETE CASCADE,
    sequence integer NOT NULL,
    item_id bigint,
    item_code character varying(60),
    description text,
    quantity numeric(15,4) DEFAULT 0 NOT NULL,
    unit character varying(6),
    unit_value numeric(15,4) DEFAULT 0 NOT NULL,
    total_value numeric(15,2) DEFAULT 0 NOT NULL,
    icms_base numeric(15,2) DEFAULT 0 NOT NULL,
    icms_pct numeric(7,4) DEFAULT 0 NOT NULL,
    icms_deferral_pct numeric(7,4) DEFAULT 0 NOT NULL,
    icms_value numeric(15,2) DEFAULT 0 NOT NULL,
    icms_deferred_value numeric(15,2) DEFAULT 0 NOT NULL,
    ipi_base numeric(15,2) DEFAULT 0 NOT NULL,
    ipi_pct numeric(7,4) DEFAULT 0 NOT NULL,
    ipi_value numeric(15,2) DEFAULT 0 NOT NULL,
    cst_icms character varying(3),
    cst_ipi character varying(3),
    cfop_id bigint REFERENCES cfops(id),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    UNIQUE (note_id, sequence)
);
INSERT INTO special_adjustment_note_items SELECT * FROM special_adjustment_note_items_retirados;
SELECT setval(pg_get_serial_sequence('special_adjustment_note_items', 'id'), GREATEST(COALESCE(MAX(id), 0), 1)) FROM special_adjustment_note_items;
DROP TABLE special_adjustment_note_items_retirados;

ALTER TABLE icms_ipi_tax_params
    ADD COLUMN bc_icms_st_modality character varying(5),
    ADD COLUMN calc_base_red_fci boolean DEFAULT false NOT NULL,
    ADD COLUMN cst_icms_contrib_dev character varying(3),
    ADD COLUMN cst_icms_fci character varying(3),
    ADD COLUMN cst_icms_non_contrib_dev character varying(3),
    ADD COLUMN cst_ipi_entry character varying(2),
    ADD COLUMN cst_situation_b character varying(2),
    ADD COLUMN dif_aliq_st_contrib_uc numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN difal_purchase_red_pct numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN difal_purchase_red_target icms_reduction_target_enum,
    ADD COLUMN fcp_st_partilha_pct numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN icms_acres_pct_contrib numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN icms_acres_sum_contrib boolean DEFAULT false NOT NULL,
    ADD COLUMN icms_acres_sum_non_contrib boolean DEFAULT false NOT NULL,
    ADD COLUMN icms_acres_type_contrib icms_acres_type_enum,
    ADD COLUMN icms_deferral_target icms_reduction_target_enum,
    ADD COLUMN icms_internal_pct numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN icms_st_acres_pct_contrib numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN icms_st_acres_pct_non_contrib numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN icms_st_acres_sum_contrib boolean DEFAULT false NOT NULL,
    ADD COLUMN icms_st_acres_sum_non_contrib boolean DEFAULT false NOT NULL,
    ADD COLUMN icms_st_acres_type_contrib icms_acres_type_enum,
    ADD COLUMN icms_st_acres_type_non_contrib icms_acres_type_enum,
    ADD COLUMN icms_subst_pct_contrib numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN icms_subst_pct_contrib_uc numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN icms_subst_pct_non_contrib numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN icms_subst_pct_origins_1238 numeric(7,4) DEFAULT 0 NOT NULL,
    ADD COLUMN legal_device_icms_subst_contrib_id bigint,
    ADD COLUMN legal_device_icms_subst_non_contrib_id bigint,
    ADD COLUMN legal_device_icms_subst_red_id bigint;
UPDATE icms_ipi_tax_params x SET bc_icms_st_modality = a.bc_icms_st_modality, calc_base_red_fci = a.calc_base_red_fci, cst_icms_contrib_dev = a.cst_icms_contrib_dev, cst_icms_fci = a.cst_icms_fci, cst_icms_non_contrib_dev = a.cst_icms_non_contrib_dev, cst_ipi_entry = a.cst_ipi_entry, cst_situation_b = a.cst_situation_b, dif_aliq_st_contrib_uc = a.dif_aliq_st_contrib_uc, difal_purchase_red_pct = a.difal_purchase_red_pct, difal_purchase_red_target = a.difal_purchase_red_target, fcp_st_partilha_pct = a.fcp_st_partilha_pct, icms_acres_pct_contrib = a.icms_acres_pct_contrib, icms_acres_sum_contrib = a.icms_acres_sum_contrib, icms_acres_sum_non_contrib = a.icms_acres_sum_non_contrib, icms_acres_type_contrib = a.icms_acres_type_contrib, icms_deferral_target = a.icms_deferral_target, icms_internal_pct = a.icms_internal_pct, icms_st_acres_pct_contrib = a.icms_st_acres_pct_contrib, icms_st_acres_pct_non_contrib = a.icms_st_acres_pct_non_contrib, icms_st_acres_sum_contrib = a.icms_st_acres_sum_contrib, icms_st_acres_sum_non_contrib = a.icms_st_acres_sum_non_contrib, icms_st_acres_type_contrib = a.icms_st_acres_type_contrib, icms_st_acres_type_non_contrib = a.icms_st_acres_type_non_contrib, icms_subst_pct_contrib = a.icms_subst_pct_contrib, icms_subst_pct_contrib_uc = a.icms_subst_pct_contrib_uc, icms_subst_pct_non_contrib = a.icms_subst_pct_non_contrib, icms_subst_pct_origins_1238 = a.icms_subst_pct_origins_1238, legal_device_icms_subst_contrib_id = a.legal_device_icms_subst_contrib_id, legal_device_icms_subst_non_contrib_id = a.legal_device_icms_subst_non_contrib_id, legal_device_icms_subst_red_id = a.legal_device_icms_subst_red_id FROM icms_ipi_tax_params_campos_retirados a WHERE a.id = x.id;
DROP TABLE icms_ipi_tax_params_campos_retirados;

ALTER TABLE cfops
    ADD COLUMN difal boolean DEFAULT false NOT NULL,
    ADD COLUMN doacao boolean DEFAULT false NOT NULL,
    ADD COLUMN ind_operacao cfop_ind_operacao_enum DEFAULT 'NORMAL'::cfop_ind_operacao_enum NOT NULL,
    ADD COLUMN tipo_utilizacao cfop_tipo_utilizacao_enum DEFAULT 'NORMAL'::cfop_tipo_utilizacao_enum NOT NULL,
    ADD COLUMN utilization cfop_utilization_enum DEFAULT 'INDUSTRIALIZACAO_COMERCIO'::cfop_utilization_enum NOT NULL;
UPDATE cfops x SET difal = a.difal, doacao = a.doacao, ind_operacao = a.ind_operacao, tipo_utilizacao = a.tipo_utilizacao, utilization = a.utilization FROM cfops_campos_retirados a WHERE a.id = x.id;
DROP TABLE cfops_campos_retirados;

ALTER TABLE fiscal_classifications
    ADD COLUMN apuracao character varying(20);
UPDATE fiscal_classifications x SET apuracao = a.apuracao FROM fiscal_classifications_campos_retirados a WHERE a.id = x.id;
DROP TABLE fiscal_classifications_campos_retirados;

ALTER TABLE entry_operation_types
    ADD COLUMN classification_code character varying(20),
    ADD COLUMN classification_type character varying(30),
    ADD COLUMN supplier_type_code bigint;
UPDATE entry_operation_types x SET classification_code = a.classification_code, classification_type = a.classification_type, supplier_type_code = a.supplier_type_code FROM entry_operation_types_campos_retirados a WHERE a.id = x.id;
DROP TABLE entry_operation_types_campos_retirados;

ALTER TABLE special_adjustment_notes
    ADD COLUMN auto_generate_summary boolean DEFAULT false NOT NULL,
    ADD COLUMN cfop_id bigint,
    ADD COLUMN total_value numeric(15,2) DEFAULT 0 NOT NULL;
UPDATE special_adjustment_notes x SET auto_generate_summary = a.auto_generate_summary, cfop_id = a.cfop_id, total_value = a.total_value FROM special_adjustment_notes_campos_retirados a WHERE a.id = x.id;
DROP TABLE special_adjustment_notes_campos_retirados;

ALTER TABLE invoice_types
    ADD COLUMN icms_pct numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN ipi_pct numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN pis_pct numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN cofins_pct numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN issqn_pct numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN ir_pct numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN csll_pct numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN inss_pct numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN considers_goals boolean DEFAULT false NOT NULL,
    ADD COLUMN lists_fiscal_books boolean DEFAULT true NOT NULL,
    ADD COLUMN gera_titulo_dev boolean DEFAULT false NOT NULL,
    ADD COLUMN model_nf character varying(2),
    ADD COLUMN cst_icms character varying(3),
    ADD COLUMN csosn_icms character varying(4),
    ADD COLUMN cst_ipi character varying(2),
    ADD COLUMN cst_pis character varying(2),
    ADD COLUMN cst_cofins character varying(2),
    ADD COLUMN ir_pct_presumption numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN csll_pct_presumption numeric(5,2) DEFAULT 0 NOT NULL,
    ADD COLUMN description_nf character varying(200),
    ADD COLUMN impostos_nfe impostos_nfe_enum,
    ADD COLUMN lista_valor_contabil boolean DEFAULT false NOT NULL,
    ADD COLUMN lista_registro_saida boolean DEFAULT false NOT NULL,
    ADD COLUMN lista_icms_ipi boolean DEFAULT false NOT NULL,
    ADD COLUMN sintegra_sped_fiscal boolean DEFAULT true NOT NULL,
    ADD COLUMN sisdeclara boolean DEFAULT false NOT NULL,
    ADD COLUMN calc_reducao boolean DEFAULT false NOT NULL,
    ADD COLUMN calc_imp_ibpt boolean DEFAULT false NOT NULL,
    ADD COLUMN cred_presumido_icms boolean DEFAULT false NOT NULL,
    ADD COLUMN desc_icms_licitacoes boolean DEFAULT false NOT NULL,
    ADD COLUMN vlr_agregado_base_subst boolean DEFAULT false NOT NULL,
    ADD COLUMN icms_st_ult_entrada boolean DEFAULT false NOT NULL,
    ADD COLUMN comp_ress_ret_st boolean DEFAULT false NOT NULL,
    ADD COLUMN ciap boolean DEFAULT false NOT NULL,
    ADD COLUMN calc_fomentar boolean DEFAULT false NOT NULL,
    ADD COLUMN excecao_fomentar boolean DEFAULT false NOT NULL,
    ADD COLUMN busca_tipo_nf boolean DEFAULT false NOT NULL,
    ADD COLUMN complemento_itens boolean DEFAULT false NOT NULL,
    ADD COLUMN somente_consulta_lotes boolean DEFAULT false NOT NULL,
    ADD COLUMN contrato_facon boolean DEFAULT false NOT NULL,
    ADD COLUMN ipi_transfer_sales_table_id bigint,
    ADD COLUMN dispositivo_legal_icms_id bigint,
    ADD COLUMN hierarchy_icms character varying(300),
    ADD COLUMN dispositivo_legal_icms_st_id bigint,
    ADD COLUMN hierarchy_icms_st character varying(300),
    ADD COLUMN dispositivo_legal_ipi_id bigint,
    ADD COLUMN hierarchy_ipi character varying(300),
    ADD COLUMN dispositivo_legal_pis_id bigint,
    ADD COLUMN hierarchy_pis character varying(300),
    ADD COLUMN dispositivo_legal_cofins_id bigint,
    ADD COLUMN hierarchy_cofins character varying(300),
    ADD COLUMN cod_beneficio_fiscal character varying(20),
    ADD COLUMN cod_motivo_rest_comp_icms_st character varying(10),
    ADD COLUMN cfop_id bigint,
    ADD COLUMN cod_clas_trib character varying(10),
    ADD COLUMN cod_clas_trib_trib_reg character varying(10);
UPDATE invoice_types x SET icms_pct = a.icms_pct, ipi_pct = a.ipi_pct, pis_pct = a.pis_pct, cofins_pct = a.cofins_pct, issqn_pct = a.issqn_pct, ir_pct = a.ir_pct, csll_pct = a.csll_pct, inss_pct = a.inss_pct, considers_goals = a.considers_goals, lists_fiscal_books = a.lists_fiscal_books, gera_titulo_dev = a.gera_titulo_dev, model_nf = a.model_nf, cst_icms = a.cst_icms, csosn_icms = a.csosn_icms, cst_ipi = a.cst_ipi, cst_pis = a.cst_pis, cst_cofins = a.cst_cofins, ir_pct_presumption = a.ir_pct_presumption, csll_pct_presumption = a.csll_pct_presumption, description_nf = a.description_nf, impostos_nfe = a.impostos_nfe, lista_valor_contabil = a.lista_valor_contabil, lista_registro_saida = a.lista_registro_saida, lista_icms_ipi = a.lista_icms_ipi, sintegra_sped_fiscal = a.sintegra_sped_fiscal, sisdeclara = a.sisdeclara, calc_reducao = a.calc_reducao, calc_imp_ibpt = a.calc_imp_ibpt, cred_presumido_icms = a.cred_presumido_icms, desc_icms_licitacoes = a.desc_icms_licitacoes, vlr_agregado_base_subst = a.vlr_agregado_base_subst, icms_st_ult_entrada = a.icms_st_ult_entrada, comp_ress_ret_st = a.comp_ress_ret_st, ciap = a.ciap, calc_fomentar = a.calc_fomentar, excecao_fomentar = a.excecao_fomentar, busca_tipo_nf = a.busca_tipo_nf, complemento_itens = a.complemento_itens, somente_consulta_lotes = a.somente_consulta_lotes, contrato_facon = a.contrato_facon, ipi_transfer_sales_table_id = a.ipi_transfer_sales_table_id, dispositivo_legal_icms_id = a.dispositivo_legal_icms_id, hierarchy_icms = a.hierarchy_icms, dispositivo_legal_icms_st_id = a.dispositivo_legal_icms_st_id, hierarchy_icms_st = a.hierarchy_icms_st, dispositivo_legal_ipi_id = a.dispositivo_legal_ipi_id, hierarchy_ipi = a.hierarchy_ipi, dispositivo_legal_pis_id = a.dispositivo_legal_pis_id, hierarchy_pis = a.hierarchy_pis, dispositivo_legal_cofins_id = a.dispositivo_legal_cofins_id, hierarchy_cofins = a.hierarchy_cofins, cod_beneficio_fiscal = a.cod_beneficio_fiscal, cod_motivo_rest_comp_icms_st = a.cod_motivo_rest_comp_icms_st, cfop_id = a.cfop_id, cod_clas_trib = a.cod_clas_trib, cod_clas_trib_trib_reg = a.cod_clas_trib_trib_reg FROM invoice_types_campos_retirados a WHERE a.id = x.id;
DROP TABLE invoice_types_campos_retirados;

ALTER TABLE carriers
    ADD COLUMN billing_type carrier_billing_type_enum DEFAULT 'CARTEIRA'::carrier_billing_type_enum NOT NULL,
    ADD COLUMN uses_credit_limit boolean DEFAULT false NOT NULL,
    ADD COLUMN consider_available boolean DEFAULT false NOT NULL,
    ADD COLUMN postpone_due_date boolean DEFAULT false NOT NULL,
    ADD COLUMN receipt_days smallint DEFAULT 0 NOT NULL,
    ADD COLUMN payment_days smallint DEFAULT 0 NOT NULL;
UPDATE carriers x SET billing_type = a.billing_type, uses_credit_limit = a.uses_credit_limit, consider_available = a.consider_available, postpone_due_date = a.postpone_due_date, receipt_days = a.receipt_days, payment_days = a.payment_days FROM carriers_campos_retirados a WHERE a.id = x.id;
DROP TABLE carriers_campos_retirados;

ALTER TABLE tax_types
    ADD COLUMN ipi_base_total_items boolean DEFAULT true NOT NULL,
    ADD COLUMN ipi_base_subtract_discount boolean DEFAULT false NOT NULL,
    ADD COLUMN ipi_base_add_freight boolean DEFAULT false NOT NULL,
    ADD COLUMN ipi_base_add_expenses boolean DEFAULT false NOT NULL,
    ADD COLUMN icms_base_total_items boolean DEFAULT true NOT NULL,
    ADD COLUMN icms_base_subtract_discount boolean DEFAULT true NOT NULL,
    ADD COLUMN icms_base_add_freight boolean DEFAULT true NOT NULL,
    ADD COLUMN icms_base_add_ipi boolean DEFAULT false NOT NULL,
    ADD COLUMN icms_base_add_expenses boolean DEFAULT false NOT NULL,
    ADD COLUMN pis_cofins_base_total_items boolean DEFAULT true NOT NULL,
    ADD COLUMN pis_cofins_base_subtract_discount boolean DEFAULT true NOT NULL,
    ADD COLUMN pis_cofins_base_add_freight boolean DEFAULT false NOT NULL,
    ADD COLUMN pis_cofins_base_add_insurance boolean DEFAULT false NOT NULL,
    ADD COLUMN pis_cofins_base_add_expenses boolean DEFAULT false NOT NULL,
    ADD COLUMN csll_base_total_items boolean DEFAULT true NOT NULL,
    ADD COLUMN csll_base_subtract_discount boolean DEFAULT true NOT NULL,
    ADD COLUMN csll_base_add_freight boolean DEFAULT false NOT NULL,
    ADD COLUMN ir_base_total_items boolean DEFAULT true NOT NULL,
    ADD COLUMN ir_base_subtract_discount boolean DEFAULT true NOT NULL,
    ADD COLUMN ir_base_add_freight boolean DEFAULT false NOT NULL;
UPDATE tax_types x SET ipi_base_total_items = a.ipi_base_total_items, ipi_base_subtract_discount = a.ipi_base_subtract_discount, ipi_base_add_freight = a.ipi_base_add_freight, ipi_base_add_expenses = a.ipi_base_add_expenses, icms_base_total_items = a.icms_base_total_items, icms_base_subtract_discount = a.icms_base_subtract_discount, icms_base_add_freight = a.icms_base_add_freight, icms_base_add_ipi = a.icms_base_add_ipi, icms_base_add_expenses = a.icms_base_add_expenses, pis_cofins_base_total_items = a.pis_cofins_base_total_items, pis_cofins_base_subtract_discount = a.pis_cofins_base_subtract_discount, pis_cofins_base_add_freight = a.pis_cofins_base_add_freight, pis_cofins_base_add_insurance = a.pis_cofins_base_add_insurance, pis_cofins_base_add_expenses = a.pis_cofins_base_add_expenses, csll_base_total_items = a.csll_base_total_items, csll_base_subtract_discount = a.csll_base_subtract_discount, csll_base_add_freight = a.csll_base_add_freight, ir_base_total_items = a.ir_base_total_items, ir_base_subtract_discount = a.ir_base_subtract_discount, ir_base_add_freight = a.ir_base_add_freight FROM tax_types_campos_retirados a WHERE a.id = x.id;
DROP TABLE tax_types_campos_retirados;

ALTER TABLE icms_ipi_tax_params ADD CONSTRAINT icms_ipi_tax_params_legal_device_icms_subst_contrib_id_fkey FOREIGN KEY (legal_device_icms_subst_contrib_id) REFERENCES legal_devices(id) ON DELETE SET NULL;
ALTER TABLE icms_ipi_tax_params ADD CONSTRAINT icms_ipi_tax_params_legal_device_icms_subst_non_contrib_id_fkey FOREIGN KEY (legal_device_icms_subst_non_contrib_id) REFERENCES legal_devices(id) ON DELETE SET NULL;
ALTER TABLE icms_ipi_tax_params ADD CONSTRAINT icms_ipi_tax_params_legal_device_icms_subst_red_id_fkey FOREIGN KEY (legal_device_icms_subst_red_id) REFERENCES legal_devices(id) ON DELETE SET NULL;
ALTER TABLE special_adjustment_notes ADD CONSTRAINT special_adjustment_notes_cfop_id_fkey FOREIGN KEY (cfop_id) REFERENCES cfops(id);
ALTER TABLE invoice_types ADD CONSTRAINT invoice_types_cfop_id_fkey FOREIGN KEY (cfop_id) REFERENCES cfops(id);
ALTER TABLE invoice_types ADD CONSTRAINT invoice_types_dispositivo_legal_cofins_id_fkey FOREIGN KEY (dispositivo_legal_cofins_id) REFERENCES legal_devices(id);
ALTER TABLE invoice_types ADD CONSTRAINT invoice_types_dispositivo_legal_icms_id_fkey FOREIGN KEY (dispositivo_legal_icms_id) REFERENCES legal_devices(id);
ALTER TABLE invoice_types ADD CONSTRAINT invoice_types_dispositivo_legal_icms_st_id_fkey FOREIGN KEY (dispositivo_legal_icms_st_id) REFERENCES legal_devices(id);
ALTER TABLE invoice_types ADD CONSTRAINT invoice_types_dispositivo_legal_ipi_id_fkey FOREIGN KEY (dispositivo_legal_ipi_id) REFERENCES legal_devices(id);
ALTER TABLE invoice_types ADD CONSTRAINT invoice_types_dispositivo_legal_pis_id_fkey FOREIGN KEY (dispositivo_legal_pis_id) REFERENCES legal_devices(id);
ALTER TABLE invoice_types ADD CONSTRAINT invoice_types_ipi_transfer_sales_table_id_empresa_fkey FOREIGN KEY (enterprise_id, ipi_transfer_sales_table_id) REFERENCES sales_tables(enterprise_id, id);
ALTER TABLE invoice_types ADD CONSTRAINT invoice_types_ipi_transfer_sales_table_id_fkey FOREIGN KEY (ipi_transfer_sales_table_id) REFERENCES sales_tables(id);

SELECT instalar_conferencia_de_empresa();
