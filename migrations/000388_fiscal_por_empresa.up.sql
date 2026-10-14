-- Fiscal por empresa.
--
-- Sem coluna de empresa, uma empresa lia e alterava — e na NFS-e chegava a
-- AUTORIZAR e CANCELAR com o próprio token — o que era de outra:
--   * documentos e apurações (fatos de UMA empresa): NFS-e, apuração do
--     Simples Nacional e totalizadores da apuração de ICMS (com notas e
--     informações adicionais pelo totalizador);
--   * parametrização (configuração que cada empresa ajusta): parâmetros
--     ICMS/IPI, redução/substituição/diferimento de ICMS, linhas da apuração de
--     ICMS e cenários fiscais.
--
-- Backfill:
--   * fato vai para UMA empresa — NFS-e pela empresa do pedido de venda, senão
--     pela única empresa de quem a criou; totalizador gerado por nota de ajuste
--     vai para a empresa da nota; o resto vai para a empresa fiscal (a que tem
--     configuração fiscal; havendo várias, a menor) — nunca é duplicado;
--   * parametrização vai para TODAS as empresas (cada uma recebe a sua cópia),
--     exceto a regra que cita item, fornecedor ou cliente: essa só vai para a
--     empresa dona do item/fornecedor, e para cliente vai para cada empresa que
--     tem o cliente com o mesmo código (apontando para o cadastro dela).
--
-- Continuam globais as tabelas oficiais: CFOP, NCM, alíquotas de ICMS por UF,
-- dispositivos legais, códigos de ajuste da apuração, motivos DAPI.

CREATE TEMP TABLE _empresa_fiscal ON COMMIT DROP AS
SELECT COALESCE((SELECT MIN(enterprise_id) FROM public.fiscal_configs WHERE enterprise_id IS NOT NULL),
                (SELECT MIN(id) FROM public.enterprise)) AS id;

-- ─── NFS-e ──────────────────────────────────────────────────────────────────
ALTER TABLE public.nfse ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.nfse n SET enterprise_id = e.id
  FROM public.sales_orders so JOIN public.enterprise e ON e.code = so.enterprise_code
 WHERE so.code = n.sales_order_code AND n.enterprise_id IS NULL;
UPDATE public.nfse n SET enterprise_id = u.enterprise_id
  FROM (SELECT user_id, MIN(enterprise_id) AS enterprise_id FROM public.user_enterprises GROUP BY user_id HAVING COUNT(*) = 1) u
 WHERE u.user_id = n.created_by AND n.enterprise_id IS NULL;
UPDATE public.nfse SET enterprise_id = (SELECT id FROM _empresa_fiscal) WHERE enterprise_id IS NULL;
ALTER TABLE public.nfse ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_nfse_enterprise ON public.nfse (enterprise_id, created_at DESC);

-- ─── Simples Nacional ───────────────────────────────────────────────────────
ALTER TABLE public.simples_nacional_apuracoes ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.simples_nacional_apuracoes SET enterprise_id = (SELECT id FROM _empresa_fiscal) WHERE enterprise_id IS NULL;
ALTER TABLE public.simples_nacional_apuracoes ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.simples_nacional_apuracoes DROP CONSTRAINT IF EXISTS simples_nacional_apuracoes_period_annex_key;
ALTER TABLE public.simples_nacional_apuracoes ADD CONSTRAINT simples_nacional_apuracoes_enterprise_period_annex_key UNIQUE (enterprise_id, period, annex);

-- ─── Totalizadores da apuração de ICMS ──────────────────────────────────────
ALTER TABLE public.icms_summary_entries ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.icms_summary_entries s SET enterprise_id = n.empresa_id
  FROM public.special_adjustment_notes n
 WHERE n.generated_summary_entry_id = s.id AND s.enterprise_id IS NULL
   AND EXISTS (SELECT 1 FROM public.enterprise e WHERE e.id = n.empresa_id);
