-- Rateio de despesas indiretas por empresa. A regra (overhead_allocations) e
-- os destinos não tinham empresa: a lista e a leitura por código mostravam as
-- regras de todas as empresas. A regra é da empresa do centro de custo de
-- origem; os destinos acompanham a regra.
ALTER TABLE public.overhead_allocations ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.overhead_allocations o SET enterprise_id = cc.enterprise_id
  FROM public.cost_centers cc
 WHERE o.enterprise_id IS NULL AND (cc.code = o.cost_center_code OR cc.id = o.cost_center_id) AND cc.enterprise_id IS NOT NULL;
UPDATE public.overhead_allocations SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE public.overhead_allocations ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_overhead_allocations_enterprise ON public.overhead_allocations (enterprise_id, created_at DESC);
