ALTER TABLE public.contas_pagar DROP COLUMN IF EXISTS freight_document_id;
DROP TABLE IF EXISTS public.fiscal_freight_allocations;
DROP TABLE IF EXISTS public.fiscal_freight_document_entries;
DROP TABLE IF EXISTS public.fiscal_freight_documents;
DROP INDEX IF EXISTS public.ix_fiscal_cte_tenant;
ALTER TABLE public.fiscal_cte DROP COLUMN IF EXISTS enterprise_id;