UPDATE public.icms_summary_entries SET enterprise_id = (SELECT id FROM _empresa_fiscal) WHERE enterprise_id IS NULL;
ALTER TABLE public.icms_summary_entries ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.icms_summary_entries DROP CONSTRAINT IF EXISTS icms_summary_entries_period_uf_cfop_id_key;
ALTER TABLE public.icms_summary_entries ADD CONSTRAINT icms_summary_entries_enterprise_period_uf_cfop_key UNIQUE (enterprise_id, period, uf, cfop_id);

-- ─── Cenários fiscais ───────────────────────────────────────────────────────
ALTER TABLE public.tax_scenarios ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.tax_scenarios SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE public.tax_scenarios DROP CONSTRAINT IF EXISTS tax_scenarios_scenario_name_key;
INSERT INTO public.tax_scenarios (scenario_name, destination_uf, destination_type, aliq_icms, dif_icms_pct, cst_icms,
                                  calc_difal, aliq_fcp, is_active, created_at, updated_at, enterprise_id)
SELECT t.scenario_name, t.destination_uf, t.destination_type, t.aliq_icms, t.dif_icms_pct, t.cst_icms,
       t.calc_difal, t.aliq_fcp, t.is_active, t.created_at, NOW(), e.id
  FROM public.tax_scenarios t CROSS JOIN public.enterprise e
 WHERE e.id <> t.enterprise_id;
-- Instalação nova (nenhuma empresa ainda): os cenários padrão não têm dono.
DELETE FROM public.tax_scenarios WHERE enterprise_id IS NULL;
ALTER TABLE public.tax_scenarios ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.tax_scenarios ADD CONSTRAINT tax_scenarios_enterprise_name_key UNIQUE (enterprise_id, scenario_name);

-- ─── Linhas da apuração de ICMS ─────────────────────────────────────────────
ALTER TABLE public.icms_apuracao_lines ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.icms_apuracao_lines SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE public.icms_apuracao_lines DROP CONSTRAINT IF EXISTS icms_apuracao_lines_code_key;
INSERT INTO public.icms_apuracao_lines (code, description, line_type, accepts_entries, nature, apuracao_adjustment_code_id,
                                        is_active, created_at, enterprise_id)
SELECT l.code, l.description, l.line_type, l.accepts_entries, l.nature, l.apuracao_adjustment_code_id,
       l.is_active, l.created_at, e.id
  FROM public.icms_apuracao_lines l CROSS JOIN public.enterprise e
 WHERE e.id <> l.enterprise_id;
-- Nota de ajuste aponta para a linha da própria empresa (mesmo código).
UPDATE public.special_adjustment_notes n SET icms_apuracao_line_id = minha.id
  FROM public.icms_apuracao_lines atual, public.icms_apuracao_lines minha
 WHERE atual.id = n.icms_apuracao_line_id AND minha.code = atual.code
   AND minha.enterprise_id = n.empresa_id AND atual.enterprise_id <> n.empresa_id;
DELETE FROM public.icms_apuracao_lines WHERE enterprise_id IS NULL;
ALTER TABLE public.icms_apuracao_lines ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.icms_apuracao_lines ADD CONSTRAINT icms_apuracao_lines_enterprise_code_key UNIQUE (enterprise_id, code);

-- ─── Parâmetros ICMS/IPI ────────────────────────────────────────────────────
ALTER TABLE public.icms_ipi_tax_params ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
CREATE TEMP TABLE _destino_tp ON COMMIT DROP AS
SELECT p.id, e.id AS enterprise_id
  FROM public.icms_ipi_tax_params p CROSS JOIN public.enterprise e
 WHERE (p.item_code IS NULL OR EXISTS (SELECT 1 FROM public.items i WHERE i.code = p.item_code AND i.enterprise_id = e.id))
   AND (p.customer_code IS NULL OR EXISTS (SELECT 1 FROM public.customers c WHERE c.code = p.customer_code AND c.enterprise_id = e.id));
