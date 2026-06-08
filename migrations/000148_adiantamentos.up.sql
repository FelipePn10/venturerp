-- Adiantamentos (advance payments) — ledger of advances paid to suppliers or
-- received from customers, plus the applications of those advances onto
-- contas a pagar / a receber.

CREATE TABLE IF NOT EXISTS public.adiantamentos (
    id                 BIGSERIAL PRIMARY KEY,
    tipo               VARCHAR(10)   NOT NULL CHECK (tipo IN ('PAGAR', 'RECEBER')),
    parceiro_id        BIGINT, -- fornecedor (PAGAR) ou cliente (RECEBER)
    conta_bancaria_id  BIGINT        NOT NULL,
    numero_documento   VARCHAR(60),
    data_adiantamento  DATE          NOT NULL,
    valor_original     NUMERIC(15,2) NOT NULL CHECK (valor_original > 0),
    valor_utilizado    NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (valor_utilizado >= 0),
    status             VARCHAR(20)   NOT NULL DEFAULT 'ABERTO'
                         CHECK (status IN ('ABERTO', 'PARCIAL', 'QUITADO', 'CANCELADO')),
    descricao          TEXT,
    is_active          BOOLEAN       NOT NULL DEFAULT TRUE,
    created_by         UUID,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT adiantamentos_utilizado_le_original CHECK (valor_utilizado <= valor_original)
);

CREATE INDEX IF NOT EXISTS idx_adiantamentos_tipo_parceiro
    ON public.adiantamentos (tipo, parceiro_id);

CREATE TABLE IF NOT EXISTS public.adiantamento_aplicacoes (
    id               BIGSERIAL PRIMARY KEY,
    adiantamento_id  BIGINT        NOT NULL REFERENCES public.adiantamentos (id),
    conta_tipo       VARCHAR(10)   NOT NULL CHECK (conta_tipo IN ('PAGAR', 'RECEBER')),
    conta_id         BIGINT        NOT NULL,
    valor_aplicado   NUMERIC(15,2) NOT NULL CHECK (valor_aplicado > 0),
    data_aplicacao   DATE          NOT NULL DEFAULT CURRENT_DATE,
    created_by       UUID,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_adiantamento_aplicacoes_adiantamento
    ON public.adiantamento_aplicacoes (adiantamento_id);
CREATE INDEX IF NOT EXISTS idx_adiantamento_aplicacoes_conta
    ON public.adiantamento_aplicacoes (conta_tipo, conta_id);
