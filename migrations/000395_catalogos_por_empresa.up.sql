-- Catálogos por empresa.
--
-- Os cadastros de apoio eram do banco inteiro: uma empresa via, alterava e
-- usava as regiões, segmentos, tipos, portadores, tipos de nota e de
-- tributação, TES, grupos de UF etc. da outra. Cada empresa passa a ter os
-- seus (enterprise_id).
--
-- Família A — catálogo com id e código: cada empresa recebe a sua cópia com o
-- MESMO código (o original fica com a menor empresa). O código passa a ser
-- único por empresa. Referências por CÓDIGO (TES na nota, tipo de nota no
-- pedido, grupo de UF na TES, portador e tributação no pedido de venda...)
-- continuam certas sem mudar nada; referências por ID são remapeadas para a
-- cópia da empresa do registro que referencia.
--
-- Família B — catálogo cujo código é a chave primária (tipos de
-- representante, tipos de chamado e fontes de conhecimento do SAC, grupos e
-- motivos de defeito): as empresas que não são a dona recebem cópia com código
-- novo, e os registros delas passam para a cópia.
--
-- Chaves estrangeiras por código global deixam de existir (o código agora se
-- repete entre empresas): sales_orders → carriers(code)/tax_types(code) e
-- state_group_ufs → state_groups(code). A conferência passa a ser feita pela
-- empresa nas consultas; state_group_ufs ganha a empresa e FK composta.

-- Colunas declaradas aqui também (e não só no SQL dinâmico abaixo) para o
-- esquema que o sqlc lê enxergá-las.
ALTER TABLE public.regions ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.market_segments ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.customer_contact_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.customer_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.carriers ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.carrier_groups ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.invoice_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.tax_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.supplier_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.supplier_contact_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.entry_operation_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.state_groups ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.stock_movement_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.representative_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.consumer_service_call_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.consumer_service_knowledge_sources ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.technical_assistance_defect_groups ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.technical_assistance_defect_reasons ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);

