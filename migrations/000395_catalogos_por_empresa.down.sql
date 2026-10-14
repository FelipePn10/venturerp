-- Volta aos catálogos globais.
-- Família A: cada cópia (mesmo código, id maior) é fundida no registro de menor
-- id; as referências por id voltam para ele (pelas FKs e pelas colunas sem FK
-- conhecidas), as cópias são apagadas e a unicidade global é restaurada.
-- Família B: só perde a coluna; as cópias com código novo ficam como registros
-- independentes (podem estar em documentos).

-- gatilhos de integridade da subida
DROP TRIGGER IF EXISTS trg_customer_contacts_catalogo ON public.customer_contacts;
DROP TRIGGER IF EXISTS trg_supplier_contacts_catalogo ON public.supplier_contacts;
DROP TRIGGER IF EXISTS trg_payment_condition_installments_catalogo ON public.payment_condition_installments;
DROP TRIGGER IF EXISTS trg_supplier_enterprises_catalogo ON public.supplier_enterprises;
DROP TRIGGER IF EXISTS trg_supplier_parameters_catalogo ON public.supplier_parameters;
DROP TRIGGER IF EXISTS trg_sales_orders_catalogo ON public.sales_orders;
DROP TRIGGER IF EXISTS trg_representatives_catalogo ON public.representatives;
DROP TRIGGER IF EXISTS trg_consumer_service_calls_catalogo ON public.consumer_service_calls;
DROP TRIGGER IF EXISTS trg_consumer_service_consumers_catalogo ON public.consumer_service_consumers;
DROP TRIGGER IF EXISTS trg_technical_assistance_call_items_catalogo ON public.technical_assistance_call_items;
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_contato_cliente();
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_contato_fornecedor();
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_parcela();
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_fornecedor_empresa();
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_parametro_fornecedor();
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_pedido_venda();
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_representante();
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_atendimento_sac();
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_consumidor();
DROP FUNCTION IF EXISTS public.erp_confere_catalogo_item_assistencia();
DROP FUNCTION IF EXISTS public.erp_recusa_catalogo(TEXT);

