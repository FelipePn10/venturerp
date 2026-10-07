-- Pedido de compra: registro do envio ao fornecedor (PDF por e-mail) e o
-- acompanhamento de entrega de cada linha (o que o fornecedor prometeu, quem
-- falou e quando). Ambos são histórico: só se insere.
CREATE TABLE IF NOT EXISTS public.purchase_order_envios (
    id                  BIGSERIAL PRIMARY KEY,
    enterprise_code     BIGINT      NOT NULL,
    purchase_order_code BIGINT      NOT NULL REFERENCES public.purchase_orders(code),
    enviado_em          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    enviado_por         UUID        REFERENCES public.users(id),
    destinatarios       TEXT        NOT NULL,
    assunto             TEXT        NOT NULL,
    situacao            VARCHAR(10) NOT NULL CHECK (situacao IN ('ENVIADO', 'FALHOU')),
    erro                TEXT
);
CREATE INDEX IF NOT EXISTS ix_purchase_order_envios_pedido
    ON public.purchase_order_envios (enterprise_code, purchase_order_code, enviado_em DESC);

CREATE TABLE IF NOT EXISTS public.purchase_order_item_followups (
    id                       BIGSERIAL PRIMARY KEY,
    enterprise_code          BIGINT      NOT NULL,
    purchase_order_code      BIGINT      NOT NULL REFERENCES public.purchase_orders(code),
    purchase_order_item_code BIGINT      NOT NULL REFERENCES public.purchase_order_items(code),
    data_prometida           DATE,
    contato                  VARCHAR(150),
    observacao               TEXT,
    registrado_em            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    registrado_por           UUID        REFERENCES public.users(id)
);
CREATE INDEX IF NOT EXISTS ix_purchase_order_item_followups_linha
    ON public.purchase_order_item_followups (enterprise_code, purchase_order_item_code, registrado_em DESC);

-- Acompanhamento de entregas: linhas em aberto por empresa e data.
CREATE INDEX IF NOT EXISTS ix_purchase_order_items_entrega
    ON public.purchase_order_items (purchase_order_code, delivery_date, promised_date)
    WHERE is_active AND status IN ('OPEN', 'PARTIAL');
