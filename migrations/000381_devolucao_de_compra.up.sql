-- Devolução de compra: NF-e de saída (finalidade 4) que referencia a nota de
-- entrada e devolve parte dos itens ao fornecedor.
ALTER TABLE public.fiscal_exits
    ADD COLUMN IF NOT EXISTS finalidade       SMALLINT NOT NULL DEFAULT 1 CHECK (finalidade IN (1,2,3,4)),
    ADD COLUMN IF NOT EXISTS nfe_referenciada VARCHAR(44),
    ADD COLUMN IF NOT EXISTS fiscal_entry_id  BIGINT REFERENCES public.fiscal_entries(id),
    ADD COLUMN IF NOT EXISTS supplier_code    BIGINT;
CREATE INDEX IF NOT EXISTS ix_fiscal_exits_devolucao ON public.fiscal_exits (enterprise_id, fiscal_entry_id) WHERE fiscal_entry_id IS NOT NULL;

ALTER TABLE public.fiscal_exit_items
    ADD COLUMN IF NOT EXISTS fiscal_entry_item_id BIGINT REFERENCES public.fiscal_entry_items(id);

-- Crédito com o fornecedor: o que a devolução não conseguiu abater de títulos
-- em aberto (já pagos) vira um título a receber do fornecedor.
ALTER TABLE public.contas_receber
    ADD COLUMN IF NOT EXISTS fornecedor_id BIGINT;

-- O que a devolução fez no financeiro, para o cancelamento desfazer.
CREATE TABLE IF NOT EXISTS public.fiscal_return_settlements (
    id               BIGSERIAL PRIMARY KEY,
    enterprise_id    BIGINT NOT NULL REFERENCES public.enterprise(id),
    fiscal_exit_id   BIGINT NOT NULL REFERENCES public.fiscal_exits(id),
    conta_pagar_id   BIGINT REFERENCES public.contas_pagar(id),
    conta_receber_id BIGINT REFERENCES public.contas_receber(id),
    valor            NUMERIC(15,2) NOT NULL CHECK (valor > 0),
    -- o abatimento zerou o título (que passou a PAGO): o estorno reabre.
    quitou_titulo    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    estornado_em     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS ix_fiscal_return_settlements ON public.fiscal_return_settlements (enterprise_id, fiscal_exit_id);