CREATE FUNCTION pg_temp.catalogo_a(tab TEXT, chaves TEXT[]) RETURNS VOID LANGUAGE plpgsql AS $f$
DECLARE cols TEXT; rcols TEXT; con RECORD;
BEGIN
    EXECUTE format('ALTER TABLE public.%I ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id)', tab);
    EXECUTE format('ALTER TABLE public.%I ADD COLUMN _origem_id BIGINT', tab);
    EXECUTE format('UPDATE public.%I SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL', tab);
    -- unicidade global vira por empresa
    FOR con IN SELECT conname, pg_get_constraintdef(oid) AS def FROM pg_constraint
                WHERE conrelid = ('public.' || tab)::regclass AND contype = 'u' LOOP
        EXECUTE format('ALTER TABLE public.%I DROP CONSTRAINT %I', tab, con.conname);
    END LOOP;
    SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position),
           string_agg('t.' || quote_ident(column_name), ',' ORDER BY ordinal_position)
      INTO cols, rcols
      FROM information_schema.columns
     WHERE table_schema = 'public' AND table_name = tab AND column_name NOT IN ('id', 'enterprise_id', '_origem_id');
    EXECUTE format('INSERT INTO public.%1$I (%2$s, enterprise_id, _origem_id)
                    SELECT %3$s, e.id, t.id FROM public.%1$I t CROSS JOIN public.enterprise e
                     WHERE e.id <> t.enterprise_id AND t._origem_id IS NULL', tab, cols, rcols);
    EXECUTE format('CREATE TEMP TABLE %I ON COMMIT DROP AS
                    SELECT id AS antigo, enterprise_id, id AS novo FROM public.%I WHERE _origem_id IS NULL
                    UNION ALL
                    SELECT _origem_id, enterprise_id, id FROM public.%I WHERE _origem_id IS NOT NULL',
                   '_mapa_' || tab, tab, tab);
    EXECUTE format('DELETE FROM public.%I WHERE enterprise_id IS NULL', tab); -- instalação sem empresa
    EXECUTE format('ALTER TABLE public.%I ALTER COLUMN enterprise_id SET NOT NULL', tab);
    FOR i IN 1 .. COALESCE(array_length(chaves, 1), 0) LOOP
        EXECUTE format('ALTER TABLE public.%I ADD CONSTRAINT %I UNIQUE (enterprise_id, %I)', tab, tab || '_empresa_' || chaves[i] || '_key', chaves[i]);
    END LOOP;
END $f$;

-- remap(tabela que referencia, coluna, catálogo, expressão da empresa do
-- registro, tabela extra para achar a empresa, condição de junção dela)
CREATE FUNCTION pg_temp.remap(ref TEXT, col TEXT, cat TEXT, empresa TEXT, juncao TEXT DEFAULT '', cond TEXT DEFAULT 'TRUE')
RETURNS VOID LANGUAGE plpgsql AS $f$
BEGIN
    EXECUTE format('UPDATE public.%1$I x SET %2$I = m.novo FROM %3$I m %5$s
                     WHERE m.antigo = x.%2$I AND m.enterprise_id = %4$s AND (%6$s) AND x.%2$I IS DISTINCT FROM m.novo',
                   ref, col, '_mapa_' || cat, empresa, juncao, cond);
END $f$;

-- ─── Família A ───────────────────────────────────────────────────────────────
ALTER TABLE public.sales_orders DROP CONSTRAINT IF EXISTS sales_orders_bearer_code_fkey;
ALTER TABLE public.sales_orders DROP CONSTRAINT IF EXISTS sales_orders_tax_type_code_fkey;
DO $$ DECLARE c RECORD; BEGIN
    FOR c IN SELECT conname, conrelid::regclass AS t FROM pg_constraint
              WHERE contype = 'f' AND confrelid IN ('public.carriers'::regclass, 'public.tax_types'::regclass, 'public.state_groups'::regclass)
                AND pg_get_constraintdef(oid) ~ 'REFERENCES (carriers|tax_types|state_groups)\(code\)' LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', c.t, c.conname);
    END LOOP;
END $$;

SELECT pg_temp.catalogo_a('regions', ARRAY['code']);
SELECT pg_temp.catalogo_a('market_segments', ARRAY['code']);
SELECT pg_temp.catalogo_a('customer_contact_types', ARRAY['code']);
SELECT pg_temp.catalogo_a('customer_types', ARRAY['code', 'description']);
SELECT pg_temp.catalogo_a('carriers', ARRAY['code']);
SELECT pg_temp.catalogo_a('carrier_groups', ARRAY['code']);
SELECT pg_temp.catalogo_a('invoice_types', ARRAY['code']);
SELECT pg_temp.catalogo_a('tax_types', ARRAY['code']);
SELECT pg_temp.catalogo_a('supplier_types', ARRAY['code']);
SELECT pg_temp.catalogo_a('supplier_contact_types', ARRAY['code']);
SELECT pg_temp.catalogo_a('entry_operation_types', ARRAY['code']);
SELECT pg_temp.catalogo_a('state_groups', ARRAY['code']);
SELECT pg_temp.catalogo_a('stock_movement_types', ARRAY['sigla']);

-- hierarquia de segmentos dentro da empresa
SELECT pg_temp.remap('market_segments', 'parent_id', 'market_segments', 'x.enterprise_id');

-- vínculos portador × grupo: um conjunto por empresa
INSERT INTO public.carrier_group_carriers (carrier_group_id, carrier_id)
SELECT mg.novo, mc.novo
  FROM public.carrier_group_carriers gc
  JOIN _mapa_carrier_groups mg ON mg.antigo = gc.carrier_group_id
  JOIN _mapa_carriers mc ON mc.antigo = gc.carrier_id AND mc.enterprise_id = mg.enterprise_id
 WHERE mg.novo <> mg.antigo
ON CONFLICT DO NOTHING;

-- grupos de UF: as UFs seguem o grupo (por código + empresa)
ALTER TABLE public.state_group_ufs DROP CONSTRAINT IF EXISTS state_group_ufs_state_group_code_uf_key;
ALTER TABLE public.state_group_ufs ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.state_group_ufs u SET enterprise_id = g.enterprise_id
  FROM public.state_groups g WHERE g.code = u.state_group_code AND g._origem_id IS NULL;
DO $$ DECLARE cols TEXT; rcols TEXT; BEGIN
    SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position),
           string_agg('u.' || quote_ident(column_name), ',' ORDER BY ordinal_position)
      INTO cols, rcols FROM information_schema.columns
     WHERE table_schema = 'public' AND table_name = 'state_group_ufs' AND column_name NOT IN ('id', 'enterprise_id');
    EXECUTE format('INSERT INTO public.state_group_ufs (%s, enterprise_id)
                    SELECT %s, g.enterprise_id FROM public.state_group_ufs u
                      JOIN public.state_groups g ON g.code = u.state_group_code AND g._origem_id IS NOT NULL
                     WHERE u.enterprise_id IS DISTINCT FROM g.enterprise_id', cols, rcols);
END $$;
DELETE FROM public.state_group_ufs WHERE enterprise_id IS NULL;
ALTER TABLE public.state_group_ufs ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.state_group_ufs ADD CONSTRAINT state_group_ufs_empresa_grupo_uf_key UNIQUE (enterprise_id, state_group_code, uf);
ALTER TABLE public.state_group_ufs
    ADD CONSTRAINT state_group_ufs_grupo_fkey FOREIGN KEY (enterprise_id, state_group_code)
    REFERENCES public.state_groups (enterprise_id, code) ON DELETE CASCADE;

-- tipo de nota: tabela de venda de outra empresa não vale
UPDATE public.invoice_types it SET ipi_transfer_sales_table_id = NULL
  FROM public.sales_tables st
 WHERE st.id = it.ipi_transfer_sales_table_id AND st.enterprise_id IS DISTINCT FROM it.enterprise_id;

-- referências por id → cópia da empresa do registro
SELECT pg_temp.remap('customers', 'region_id', 'regions', 'x.enterprise_id');
SELECT pg_temp.remap('customers', 'market_segment_id', 'market_segments', 'x.enterprise_id');
SELECT pg_temp.remap('customers', 'customer_type_id', 'customer_types', 'x.enterprise_id');
SELECT pg_temp.remap('customers', 'carrier_id', 'carriers', 'x.enterprise_id');
SELECT pg_temp.remap('customers', 'carrier_group_id', 'carrier_groups', 'x.enterprise_id');
SELECT pg_temp.remap('customers', 'invoice_type_id', 'invoice_types', 'x.enterprise_id');
SELECT pg_temp.remap('customers', 'tax_type_id', 'tax_types', 'x.enterprise_id');
SELECT pg_temp.remap('customer_contacts', 'contact_type_id', 'customer_contact_types', 'c.enterprise_id', ', public.customers c', 'c.id = x.customer_id');
SELECT pg_temp.remap('supplier_contacts', 'contact_type_id', 'supplier_contact_types', 's.enterprise_id', ', public.suppliers s', 's.id = x.supplier_id');
SELECT pg_temp.remap('payment_condition_installments', 'carrier_id', 'carriers', 'p.enterprise_id', ', public.payment_conditions p', 'p.id = x.payment_condition_id');
SELECT pg_temp.remap('suppliers', 'region_id', 'regions', 'x.enterprise_id');
SELECT pg_temp.remap('suppliers', 'carrier_id', 'carriers', 'x.enterprise_id');
SELECT pg_temp.remap('suppliers', 'supplier_type_id', 'supplier_types', 'x.enterprise_id');
SELECT pg_temp.remap('commercial_policies', 'region_id', 'regions', 'x.enterprise_id');
SELECT pg_temp.remap('commercial_policies', 'market_segment_id', 'market_segments', 'x.enterprise_id');
SELECT pg_temp.remap('commercial_policies', 'customer_type_id', 'customer_types', 'x.enterprise_id');
SELECT pg_temp.remap('commercial_policies', 'carrier_id', 'carriers', 'x.enterprise_id');
SELECT pg_temp.remap('payment_conditions', 'carrier_id', 'carriers', 'x.enterprise_id');
SELECT pg_temp.remap('icms_ipi_tax_params', 'market_segment_id', 'market_segments', 'x.enterprise_id');
SELECT pg_temp.remap('icms_ipi_tax_params', 'invoice_type_exit_id', 'invoice_types', 'x.enterprise_id');
SELECT pg_temp.remap('icms_ipi_tax_params', 'invoice_type_entry_id', 'invoice_types', 'x.enterprise_id');
SELECT pg_temp.remap('icms_ipi_tax_params', 'tax_type_id', 'tax_types', 'x.enterprise_id');
SELECT pg_temp.remap('icms_reduction_substitutions', 'market_segment_id', 'market_segments', 'x.enterprise_id');
SELECT pg_temp.remap('icms_reduction_substitutions', 'invoice_type_out_id', 'invoice_types', 'x.enterprise_id');
SELECT pg_temp.remap('icms_reduction_substitutions', 'invoice_type_in_id', 'invoice_types', 'x.enterprise_id');
SELECT pg_temp.remap('special_adjustment_notes', 'invoice_type_id', 'invoice_types', 'x.empresa_id');
SELECT pg_temp.remap('supplier_enterprises', 'default_invoice_type_id', 'invoice_types', 'e.id', ', public.enterprise e', 'e.code = x.enterprise_code');
SELECT pg_temp.remap('supplier_parameters', 'purchase_supplier_type_id', 'supplier_types', 'e.id', ', public.enterprise e', 'e.code = x.enterprise_code');

DO $$ DECLARE t TEXT; BEGIN
    FOREACH t IN ARRAY ARRAY['regions','market_segments','customer_contact_types','customer_types','carriers','carrier_groups',
                             'invoice_types','tax_types','supplier_types','supplier_contact_types','entry_operation_types',
                             'state_groups','stock_movement_types'] LOOP
        EXECUTE format('ALTER TABLE public.%I DROP COLUMN _origem_id', t);
        EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON public.%I (enterprise_id)', 'ix_' || t || '_enterprise', t);
    END LOOP;
END $$;

-- ─── Família B (código é a chave) ───────────────────────────────────────────
-- catalogo_b: dono = menor empresa; as demais recebem cópia com código novo.
CREATE FUNCTION pg_temp.catalogo_b(tab TEXT) RETURNS VOID LANGUAGE plpgsql AS $f$
DECLARE cols TEXT; rcols TEXT;
BEGIN
    EXECUTE format('ALTER TABLE public.%I ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id)', tab);
    EXECUTE format('UPDATE public.%I SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL', tab);
    EXECUTE format('CREATE TEMP TABLE %I ON COMMIT DROP AS
                    SELECT t.code AS antigo, e.id AS enterprise_id,
                           CASE WHEN e.id = t.enterprise_id THEN t.code ELSE nextval(pg_get_serial_sequence(%L, ''code'')) END AS novo
                      FROM public.%I t CROSS JOIN public.enterprise e', '_mapa_' || tab, 'public.' || tab, tab);
    SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position),
           string_agg('t.' || quote_ident(column_name), ',' ORDER BY ordinal_position)
      INTO cols, rcols
      FROM information_schema.columns
     WHERE table_schema = 'public' AND table_name = tab AND column_name NOT IN ('code', 'enterprise_id');
    EXECUTE format('INSERT INTO public.%1$I (code, enterprise_id, %2$s)
                    SELECT m.novo, m.enterprise_id, %3$s FROM %4$I m JOIN public.%1$I t ON t.code = m.antigo
                     WHERE m.novo <> m.antigo', tab, cols, rcols, '_mapa_' || tab);
    EXECUTE format('DELETE FROM public.%I WHERE enterprise_id IS NULL', tab); -- instalação sem empresa
    EXECUTE format('ALTER TABLE public.%I ALTER COLUMN enterprise_id SET NOT NULL', tab);
    EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON public.%I (enterprise_id)', 'ix_' || tab || '_enterprise', tab);
END $f$;

SELECT pg_temp.catalogo_b('representative_types');
SELECT pg_temp.catalogo_b('consumer_service_call_types');
SELECT pg_temp.catalogo_b('consumer_service_knowledge_sources');
SELECT pg_temp.catalogo_b('technical_assistance_defect_groups');
SELECT pg_temp.catalogo_b('technical_assistance_defect_reasons');

-- motivo de defeito pertence ao grupo da mesma empresa
UPDATE public.technical_assistance_defect_reasons r SET group_code = m.novo
  FROM _mapa_technical_assistance_defect_groups m
 WHERE m.antigo = r.group_code AND m.enterprise_id = r.enterprise_id AND r.group_code <> m.novo;

SELECT pg_temp.remap('representatives', 'type_code', 'representative_types', 'e.id', ', public.enterprise e', 'e.code = x.enterprise_code');
SELECT pg_temp.remap('consumer_service_calls', 'call_type_code', 'consumer_service_call_types', 'e.id', ', public.enterprise e', 'e.code = x.enterprise_code');
SELECT pg_temp.remap('consumer_service_calls', 'defect_group_code', 'technical_assistance_defect_groups', 'e.id', ', public.enterprise e', 'e.code = x.enterprise_code');
SELECT pg_temp.remap('consumer_service_calls', 'defect_reason_code', 'technical_assistance_defect_reasons', 'e.id', ', public.enterprise e', 'e.code = x.enterprise_code');
SELECT pg_temp.remap('consumer_service_consumers', 'knowledge_code', 'consumer_service_knowledge_sources', 'ce.enterprise_id',
                     ', public.consumer_service_consumer_enterprises ce', 'ce.consumer_code = x.code');
SELECT pg_temp.remap('technical_assistance_call_items', 'defect_reason_code', 'technical_assistance_defect_reasons', 'e.id',
                     ', public.technical_assistance_calls c, public.enterprise e', 'c.code = x.call_code AND e.code = c.enterprise_code');

ALTER TABLE public.regions ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.market_segments ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.customer_contact_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.customer_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.carriers ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.carrier_groups ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.invoice_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.tax_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.supplier_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.supplier_contact_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.entry_operation_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.state_groups ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.stock_movement_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.representative_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.consumer_service_call_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.consumer_service_knowledge_sources ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.technical_assistance_defect_groups ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.technical_assistance_defect_reasons ALTER COLUMN enterprise_id SET NOT NULL;

-- ─── Garantia no banco: o registro só aponta para catálogo da própria empresa ─
-- FK composta (empresa, id). Vale para qualquer caminho de gravação; a API
-- responde "um dos vínculos informados não existe na empresa autenticada".
ALTER TABLE public.regions ADD CONSTRAINT regions_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.market_segments ADD CONSTRAINT market_segments_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.customer_contact_types ADD CONSTRAINT customer_contact_types_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.customer_types ADD CONSTRAINT customer_types_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.carriers ADD CONSTRAINT carriers_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.carrier_groups ADD CONSTRAINT carrier_groups_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.invoice_types ADD CONSTRAINT invoice_types_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.tax_types ADD CONSTRAINT tax_types_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.supplier_types ADD CONSTRAINT supplier_types_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.supplier_contact_types ADD CONSTRAINT supplier_contact_types_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.entry_operation_types ADD CONSTRAINT entry_operation_types_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.state_groups ADD CONSTRAINT state_groups_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.stock_movement_types ADD CONSTRAINT stock_movement_types_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.payment_conditions ADD CONSTRAINT payment_conditions_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.sales_tables ADD CONSTRAINT sales_tables_empresa_id_key UNIQUE (enterprise_id, id);
ALTER TABLE public.customers ADD CONSTRAINT customers_region_id_empresa_fkey FOREIGN KEY (enterprise_id, region_id) REFERENCES public.regions (enterprise_id, id);
ALTER TABLE public.customers ADD CONSTRAINT customers_market_segment_id_empresa_fkey FOREIGN KEY (enterprise_id, market_segment_id) REFERENCES public.market_segments (enterprise_id, id);
ALTER TABLE public.customers ADD CONSTRAINT customers_customer_type_id_empresa_fkey FOREIGN KEY (enterprise_id, customer_type_id) REFERENCES public.customer_types (enterprise_id, id);
ALTER TABLE public.customers ADD CONSTRAINT customers_carrier_id_empresa_fkey FOREIGN KEY (enterprise_id, carrier_id) REFERENCES public.carriers (enterprise_id, id);
ALTER TABLE public.customers ADD CONSTRAINT customers_carrier_group_id_empresa_fkey FOREIGN KEY (enterprise_id, carrier_group_id) REFERENCES public.carrier_groups (enterprise_id, id);
ALTER TABLE public.customers ADD CONSTRAINT customers_invoice_type_id_empresa_fkey FOREIGN KEY (enterprise_id, invoice_type_id) REFERENCES public.invoice_types (enterprise_id, id);
ALTER TABLE public.customers ADD CONSTRAINT customers_tax_type_id_empresa_fkey FOREIGN KEY (enterprise_id, tax_type_id) REFERENCES public.tax_types (enterprise_id, id);
ALTER TABLE public.customers ADD CONSTRAINT customers_payment_condition_id_empresa_fkey FOREIGN KEY (enterprise_id, payment_condition_id) REFERENCES public.payment_conditions (enterprise_id, id);
ALTER TABLE public.customers ADD CONSTRAINT customers_sales_table_id_empresa_fkey FOREIGN KEY (enterprise_id, sales_table_id) REFERENCES public.sales_tables (enterprise_id, id);
ALTER TABLE public.suppliers ADD CONSTRAINT suppliers_region_id_empresa_fkey FOREIGN KEY (enterprise_id, region_id) REFERENCES public.regions (enterprise_id, id);
ALTER TABLE public.suppliers ADD CONSTRAINT suppliers_carrier_id_empresa_fkey FOREIGN KEY (enterprise_id, carrier_id) REFERENCES public.carriers (enterprise_id, id);
ALTER TABLE public.suppliers ADD CONSTRAINT suppliers_supplier_type_id_empresa_fkey FOREIGN KEY (enterprise_id, supplier_type_id) REFERENCES public.supplier_types (enterprise_id, id);
ALTER TABLE public.commercial_policies ADD CONSTRAINT commercial_policies_region_id_empresa_fkey FOREIGN KEY (enterprise_id, region_id) REFERENCES public.regions (enterprise_id, id);
ALTER TABLE public.commercial_policies ADD CONSTRAINT commercial_policies_market_segment_id_empresa_fkey FOREIGN KEY (enterprise_id, market_segment_id) REFERENCES public.market_segments (enterprise_id, id);
ALTER TABLE public.commercial_policies ADD CONSTRAINT commercial_policies_customer_type_id_empresa_fkey FOREIGN KEY (enterprise_id, customer_type_id) REFERENCES public.customer_types (enterprise_id, id);
ALTER TABLE public.commercial_policies ADD CONSTRAINT commercial_policies_carrier_id_empresa_fkey FOREIGN KEY (enterprise_id, carrier_id) REFERENCES public.carriers (enterprise_id, id);
ALTER TABLE public.payment_conditions ADD CONSTRAINT payment_conditions_carrier_id_empresa_fkey FOREIGN KEY (enterprise_id, carrier_id) REFERENCES public.carriers (enterprise_id, id);
ALTER TABLE public.icms_ipi_tax_params ADD CONSTRAINT icms_ipi_tax_params_market_segment_id_empresa_fkey FOREIGN KEY (enterprise_id, market_segment_id) REFERENCES public.market_segments (enterprise_id, id) ON DELETE SET NULL (market_segment_id);
ALTER TABLE public.icms_ipi_tax_params ADD CONSTRAINT icms_ipi_tax_params_invoice_type_exit_id_empresa_fkey FOREIGN KEY (enterprise_id, invoice_type_exit_id) REFERENCES public.invoice_types (enterprise_id, id) ON DELETE SET NULL (invoice_type_exit_id);
ALTER TABLE public.icms_ipi_tax_params ADD CONSTRAINT icms_ipi_tax_params_invoice_type_entry_id_empresa_fkey FOREIGN KEY (enterprise_id, invoice_type_entry_id) REFERENCES public.invoice_types (enterprise_id, id) ON DELETE SET NULL (invoice_type_entry_id);
ALTER TABLE public.icms_ipi_tax_params ADD CONSTRAINT icms_ipi_tax_params_tax_type_id_empresa_fkey FOREIGN KEY (enterprise_id, tax_type_id) REFERENCES public.tax_types (enterprise_id, id) ON DELETE SET NULL (tax_type_id);
ALTER TABLE public.icms_reduction_substitutions ADD CONSTRAINT icms_reduction_substitutions_market_segment_id_empresa_fkey FOREIGN KEY (enterprise_id, market_segment_id) REFERENCES public.market_segments (enterprise_id, id);
ALTER TABLE public.icms_reduction_substitutions ADD CONSTRAINT icms_reduction_substitutions_invoice_type_out_id_empresa_fkey FOREIGN KEY (enterprise_id, invoice_type_out_id) REFERENCES public.invoice_types (enterprise_id, id);
ALTER TABLE public.icms_reduction_substitutions ADD CONSTRAINT icms_reduction_substitutions_invoice_type_in_id_empresa_fkey FOREIGN KEY (enterprise_id, invoice_type_in_id) REFERENCES public.invoice_types (enterprise_id, id);
ALTER TABLE public.market_segments ADD CONSTRAINT market_segments_parent_id_empresa_fkey FOREIGN KEY (enterprise_id, parent_id) REFERENCES public.market_segments (enterprise_id, id);
ALTER TABLE public.invoice_types ADD CONSTRAINT invoice_types_ipi_transfer_sales_table_id_empresa_fkey FOREIGN KEY (enterprise_id, ipi_transfer_sales_table_id) REFERENCES public.sales_tables (enterprise_id, id);
ALTER TABLE public.special_adjustment_notes ADD CONSTRAINT special_adjustment_notes_invoice_type_empresa_fkey
    FOREIGN KEY (empresa_id, invoice_type_id) REFERENCES public.invoice_types (enterprise_id, id);
ALTER TABLE public.technical_assistance_defect_groups ADD CONSTRAINT technical_assistance_defect_groups_empresa_code_key UNIQUE (enterprise_id, code);
ALTER TABLE public.technical_assistance_defect_reasons ADD CONSTRAINT technical_assistance_defect_reasons_grupo_empresa_fkey
    FOREIGN KEY (enterprise_id, group_code) REFERENCES public.technical_assistance_defect_groups (enterprise_id, code);

-- ─── Gatilhos de integridade: catálogo da mesma empresa ─────────────────────
-- Onde a FK composta não cabe (tabela sem enterprise_id, ou na convenção
-- enterprise_code), um gatilho confere a empresa e recusa com o mesmo código de
-- erro de FK (23503), que a API traduz para "um dos vínculos informados não
-- existe na empresa autenticada".
CREATE OR REPLACE FUNCTION public.erp_recusa_catalogo(o_que TEXT) RETURNS VOID LANGUAGE plpgsql AS $f$
BEGIN
    RAISE EXCEPTION '% não pertence à empresa do registro', o_que USING ERRCODE = 'foreign_key_violation';
END $f$;

CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_contato_cliente() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.contact_type_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.customer_contact_types t JOIN public.customers c ON c.enterprise_id = t.enterprise_id
         WHERE t.id = NEW.contact_type_id AND c.id = NEW.customer_id) THEN
        PERFORM public.erp_recusa_catalogo('tipo de contato');
    END IF;
    RETURN NEW;
END $f$;
CREATE TRIGGER trg_customer_contacts_catalogo BEFORE INSERT OR UPDATE OF contact_type_id, customer_id ON public.customer_contacts
    FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_contato_cliente();

CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_contato_fornecedor() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.contact_type_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.supplier_contact_types t JOIN public.suppliers s ON s.enterprise_id = t.enterprise_id
         WHERE t.id = NEW.contact_type_id AND s.id = NEW.supplier_id) THEN
        PERFORM public.erp_recusa_catalogo('tipo de contato');
    END IF;
    RETURN NEW;
