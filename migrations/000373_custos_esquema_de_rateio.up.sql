-- Evolução do custo-padrão: componentes de custo, esquema de rateio de
-- indiretos, histórico por versão e isolamento por empresa.
--
-- ── O que faltava ────────────────────────────────────────────────────────────
-- O motor já fazia o essencial: desce a estrutura, aplica perda, credita
-- co-produto, escolhe substituto, cobra cada operação na taxa do SEU centro de
-- trabalho separando hora-máquina de hora-homem, amortiza setup pelo lote e
-- reconhece operação de terceiro. O que faltava é o que os ERPs grandes chamam de
-- esquema de cálculo:
--
--   1. `overhead_cost` era gravado SEMPRE ZERO. A coluna existia, o comentário do
--      código dizia "currently 0 unless configured" — e não havia como
--      configurar. Custo indireto (energia, depreciação, supervisão, aluguel do
--      galpão) simplesmente não entrava no custo do produto. É a maior diferença
--      contra SAP (CO-PC, esquema de cálculo com taxas de sobrecarga), Oracle
--      (overhead rates por resource/departamento), TOTVS (taxas de CIF por centro
--      de custo) e Focco (despesas indiretas na formação do preço).
--
--   2. O custo saía em dois números — material e "operação". Quem analisa precisa
--      saber QUANTO é setup, quanto é máquina, quanto é mão de obra e quanto é
--      serviço de terceiro, porque cada um se ataca de um jeito diferente: setup
--      alto pede lote maior, máquina alta pede outro recurso, terceiro alto pede
--      internalizar. É a estrutura de componentes de custo do SAP.
--
--   3. Não havia separação entre nível próprio e níveis inferiores
--      (this level / lower level). Sem ela, um aumento no custo do produto não diz
--      se veio da fábrica ou do que se comprou.
--
--   4. Toda apuração sobrescrevia a anterior. Sem histórico não se responde
--      "por que o custo subiu 12% este mês".
--
-- ── Isolamento por empresa ───────────────────────────────────────────────────
-- `item_standard_costs`, `item_purchase_costs` e `cost_rollup_log` não têm coluna
-- de empresa. NÃO acrescentamos: `items.code` é único global e cada item já
-- pertence a uma empresa, então a empresa do custo é derivável do item e uma
-- coluna denormalizada poderia discordar de `items.enterprise_id` — um custo
-- atribuído à empresa errada é pior que nenhum. A posse passa a ser conferida no
-- repositório contra `items.enterprise_id`. As tabelas NOVAS desta migração já
-- nascem com `enterprise_id`, porque não pendem de item.

BEGIN;

