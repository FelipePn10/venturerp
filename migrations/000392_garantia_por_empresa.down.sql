-- As cópias criadas na subida continuam como responsáveis independentes.
DROP INDEX IF EXISTS public.ix_ta_warranty_responsibles_enterprise;
ALTER TABLE public.technical_assistance_warranty_responsibles DROP COLUMN IF EXISTS enterprise_code;
