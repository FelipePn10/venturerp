-- Volta sem a coluna de empresa. As cópias criadas na subida continuam como
-- representantes independentes: os códigos novos já podem estar em pedidos.
DROP INDEX IF EXISTS public.ix_representatives_enterprise;
ALTER TABLE public.representatives DROP COLUMN IF EXISTS enterprise_code;
