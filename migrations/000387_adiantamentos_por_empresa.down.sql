DROP INDEX IF EXISTS public.ix_adiantamentos_enterprise;
ALTER TABLE public.adiantamentos DROP COLUMN IF EXISTS enterprise_id;
