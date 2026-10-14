-- As cópias criadas na subida continuam como ferramentas independentes.
DROP INDEX IF EXISTS public.ix_tools_enterprise;
ALTER TABLE public.tools DROP COLUMN IF EXISTS enterprise_id;
