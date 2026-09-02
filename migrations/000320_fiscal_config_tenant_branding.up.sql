ALTER TABLE public.fiscal_configs
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id),
    ADD COLUMN IF NOT EXISTS trade_name VARCHAR(200),
    ADD COLUMN IF NOT EXISTS email VARCHAR(200);

UPDATE public.fiscal_configs
SET enterprise_id = (SELECT MIN(id) FROM public.enterprise)
WHERE enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM public.enterprise) = 1;

-- A fundação fiscal cria um placeholder antes de existir qualquer empresa.
-- Ele não contém configuração operacional e não pode permanecer sem tenant.
DELETE FROM public.fiscal_configs
WHERE enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM public.enterprise) = 0;

INSERT INTO public.fiscal_configs (
    enterprise_id, cnpj_empresa, razao_social, trade_name, ie_empresa,
    regime_tributario, uf_empresa, updated_by
)
SELECT
    e.id, '00000000000000', e.name, e.name, 'ISENTO',
    'lucro_real', 'PR', '00000000-0000-0000-0000-000000000000'
FROM public.enterprise e
WHERE NOT EXISTS (
    SELECT 1 FROM public.fiscal_configs f WHERE f.enterprise_id = e.id
);

ALTER TABLE public.fiscal_configs
    ALTER COLUMN enterprise_id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_fiscal_configs_enterprise
    ON public.fiscal_configs (enterprise_id);
