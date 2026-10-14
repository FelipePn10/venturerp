-- Volta à parametrização fiscal compartilhada: apaga as cópias (linha igual a
-- outra de id menor, tirando id, empresa e cliente) e remove a empresa.
DO $$
DECLARE t TEXT; cond TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['icms_ipi_tax_params', 'icms_reduction_substitutions', 'tax_scenarios', 'icms_apuracao_lines'] LOOP
        SELECT string_agg(format('o.%1$I IS NOT DISTINCT FROM c.%1$I', column_name), ' AND ')
          INTO cond
          FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = t
           AND column_name NOT IN ('id', 'enterprise_id', 'customer_id', 'created_at', 'updated_at');
        IF t = 'icms_apuracao_lines' THEN
            EXECUTE format('UPDATE public.special_adjustment_notes n SET icms_apuracao_line_id = o.id
                              FROM public.icms_apuracao_lines c, public.icms_apuracao_lines o
                             WHERE c.id = n.icms_apuracao_line_id AND o.id < c.id AND %s', cond);
        END IF;
        EXECUTE format('DELETE FROM public.%1$I c USING public.%1$I o WHERE o.id < c.id AND %2$s', t, cond);
    END LOOP;
END $$;

DROP INDEX IF EXISTS public.ix_icms_reduction_substitutions_enterprise;
ALTER TABLE public.icms_reduction_substitutions DROP COLUMN IF EXISTS enterprise_id;
DROP INDEX IF EXISTS public.ix_icms_ipi_tax_params_enterprise;
ALTER TABLE public.icms_ipi_tax_params DROP COLUMN IF EXISTS enterprise_id;

ALTER TABLE public.icms_apuracao_lines DROP CONSTRAINT IF EXISTS icms_apuracao_lines_enterprise_code_key;
ALTER TABLE public.icms_apuracao_lines ADD CONSTRAINT icms_apuracao_lines_code_key UNIQUE (code);
ALTER TABLE public.icms_apuracao_lines DROP COLUMN IF EXISTS enterprise_id;

ALTER TABLE public.tax_scenarios DROP CONSTRAINT IF EXISTS tax_scenarios_enterprise_name_key;
ALTER TABLE public.tax_scenarios ADD CONSTRAINT tax_scenarios_scenario_name_key UNIQUE (scenario_name);
ALTER TABLE public.tax_scenarios DROP COLUMN IF EXISTS enterprise_id;

ALTER TABLE public.icms_summary_entries DROP CONSTRAINT IF EXISTS icms_summary_entries_enterprise_period_uf_cfop_key;
ALTER TABLE public.icms_summary_entries ADD CONSTRAINT icms_summary_entries_period_uf_cfop_id_key UNIQUE (period, uf, cfop_id);
ALTER TABLE public.icms_summary_entries DROP COLUMN IF EXISTS enterprise_id;

ALTER TABLE public.simples_nacional_apuracoes DROP CONSTRAINT IF EXISTS simples_nacional_apuracoes_enterprise_period_annex_key;
ALTER TABLE public.simples_nacional_apuracoes ADD CONSTRAINT simples_nacional_apuracoes_period_annex_key UNIQUE (period, annex);
ALTER TABLE public.simples_nacional_apuracoes DROP COLUMN IF EXISTS enterprise_id;

DROP INDEX IF EXISTS public.ix_nfse_enterprise;
ALTER TABLE public.nfse DROP COLUMN IF EXISTS enterprise_id;