-- ── 1. Componentes de custo no custo-padrão do item ─────────────────────────
ALTER TABLE item_standard_costs
    ADD COLUMN IF NOT EXISTS setup_cost       NUMERIC(18,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS machine_cost     NUMERIC(18,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS subcontract_cost NUMERIC(18,6) NOT NULL DEFAULT 0,
    -- Nível próprio × níveis inferiores: "o que ESTA fábrica agrega" contra "o
    -- que veio pronto de baixo". É o corte que diz onde atacar o custo.
    ADD COLUMN IF NOT EXISTS own_level_cost   NUMERIC(18,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS lower_level_cost NUMERIC(18,6) NOT NULL DEFAULT 0,
    -- O lote muda o custo unitário (setup diluído); guardar o lote da apuração é
    -- o que impede comparar duas apurações que não são comparáveis.
    ADD COLUMN IF NOT EXISTS lot_size         NUMERIC(18,6) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS overhead_detail  JSONB;

ALTER TABLE cost_rollup_log
    ADD COLUMN IF NOT EXISTS setup_cost       NUMERIC(18,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS machine_cost     NUMERIC(18,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS subcontract_cost NUMERIC(18,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS lower_level_cost NUMERIC(18,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS quantity         NUMERIC(18,6) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS parent_code      BIGINT;

-- ── 2. Esquema de rateio de custos indiretos ao produto ─────────────────────
DO $$ BEGIN
    CREATE TYPE cost_overhead_base_enum AS ENUM (
        'MATERIAL',        -- sobre o material do item
        'SETUP',           -- sobre o custo de preparação
        'MAQUINA',         -- sobre a hora-máquina
        'MAO_DE_OBRA',     -- sobre a hora-homem
        'CONVERSAO',       -- sobre setup + máquina + mão de obra
        'SUBCONTRATACAO',  -- sobre o serviço de terceiro
        'TOTAL'            -- sobre tudo que veio antes
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE cost_overhead_method_enum AS ENUM (
        'PERCENTUAL',        -- taxa % sobre a base
        'VALOR_POR_HORA',    -- R$ por hora da base (máquina ou mão de obra)
        'VALOR_POR_UNIDADE'  -- R$ fixo por unidade produzida
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS cost_overhead_rules (
    id             BIGSERIAL PRIMARY KEY,
    enterprise_id  BIGINT NOT NULL REFERENCES enterprise(id),
    code           VARCHAR(20) NOT NULL,
    description    VARCHAR(120) NOT NULL,
    base           cost_overhead_base_enum NOT NULL,
    method         cost_overhead_method_enum NOT NULL,
    -- Percentual entra como fração (0.12 = 12%), igual ao resto do domínio
    -- fiscal e de perdas. Valor por hora/unidade entra em reais.
    rate           NUMERIC(18,6) NOT NULL,
    -- Escopo: nulo é "vale para tudo". Centro de trabalho permite energia caríssima
    -- só na usinagem; item permite tratar um produto específico.
    work_center_id BIGINT,
    item_code      BIGINT,
    -- Conta do plano e centro de custo de destino: é o que liga o rateio do
    -- produto à contabilidade, e o que a apuração usa para conferir se o indireto
    -- aplicado fecha com o indireto lançado.
    plano_contas_id BIGINT,
    centro_custo_id BIGINT,
    valid_from     DATE NOT NULL,
    valid_to       DATE,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    notes          TEXT,
    created_by     UUID NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT cost_overhead_rules_codigo_unico UNIQUE (enterprise_id, code),
    -- Taxa negativa seria crédito de indireto: não existe no esquema de cálculo,
    -- e passaria batido produzindo custo menor que o material.
    CONSTRAINT cost_overhead_rules_taxa_positiva CHECK (rate > 0),
    -- Percentual acima de 100% do próprio custo é quase sempre erro de digitação
    -- (12 em vez de 0,12). O teto de 10 (1000%) deixa passar caso extremo real e
    -- barra o dedo escorregado.
    CONSTRAINT cost_overhead_rules_percentual_plausivel
        CHECK (method <> 'PERCENTUAL' OR rate <= 10),
    -- Valor por hora só faz sentido sobre uma base medida em horas.
    CONSTRAINT cost_overhead_rules_hora_com_base_de_hora
        CHECK (method <> 'VALOR_POR_HORA' OR base IN ('MAQUINA','MAO_DE_OBRA','CONVERSAO','SETUP')),
    CONSTRAINT cost_overhead_rules_vigencia CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

CREATE INDEX IF NOT EXISTS idx_cost_overhead_rules_vigentes
    ON cost_overhead_rules (enterprise_id, valid_from DESC)
    WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_cost_overhead_rules_centro
    ON cost_overhead_rules (enterprise_id, work_center_id)
    WHERE work_center_id IS NOT NULL AND is_active;

-- ── 3. Histórico de custo-padrão por apuração ───────────────────────────────
-- Uma linha por apuração, nunca sobrescrita: é o que responde "por que o custo
-- subiu" comparando a apuração de hoje com a do mês passado, componente a
-- componente.
CREATE TABLE IF NOT EXISTS item_standard_cost_history (
    id               BIGSERIAL PRIMARY KEY,
    enterprise_id    BIGINT NOT NULL REFERENCES enterprise(id),
    item_code        BIGINT NOT NULL,
    mask             VARCHAR(60) NOT NULL DEFAULT '',
    lot_size         NUMERIC(18,6) NOT NULL DEFAULT 1,
    material_cost    NUMERIC(18,6) NOT NULL DEFAULT 0,
    setup_cost       NUMERIC(18,6) NOT NULL DEFAULT 0,
    machine_cost     NUMERIC(18,6) NOT NULL DEFAULT 0,
    labor_cost       NUMERIC(18,6) NOT NULL DEFAULT 0,
    subcontract_cost NUMERIC(18,6) NOT NULL DEFAULT 0,
    overhead_cost    NUMERIC(18,6) NOT NULL DEFAULT 0,
    own_level_cost   NUMERIC(18,6) NOT NULL DEFAULT 0,
    lower_level_cost NUMERIC(18,6) NOT NULL DEFAULT 0,
    total_cost       NUMERIC(18,6) NOT NULL DEFAULT 0,
    currency         VARCHAR(3) NOT NULL DEFAULT 'BRL',
    overhead_detail  JSONB,
    calculated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    calculated_by    UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_isc_history_item
    ON item_standard_cost_history (enterprise_id, item_code, mask, calculated_at DESC);

-- Histórico é fato consumado. Alterar uma apuração passada reescreveria a
-- explicação de uma variação já analisada.
CREATE OR REPLACE FUNCTION prevent_cost_history_mutation() RETURNS TRIGGER AS $$
BEGIN RAISE EXCEPTION 'histórico de custo-padrão é imutável'; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_cost_history_immutable ON item_standard_cost_history;
CREATE TRIGGER trg_cost_history_immutable
BEFORE UPDATE OR DELETE ON item_standard_cost_history
FOR EACH ROW EXECUTE FUNCTION prevent_cost_history_mutation();

COMMIT;
