-- CT-e SEFAZ authorization support. The CT-e was previously a local cost record;
-- these columns let it be transmitted to SEFAZ via Focus NF-e. The full emission
-- detail (remetente, destinatário, tomador, modal, municípios) is stored as JSONB
-- so the document can be authorized without a 40-column schema.

ALTER TABLE public.fiscal_cte
    ADD COLUMN IF NOT EXISTS focus_ref     VARCHAR(60),
    ADD COLUMN IF NOT EXISTS protocolo     VARCHAR(60),
    ADD COLUMN IF NOT EXISTS emission_data JSONB;
