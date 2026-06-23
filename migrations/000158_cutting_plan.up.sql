BEGIN;

-- Plano de Corte (Fase 1: corte linear 1D de barras/perfis/tubos).
-- Um plano corta UM item de matéria-prima em várias peças, minimizando perda.
-- Não há "barra padrão": o estoque a cortar é heterogêneo, então cada peça de
-- estoque carrega seu próprio comprimento (cutting_stock_pieces).
CREATE TABLE IF NOT EXISTS public.cutting_plans (
    id                 BIGSERIAL PRIMARY KEY,
    code               BIGINT NOT NULL UNIQUE,
    description        TEXT,
    cut_type           VARCHAR(20) NOT NULL DEFAULT 'LINEAR_1D'
                         CHECK (cut_type IN ('LINEAR_1D','GUILLOTINE_2D','TRUE_SHAPE_2D')),
    source             VARCHAR(20) NOT NULL DEFAULT 'MANUAL'
                         CHECK (source IN ('MANUAL','ORDEM_PRODUCAO','ORDEM_PLANEJADA')),
    status             VARCHAR(20) NOT NULL DEFAULT 'RASCUNHO'
                         CHECK (status IN ('RASCUNHO','OTIMIZADO','FIRMADO','EM_EXECUCAO','CONCLUIDO')),
    material_item_code BIGINT NOT NULL,
    machine_code       BIGINT,
    kerf_mm            NUMERIC(12,4) NOT NULL DEFAULT 0,
    trim_mm            NUMERIC(12,4) NOT NULL DEFAULT 0,
    min_remnant_mm     NUMERIC(12,4) NOT NULL DEFAULT 0,
    -- métricas preenchidas pela otimização
    utilization_pct    NUMERIC(7,4) NOT NULL DEFAULT 0,
    scrap_pct          NUMERIC(7,4) NOT NULL DEFAULT 0,
    stock_used_count   INTEGER NOT NULL DEFAULT 0,
    cut_count          INTEGER NOT NULL DEFAULT 0,
    total_demand_mm    NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_stock_mm     NUMERIC(15,4) NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by         UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cutting_plans_material ON public.cutting_plans(material_item_code);
CREATE INDEX IF NOT EXISTS idx_cutting_plans_status ON public.cutting_plans(status);

-- Demanda: peças a cortar (comprimento × quantidade).
CREATE TABLE IF NOT EXISTS public.cutting_plan_parts (
    id          BIGSERIAL PRIMARY KEY,
    plan_id     BIGINT NOT NULL REFERENCES public.cutting_plans(id) ON DELETE CASCADE,
    item_code   BIGINT,
    label       VARCHAR(200) NOT NULL DEFAULT '',
    length_mm   NUMERIC(15,4) NOT NULL,
    quantity    INTEGER NOT NULL,
    source_ref  VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cutting_plan_parts_plan ON public.cutting_plan_parts(plan_id);

-- Estoque disponível para o plano: cada peça com seu próprio comprimento.
CREATE TABLE IF NOT EXISTS public.cutting_stock_pieces (
    id          BIGSERIAL PRIMARY KEY,
    plan_id     BIGINT NOT NULL REFERENCES public.cutting_plans(id) ON DELETE CASCADE,
    length_mm   NUMERIC(15,4) NOT NULL,
    quantity    INTEGER NOT NULL,
    lot         VARCHAR(50),
    is_remnant  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cutting_stock_pieces_plan ON public.cutting_stock_pieces(plan_id);

-- Resultado: padrões de corte (um layout repetido N vezes).
CREATE TABLE IF NOT EXISTS public.cutting_patterns (
    id              BIGSERIAL PRIMARY KEY,
    plan_id         BIGINT NOT NULL REFERENCES public.cutting_plans(id) ON DELETE CASCADE,
    sequence        INTEGER NOT NULL,
    stock_length_mm NUMERIC(15,4) NOT NULL,
    repeat_count    INTEGER NOT NULL DEFAULT 1,
    used_mm         NUMERIC(15,4) NOT NULL DEFAULT 0,
    kerf_loss_mm    NUMERIC(15,4) NOT NULL DEFAULT 0,
    remnant_mm      NUMERIC(15,4) NOT NULL DEFAULT 0,
    utilization_pct NUMERIC(7,4) NOT NULL DEFAULT 0,
    is_remnant      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cutting_patterns_plan ON public.cutting_patterns(plan_id);

-- Posicionamento de cada peça ao longo da barra (para a instrução de chão-de-fábrica).
CREATE TABLE IF NOT EXISTS public.cutting_pattern_placements (
    id          BIGSERIAL PRIMARY KEY,
    pattern_id  BIGINT NOT NULL REFERENCES public.cutting_patterns(id) ON DELETE CASCADE,
    sequence    INTEGER NOT NULL,
    part_id     BIGINT,
    label       VARCHAR(200) NOT NULL DEFAULT '',
    length_mm   NUMERIC(15,4) NOT NULL,
    offset_mm   NUMERIC(15,4) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_cutting_pattern_placements_pattern ON public.cutting_pattern_placements(pattern_id);

COMMIT;
