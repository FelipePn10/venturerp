-- XML do evento de cancelamento da NF-e (também tem de ser guardado 5 anos) e
-- a marca da última tentativa de baixar o XML, para a rotina que completa as
-- notas antigas não insistir na mesma nota a cada ciclo.
ALTER TABLE public.fiscal_exits
    ADD COLUMN IF NOT EXISTS xml_cancelamento TEXT,
    ADD COLUMN IF NOT EXISTS xml_tentativa_em TIMESTAMPTZ;
ALTER TABLE public.nfse ADD COLUMN IF NOT EXISTS xml_tentativa_em TIMESTAMPTZ;
