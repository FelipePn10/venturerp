ALTER TABLE public.nfse DROP COLUMN IF EXISTS xml_tentativa_em;
ALTER TABLE public.fiscal_exits
    DROP COLUMN IF EXISTS xml_tentativa_em,
    DROP COLUMN IF EXISTS xml_cancelamento;
