BEGIN;

-- Plano de Corte — Fase 2: firmar (baixa real de estoque) + retalhos rastreáveis.

-- Retalhos reaproveitáveis: cada retalho é uma peça física ÚNICA (geometria
-- própria), por isso tem inventário dedicado — não é saldo fungível. Herda
-- corrida (heat_number) e certificado do material de origem para rastreabilidade.
CREATE TABLE IF NOT EXISTS public.stock_remnants (
    id              BIGSERIAL PRIMARY KEY,
    item_code       BIGINT NOT NULL,                 -- item de matéria-prima
    warehouse_id    BIGINT NOT NULL,
    length_mm       NUMERIC(15,4) NOT NULL,          -- geometria 1D (fase 3: width/height)
    lot             VARCHAR(50),                     -- lote de origem
    heat_number     VARCHAR(50),                     -- corrida herdada
    certificate     VARCHAR(120),                    -- certificado herdado
    status          VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE'
                      CHECK (status IN ('AVAILABLE','RESERVED','CONSUMED')),
    unit_cost       NUMERIC(15,4) NOT NULL DEFAULT 0,
    origin_plan_id  BIGINT,                          -- plano que gerou o retalho
    consumed_plan_id BIGINT,                         -- plano que consumiu o retalho
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_stock_remnants_item_status ON public.stock_remnants(item_code, warehouse_id, status);
CREATE INDEX IF NOT EXISTS idx_stock_remnants_origin ON public.stock_remnants(origin_plan_id);

-- Consumo do plano: o que cada plano firmado baixou (lote ou retalho), com custo,
-- formando a rastreabilidade de quem-consumiu-o-quê.
CREATE TABLE IF NOT EXISTS public.cutting_plan_consumptions (
    id            BIGSERIAL PRIMARY KEY,
    plan_id       BIGINT NOT NULL REFERENCES public.cutting_plans(id) ON DELETE CASCADE,
    item_code     BIGINT NOT NULL,
    source_type   VARCHAR(20) NOT NULL CHECK (source_type IN ('LOT','REMNANT')),
    lot           VARCHAR(50),
    remnant_id    BIGINT,
    quantity      NUMERIC(15,4) NOT NULL,            -- peças consumidas
    length_mm     NUMERIC(15,4) NOT NULL,            -- comprimento por peça
    unit_cost     NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_cost    NUMERIC(15,4) NOT NULL DEFAULT 0,
    warehouse_id  BIGINT NOT NULL,
    movement_id   BIGINT,                            -- stock_movements.id quando houve baixa
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cutting_plan_consumptions_plan ON public.cutting_plan_consumptions(plan_id);

-- Configuração da empresa para o plano de corte (singleton id=1): a empresa
-- decide o modo de consumo padrão (automático FIFO ou manual por lote).
CREATE TABLE IF NOT EXISTS public.cutting_settings (
    id                       BIGINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    default_consumption_mode VARCHAR(20) NOT NULL DEFAULT 'AUTOMATIC'
                               CHECK (default_consumption_mode IN ('AUTOMATIC','MANUAL')),
    default_min_remnant_mm   NUMERIC(12,4) NOT NULL DEFAULT 0,
    default_warehouse_id     BIGINT,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO public.cutting_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

-- Extensões ao cabeçalho do plano: depósito de baixa, OP vinculada, modo de
-- consumo (NULL = herda da configuração da empresa), inclusão automática de
-- retalhos na otimização e carimbo de quando foi firmado.
ALTER TABLE public.cutting_plans
    ADD COLUMN IF NOT EXISTS warehouse_id          BIGINT,
    ADD COLUMN IF NOT EXISTS production_order_code  BIGINT,
    ADD COLUMN IF NOT EXISTS lot_consumption_mode   VARCHAR(20)
        CHECK (lot_consumption_mode IS NULL OR lot_consumption_mode IN ('AUTOMATIC','MANUAL')),
    ADD COLUMN IF NOT EXISTS include_remnants       BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS released_at            TIMESTAMPTZ;

-- Extensões à peça de estoque: vínculo com um retalho real do inventário e
-- corrida carimbada (para entrada manual com rastreabilidade).
ALTER TABLE public.cutting_stock_pieces
    ADD COLUMN IF NOT EXISTS remnant_id   BIGINT,
    ADD COLUMN IF NOT EXISTS heat_number  VARCHAR(50);

COMMIT;