-- FKs compostas e unicidades (empresa, id) da subida
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_region_id_empresa_fkey;
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_market_segment_id_empresa_fkey;
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_customer_type_id_empresa_fkey;
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_carrier_id_empresa_fkey;
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_carrier_group_id_empresa_fkey;
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_invoice_type_id_empresa_fkey;
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_tax_type_id_empresa_fkey;
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_payment_condition_id_empresa_fkey;
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_sales_table_id_empresa_fkey;
ALTER TABLE public.suppliers DROP CONSTRAINT IF EXISTS suppliers_region_id_empresa_fkey;
ALTER TABLE public.suppliers DROP CONSTRAINT IF EXISTS suppliers_carrier_id_empresa_fkey;
ALTER TABLE public.suppliers DROP CONSTRAINT IF EXISTS suppliers_supplier_type_id_empresa_fkey;
ALTER TABLE public.commercial_policies DROP CONSTRAINT IF EXISTS commercial_policies_region_id_empresa_fkey;
ALTER TABLE public.commercial_policies DROP CONSTRAINT IF EXISTS commercial_policies_market_segment_id_empresa_fkey;
ALTER TABLE public.commercial_policies DROP CONSTRAINT IF EXISTS commercial_policies_customer_type_id_empresa_fkey;
ALTER TABLE public.commercial_policies DROP CONSTRAINT IF EXISTS commercial_policies_carrier_id_empresa_fkey;
ALTER TABLE public.payment_conditions DROP CONSTRAINT IF EXISTS payment_conditions_carrier_id_empresa_fkey;
ALTER TABLE public.icms_ipi_tax_params DROP CONSTRAINT IF EXISTS icms_ipi_tax_params_market_segment_id_empresa_fkey;
ALTER TABLE public.icms_ipi_tax_params DROP CONSTRAINT IF EXISTS icms_ipi_tax_params_invoice_type_exit_id_empresa_fkey;
ALTER TABLE public.icms_ipi_tax_params DROP CONSTRAINT IF EXISTS icms_ipi_tax_params_invoice_type_entry_id_empresa_fkey;
ALTER TABLE public.icms_ipi_tax_params DROP CONSTRAINT IF EXISTS icms_ipi_tax_params_tax_type_id_empresa_fkey;
ALTER TABLE public.icms_reduction_substitutions DROP CONSTRAINT IF EXISTS icms_reduction_substitutions_market_segment_id_empresa_fkey;
ALTER TABLE public.icms_reduction_substitutions DROP CONSTRAINT IF EXISTS icms_reduction_substitutions_invoice_type_out_id_empresa_fkey;
ALTER TABLE public.icms_reduction_substitutions DROP CONSTRAINT IF EXISTS icms_reduction_substitutions_invoice_type_in_id_empresa_fkey;
ALTER TABLE public.market_segments DROP CONSTRAINT IF EXISTS market_segments_parent_id_empresa_fkey;
ALTER TABLE public.invoice_types DROP CONSTRAINT IF EXISTS invoice_types_ipi_transfer_sales_table_id_empresa_fkey;
ALTER TABLE public.special_adjustment_notes DROP CONSTRAINT IF EXISTS special_adjustment_notes_invoice_type_empresa_fkey;
ALTER TABLE public.technical_assistance_defect_reasons DROP CONSTRAINT IF EXISTS technical_assistance_defect_reasons_grupo_empresa_fkey;
ALTER TABLE public.technical_assistance_defect_groups DROP CONSTRAINT IF EXISTS technical_assistance_defect_groups_empresa_code_key;
ALTER TABLE public.regions DROP CONSTRAINT IF EXISTS regions_empresa_id_key;
ALTER TABLE public.market_segments DROP CONSTRAINT IF EXISTS market_segments_empresa_id_key;
ALTER TABLE public.customer_contact_types DROP CONSTRAINT IF EXISTS customer_contact_types_empresa_id_key;
ALTER TABLE public.customer_types DROP CONSTRAINT IF EXISTS customer_types_empresa_id_key;
ALTER TABLE public.carriers DROP CONSTRAINT IF EXISTS carriers_empresa_id_key;
ALTER TABLE public.carrier_groups DROP CONSTRAINT IF EXISTS carrier_groups_empresa_id_key;
ALTER TABLE public.invoice_types DROP CONSTRAINT IF EXISTS invoice_types_empresa_id_key;
ALTER TABLE public.tax_types DROP CONSTRAINT IF EXISTS tax_types_empresa_id_key;
ALTER TABLE public.supplier_types DROP CONSTRAINT IF EXISTS supplier_types_empresa_id_key;
ALTER TABLE public.supplier_contact_types DROP CONSTRAINT IF EXISTS supplier_contact_types_empresa_id_key;
ALTER TABLE public.entry_operation_types DROP CONSTRAINT IF EXISTS entry_operation_types_empresa_id_key;
ALTER TABLE public.state_groups DROP CONSTRAINT IF EXISTS state_groups_empresa_id_key;
ALTER TABLE public.stock_movement_types DROP CONSTRAINT IF EXISTS stock_movement_types_empresa_id_key;
ALTER TABLE public.payment_conditions DROP CONSTRAINT IF EXISTS payment_conditions_empresa_id_key;
ALTER TABLE public.sales_tables DROP CONSTRAINT IF EXISTS sales_tables_empresa_id_key;

