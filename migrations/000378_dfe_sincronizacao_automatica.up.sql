-- Sincronização automática das NF-e recebidas (distribuição DF-e) e o
-- registro do resultado, para a tela mostrar quando foi a última consulta e o
-- erro da Focus/SEFAZ, se houve.
ALTER TABLE public.fiscal_configs
    ADD COLUMN IF NOT EXISTS dfe_sync_automatico  BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS dfe_ultima_tentativa TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS dfe_ultimo_erro      TEXT;

-- Alerta de prazo de manifestação: notas sem manifestação conclusiva.
CREATE INDEX IF NOT EXISTS ix_fiscal_received_documents_prazo
    ON public.fiscal_received_documents (enterprise_id, data_emissao)
    WHERE manifestacao IS NULL OR manifestacao = 'ciencia';