END $f$;
CREATE TRIGGER trg_supplier_contacts_catalogo BEFORE INSERT OR UPDATE OF contact_type_id, supplier_id ON public.supplier_contacts
    FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_contato_fornecedor();

CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_parcela() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.carrier_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.carriers k JOIN public.payment_conditions p ON p.enterprise_id = k.enterprise_id
         WHERE k.id = NEW.carrier_id AND p.id = NEW.payment_condition_id) THEN
        PERFORM public.erp_recusa_catalogo('portador');
    END IF;
    RETURN NEW;
END $f$;
CREATE TRIGGER trg_payment_condition_installments_catalogo BEFORE INSERT OR UPDATE OF carrier_id, payment_condition_id ON public.payment_condition_installments
    FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_parcela();

CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_fornecedor_empresa() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.default_invoice_type_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.invoice_types t JOIN public.enterprise e ON e.id = t.enterprise_id
         WHERE t.id = NEW.default_invoice_type_id AND e.code = NEW.enterprise_code) THEN
        PERFORM public.erp_recusa_catalogo('tipo de nota');
    END IF;
    RETURN NEW;
END $f$;
CREATE TRIGGER trg_supplier_enterprises_catalogo BEFORE INSERT OR UPDATE OF default_invoice_type_id, enterprise_code ON public.supplier_enterprises
    FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_fornecedor_empresa();

CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_parametro_fornecedor() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.purchase_supplier_type_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.supplier_types t JOIN public.enterprise e ON e.id = t.enterprise_id
         WHERE t.id = NEW.purchase_supplier_type_id AND e.code = NEW.enterprise_code) THEN
        PERFORM public.erp_recusa_catalogo('tipo de fornecedor');
    END IF;
    RETURN NEW;
END $f$;
CREATE TRIGGER trg_supplier_parameters_catalogo BEFORE INSERT OR UPDATE OF purchase_supplier_type_id, enterprise_code ON public.supplier_parameters
    FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_parametro_fornecedor();

CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_pedido_venda() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.bearer_code IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.carriers k JOIN public.enterprise e ON e.id = k.enterprise_id
         WHERE k.code = NEW.bearer_code AND e.code = NEW.enterprise_code) THEN
        PERFORM public.erp_recusa_catalogo('portador');
    END IF;
    IF NEW.tax_type_code IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.tax_types t JOIN public.enterprise e ON e.id = t.enterprise_id
         WHERE t.code = NEW.tax_type_code AND e.code = NEW.enterprise_code) THEN
        PERFORM public.erp_recusa_catalogo('tributação');
    END IF;
    RETURN NEW;
END $f$;
CREATE TRIGGER trg_sales_orders_catalogo BEFORE INSERT OR UPDATE OF bearer_code, tax_type_code, enterprise_code ON public.sales_orders
    FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_pedido_venda();

CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_representante() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.type_code IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.representative_types t JOIN public.enterprise e ON e.id = t.enterprise_id
         WHERE t.code = NEW.type_code AND e.code = NEW.enterprise_code) THEN
        PERFORM public.erp_recusa_catalogo('tipo de representante');
    END IF;
    RETURN NEW;
END $f$;
CREATE TRIGGER trg_representatives_catalogo BEFORE INSERT OR UPDATE OF type_code, enterprise_code ON public.representatives
    FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_representante();

CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_atendimento_sac() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.call_type_code IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.consumer_service_call_types t JOIN public.enterprise e ON e.id = t.enterprise_id
         WHERE t.code = NEW.call_type_code AND e.code = NEW.enterprise_code) THEN
        PERFORM public.erp_recusa_catalogo('tipo de chamado');
    END IF;
    IF NEW.defect_group_code IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.technical_assistance_defect_groups t JOIN public.enterprise e ON e.id = t.enterprise_id
         WHERE t.code = NEW.defect_group_code AND e.code = NEW.enterprise_code) THEN
        PERFORM public.erp_recusa_catalogo('grupo de defeito');
    END IF;
    IF NEW.defect_reason_code IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.technical_assistance_defect_reasons t JOIN public.enterprise e ON e.id = t.enterprise_id
         WHERE t.code = NEW.defect_reason_code AND e.code = NEW.enterprise_code) THEN
        PERFORM public.erp_recusa_catalogo('motivo de defeito');
    END IF;
    RETURN NEW;