-- Regra cujo item/cliente não existe mais em empresa nenhuma: fica com a menor.
INSERT INTO _destino_tp
SELECT p.id, (SELECT MIN(id) FROM public.enterprise) FROM public.icms_ipi_tax_params p
 WHERE NOT EXISTS (SELECT 1 FROM _destino_tp d WHERE d.id = p.id);
UPDATE public.icms_ipi_tax_params p SET enterprise_id = d.dono
  FROM (SELECT id, MIN(enterprise_id) AS dono FROM _destino_tp GROUP BY id) d WHERE d.id = p.id;
DO $$
DECLARE cols TEXT; rcols TEXT;
BEGIN
    SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position),
           string_agg('p.' || quote_ident(column_name), ',' ORDER BY ordinal_position)
      INTO cols, rcols
      FROM information_schema.columns
     WHERE table_schema = 'public' AND table_name = 'icms_ipi_tax_params' AND column_name NOT IN ('id', 'enterprise_id');
    EXECUTE format('INSERT INTO public.icms_ipi_tax_params (%s, enterprise_id)
                    SELECT %s, d.enterprise_id FROM _destino_tp d JOIN public.icms_ipi_tax_params p ON p.id = d.id
                     WHERE d.enterprise_id <> p.enterprise_id', cols, rcols);
END $$;
ALTER TABLE public.icms_ipi_tax_params ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_icms_ipi_tax_params_enterprise ON public.icms_ipi_tax_params (enterprise_id, uf);

-- ─── Redução / substituição / diferimento de ICMS ───────────────────────────
ALTER TABLE public.icms_reduction_substitutions ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
CREATE TEMP TABLE _destino_rs ON COMMIT DROP AS
SELECT r.id, e.id AS enterprise_id,
       (SELECT c2.id FROM public.customers c1 JOIN public.customers c2 ON c2.code = c1.code AND c2.enterprise_id = e.id
         WHERE c1.id = r.customer_id) AS customer_id
  FROM public.icms_reduction_substitutions r CROSS JOIN public.enterprise e
 WHERE (r.item_id IS NULL OR EXISTS (SELECT 1 FROM public.items i WHERE i.id = r.item_id AND i.enterprise_id = e.id))
   AND (r.supplier_id IS NULL OR EXISTS (SELECT 1 FROM public.suppliers s WHERE s.id = r.supplier_id AND s.enterprise_id = e.id))
   AND (r.customer_id IS NULL OR EXISTS (SELECT 1 FROM public.customers c1 JOIN public.customers c2 ON c2.code = c1.code AND c2.enterprise_id = e.id
                                          WHERE c1.id = r.customer_id));
INSERT INTO _destino_rs
SELECT r.id, (SELECT MIN(id) FROM public.enterprise), r.customer_id FROM public.icms_reduction_substitutions r
 WHERE NOT EXISTS (SELECT 1 FROM _destino_rs d WHERE d.id = r.id);
UPDATE public.icms_reduction_substitutions r SET enterprise_id = d.enterprise_id, customer_id = d.customer_id
  FROM _destino_rs d
 WHERE d.id = r.id AND d.enterprise_id = (SELECT MIN(enterprise_id) FROM _destino_rs x WHERE x.id = r.id);
DO $$
DECLARE cols TEXT; rcols TEXT;
BEGIN
    SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position),
           string_agg('r.' || quote_ident(column_name), ',' ORDER BY ordinal_position)
      INTO cols, rcols
      FROM information_schema.columns
     WHERE table_schema = 'public' AND table_name = 'icms_reduction_substitutions'
       AND column_name NOT IN ('id', 'enterprise_id', 'customer_id');
    EXECUTE format('INSERT INTO public.icms_reduction_substitutions (%s, enterprise_id, customer_id)
                    SELECT %s, d.enterprise_id, d.customer_id
                      FROM _destino_rs d JOIN public.icms_reduction_substitutions r ON r.id = d.id
                     WHERE d.enterprise_id <> r.enterprise_id', cols, rcols);
END $$;
ALTER TABLE public.icms_reduction_substitutions ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_icms_reduction_substitutions_enterprise ON public.icms_reduction_substitutions (enterprise_id, uf);
