-- Título a receber da NFS-e.
--
-- A NFS-e autorizada não gerava conta a receber: o serviço faturado só entrava
-- no financeiro se alguém lembrasse de lançá-lo à mão. Agora a nota guarda o
-- cliente (opcional) e o vencimento; com cliente, a autorização gera o título
-- pelo valor líquido e o cancelamento da nota cancela o título em aberto.
ALTER TABLE public.nfse
    ADD COLUMN IF NOT EXISTS customer_code   BIGINT,
    ADD COLUMN IF NOT EXISTS data_vencimento DATE;

ALTER TABLE public.contas_receber
    ADD COLUMN IF NOT EXISTS nfse_id BIGINT REFERENCES public.nfse(id);

CREATE INDEX IF NOT EXISTS ix_contas_receber_nfse
    ON public.contas_receber (nfse_id) WHERE nfse_id IS NOT NULL;

-- O cliente e a NFS-e do título têm de ser da mesma empresa da linha.
SELECT instalar_conferencia_de_empresa();
