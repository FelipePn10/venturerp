-- Volta ao cadastro único: mantém o registro mais antigo de cada código (o
-- original) e apaga as cópias criadas por empresa, com endereços e contatos.
DELETE FROM public.customers c
 USING public.customers o
 WHERE o.code = c.code AND o.id < c.id;
ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_enterprise_code_key;
DROP INDEX IF EXISTS public.ix_customers_enterprise_document;
CREATE INDEX IF NOT EXISTS ix_customers_document ON public.customers (document_number);
ALTER TABLE public.customers ADD CONSTRAINT customers_code_key UNIQUE (code);
ALTER TABLE public.customers DROP COLUMN IF EXISTS enterprise_id;