CREATE FUNCTION pg_temp.volta_a(tab TEXT, chave TEXT, extras TEXT[]) RETURNS VOID LANGUAGE plpgsql AS $f$
DECLARE fk RECORD; i INT;
BEGIN
    EXECUTE format('CREATE TEMP TABLE %I ON COMMIT DROP AS
                    SELECT c.id AS copia, (SELECT MIN(o.id) FROM public.%I o WHERE o.%I = c.%I) AS original
                      FROM public.%I c', '_volta_' || tab, tab, chave, chave, tab);
    EXECUTE format('DELETE FROM %I WHERE copia = original', '_volta_' || tab);
    FOR fk IN SELECT conrelid::regclass AS ref, a.attname AS col
                FROM pg_constraint k
                JOIN pg_attribute a ON a.attrelid = k.conrelid AND a.attnum = k.conkey[1]
               WHERE k.contype = 'f' AND k.confrelid = ('public.' || tab)::regclass AND array_length(k.conkey, 1) = 1 LOOP
        EXECUTE format('UPDATE %s x SET %I = v.original FROM %I v WHERE v.copia = x.%I', fk.ref, fk.col, '_volta_' || tab, fk.col);
    END LOOP;
    FOR i IN 1 .. COALESCE(array_length(extras, 1), 0) LOOP
        EXECUTE format('UPDATE public.%s x SET %s = v.original FROM %I v WHERE v.copia = x.%s',
                       split_part(extras[i], '.', 1), split_part(extras[i], '.', 2), '_volta_' || tab, split_part(extras[i], '.', 2));
    END LOOP;
    EXECUTE format('DELETE FROM public.%I WHERE id IN (SELECT copia FROM %I)', tab, '_volta_' || tab);
    EXECUTE format('DROP INDEX IF EXISTS public.%I', 'ix_' || tab || '_enterprise');
    EXECUTE format('ALTER TABLE public.%I DROP CONSTRAINT IF EXISTS %I', tab, tab || '_empresa_' || chave || '_key');
    EXECUTE format('ALTER TABLE public.%I DROP COLUMN enterprise_id', tab);
    EXECUTE format('ALTER TABLE public.%I ADD CONSTRAINT %I UNIQUE (%I)', tab, tab || '_' || chave || '_key', chave);
END $f$;

-- grupos de UF: as UFs das cópias saem com elas
ALTER TABLE public.state_group_ufs DROP CONSTRAINT IF EXISTS state_group_ufs_grupo_fkey;
DELETE FROM public.state_group_ufs u
 USING public.state_groups g
 WHERE g.enterprise_id = u.enterprise_id AND g.code = u.state_group_code
   AND g.id <> (SELECT MIN(o.id) FROM public.state_groups o WHERE o.code = g.code);
ALTER TABLE public.state_group_ufs DROP CONSTRAINT IF EXISTS state_group_ufs_empresa_grupo_uf_key;
ALTER TABLE public.state_group_ufs DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE public.state_group_ufs ADD CONSTRAINT state_group_ufs_state_group_code_uf_key UNIQUE (state_group_code, uf);

-- (vínculos portador × grupo das cópias saem com elas: ON DELETE CASCADE)
ALTER TABLE public.customer_types DROP CONSTRAINT IF EXISTS customer_types_empresa_description_key;

SELECT pg_temp.volta_a('regions', 'code', ARRAY[]::TEXT[]);
SELECT pg_temp.volta_a('market_segments', 'code', ARRAY['icms_reduction_substitutions.market_segment_id']);
SELECT pg_temp.volta_a('customer_contact_types', 'code', ARRAY[]::TEXT[]);
SELECT pg_temp.volta_a('customer_types', 'code', ARRAY[]::TEXT[]);
ALTER TABLE public.customer_types ADD CONSTRAINT customer_types_description_key UNIQUE (description);
SELECT pg_temp.volta_a('carriers', 'code', ARRAY[]::TEXT[]);
SELECT pg_temp.volta_a('carrier_groups', 'code', ARRAY[]::TEXT[]);
SELECT pg_temp.volta_a('invoice_types', 'code', ARRAY['icms_reduction_substitutions.invoice_type_out_id', 'icms_reduction_substitutions.invoice_type_in_id']);
SELECT pg_temp.volta_a('tax_types', 'code', ARRAY[]::TEXT[]);
SELECT pg_temp.volta_a('supplier_types', 'code', ARRAY[]::TEXT[]);
SELECT pg_temp.volta_a('supplier_contact_types', 'code', ARRAY[]::TEXT[]);
SELECT pg_temp.volta_a('entry_operation_types', 'code', ARRAY[]::TEXT[]);
SELECT pg_temp.volta_a('state_groups', 'code', ARRAY[]::TEXT[]);
SELECT pg_temp.volta_a('stock_movement_types', 'sigla', ARRAY[]::TEXT[]);

ALTER TABLE public.state_group_ufs
    ADD CONSTRAINT state_group_ufs_state_group_code_fkey FOREIGN KEY (state_group_code) REFERENCES public.state_groups(code) ON DELETE CASCADE;
-- Nomes da 000120, que criou essas chaves.
ALTER TABLE public.sales_orders ADD CONSTRAINT fk_so_tax_type FOREIGN KEY (tax_type_code) REFERENCES public.tax_types(code);
ALTER TABLE public.sales_orders ADD CONSTRAINT fk_so_bearer FOREIGN KEY (bearer_code) REFERENCES public.carriers(code);

-- Família B: a cópia (código novo) é igual ao original em tudo menos código,
-- empresa e datas; é fundida no de menor código, e as referências voltam.
CREATE FUNCTION pg_temp.volta_b(tab TEXT, refs TEXT[]) RETURNS VOID LANGUAGE plpgsql AS $f$
DECLARE cond TEXT; i INT;
BEGIN
    SELECT string_agg(format('o.%1$I IS NOT DISTINCT FROM c.%1$I', column_name), ' AND ')
      INTO cond FROM information_schema.columns
     WHERE table_schema = 'public' AND table_name = tab
       AND column_name NOT IN ('code', 'enterprise_id', 'created_at', 'updated_at', 'group_code');
    IF tab = 'technical_assistance_defect_reasons' THEN
        -- o grupo do motivo já voltou ao original; compara pelo grupo original
        cond := cond || ' AND o.group_code = c.group_code';
    END IF;
    EXECUTE format('CREATE TEMP TABLE %I ON COMMIT DROP AS
                    SELECT c.code AS copia, (SELECT MIN(o.code) FROM public.%I o WHERE %s) AS original FROM public.%I c',
                   '_volta_' || tab, tab, cond, tab);
    EXECUTE format('DELETE FROM %I WHERE copia = original OR original IS NULL', '_volta_' || tab);
    FOR i IN 1 .. COALESCE(array_length(refs, 1), 0) LOOP
        EXECUTE format('UPDATE public.%s x SET %s = v.original FROM %I v WHERE v.copia = x.%s',
                       split_part(refs[i], '.', 1), split_part(refs[i], '.', 2), '_volta_' || tab, split_part(refs[i], '.', 2));
    END LOOP;
    EXECUTE format('DELETE FROM public.%I WHERE code IN (SELECT copia FROM %I)', tab, '_volta_' || tab);
    EXECUTE format('DROP INDEX IF EXISTS public.%I', 'ix_' || tab || '_enterprise');
    EXECUTE format('ALTER TABLE public.%I DROP COLUMN enterprise_id', tab);
END $f$;

SELECT pg_temp.volta_b('representative_types', ARRAY['representatives.type_code']);
SELECT pg_temp.volta_b('consumer_service_call_types', ARRAY['consumer_service_calls.call_type_code']);
SELECT pg_temp.volta_b('consumer_service_knowledge_sources', ARRAY['consumer_service_consumers.knowledge_code']);
SELECT pg_temp.volta_b('technical_assistance_defect_groups', ARRAY['technical_assistance_defect_reasons.group_code', 'consumer_service_calls.defect_group_code']);
SELECT pg_temp.volta_b('technical_assistance_defect_reasons', ARRAY['technical_assistance_call_items.defect_reason_code', 'consumer_service_calls.defect_reason_code']);