END $f$;
CREATE TRIGGER trg_consumer_service_calls_catalogo BEFORE INSERT OR UPDATE OF call_type_code, defect_group_code, defect_reason_code, enterprise_code
    ON public.consumer_service_calls FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_atendimento_sac();

-- O consumidor ganha o vínculo de empresa na mesma transação, depois do
-- INSERT: a conferência é adiada para o fim da transação.
CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_consumidor() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.knowledge_code IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.consumer_service_knowledge_sources k
          JOIN public.consumer_service_consumer_enterprises ce ON ce.enterprise_id = k.enterprise_id
         WHERE k.code = NEW.knowledge_code AND ce.consumer_code = NEW.code) THEN
        PERFORM public.erp_recusa_catalogo('fonte de conhecimento');
    END IF;
    RETURN NULL;
END $f$;
CREATE CONSTRAINT TRIGGER trg_consumer_service_consumers_catalogo AFTER INSERT OR UPDATE OF knowledge_code ON public.consumer_service_consumers
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_consumidor();

CREATE OR REPLACE FUNCTION public.erp_confere_catalogo_item_assistencia() RETURNS TRIGGER LANGUAGE plpgsql AS $f$
BEGIN
    IF NEW.defect_reason_code IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM public.technical_assistance_defect_reasons t
          JOIN public.enterprise e ON e.id = t.enterprise_id
          JOIN public.technical_assistance_calls c ON c.enterprise_code = e.code
         WHERE t.code = NEW.defect_reason_code AND c.code = NEW.call_code) THEN
        PERFORM public.erp_recusa_catalogo('motivo de defeito');
    END IF;
    RETURN NEW;
END $f$;
CREATE TRIGGER trg_technical_assistance_call_items_catalogo BEFORE INSERT OR UPDATE OF defect_reason_code, call_code
    ON public.technical_assistance_call_items FOR EACH ROW EXECUTE FUNCTION public.erp_confere_catalogo_item_assistencia();
