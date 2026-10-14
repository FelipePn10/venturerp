-- Adiantamentos por empresa. A empresa vinha só da conta bancária (obrigatória);
-- passa a ser coluna do próprio adiantamento, como nos demais documentos
-- financeiros. Backfill pela empresa da conta bancária.
ALTER TABLE public.adiantamentos ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.adiantamentos a SET enterprise_id = cb.enterprise_id
  FROM public.contas_bancarias cb WHERE cb.id = a.conta_bancaria_id AND a.enterprise_id IS NULL;
UPDATE public.adiantamentos SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE public.adiantamentos ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_adiantamentos_enterprise ON public.adiantamentos (enterprise_id, data_adiantamento DESC);
