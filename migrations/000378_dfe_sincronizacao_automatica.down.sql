DROP INDEX IF EXISTS public.ix_fiscal_received_documents_prazo;
ALTER TABLE public.fiscal_configs
    DROP COLUMN IF EXISTS dfe_ultimo_erro,
    DROP COLUMN IF EXISTS dfe_ultima_tentativa,
    DROP COLUMN IF EXISTS dfe_sync_automatico;
