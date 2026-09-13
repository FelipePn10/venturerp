-- Endereçamento inteligente (putaway/picking) e curva ABC calculada.
--
-- Três lacunas diante dos concorrentes:
--
-- 1. Os endereços não tinham atributo nenhum além de ativo/inativo. Sem zona,
--    capacidade e bloqueio não há como SUGERIR onde guardar: o conferente
--    decide de cabeça e o material se espalha.
-- 2. A classe ABC (`items.planning_abc_class`) era digitada à mão. Ninguém
--    revisa isso, então ela envelhece e deixa de refletir o consumo real.
-- 3. A frequência de contagem vinha de um intervalo digitado item a item
--    (`warehouse_cyclical_count_config`). O Oracle deriva da curva ABC — item A
--    conta mais vezes por ano que item C, que é o ponto da contagem cíclica.

ALTER TABLE manufacturing_warehouse_addresses
    ADD COLUMN IF NOT EXISTS zone            VARCHAR(40)  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS capacity        NUMERIC(18,6),
    ADD COLUMN IF NOT EXISTS is_blocked      BOOLEAN      NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS block_reason    VARCHAR(200),
    ADD COLUMN IF NOT EXISTS fixed_item_code BIGINT,
    ADD COLUMN IF NOT EXISTS pick_sequence   INTEGER      NOT NULL DEFAULT 0;

COMMENT ON COLUMN manufacturing_warehouse_addresses.capacity IS
    'Capacidade na unidade do item; nulo = sem limite declarado.';
COMMENT ON COLUMN manufacturing_warehouse_addresses.fixed_item_code IS
    'Endereço fixo de um item (o "fixed bin" do SAP): a guarda vai sempre para cá.';
COMMENT ON COLUMN manufacturing_warehouse_addresses.pick_sequence IS
    'Ordem física do endereço na rota de separação; 0 = sem rota definida. '
    'É o que evita o separador cruzar o galpão de um lado para o outro.';

CREATE INDEX IF NOT EXISTS idx_wh_addresses_zona
    ON manufacturing_warehouse_addresses (enterprise_id, warehouse_id, zone)
 WHERE is_active AND NOT is_blocked;
CREATE INDEX IF NOT EXISTS idx_wh_addresses_item_fixo
    ON manufacturing_warehouse_addresses (enterprise_id, fixed_item_code)
 WHERE fixed_item_code IS NOT NULL;

-- Política de contagem por classe ABC. Os padrões seguem a prática corrente:
-- item A quatro vezes por ano, B trimestral, C semestral.
CREATE TABLE IF NOT EXISTS stock_abc_count_policy (
    enterprise_id BIGINT      NOT NULL REFERENCES enterprise(id),
    abc_class     CHAR(1)     NOT NULL,
    days_interval INTEGER     NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, abc_class),
    CONSTRAINT chk_abc_class    CHECK (abc_class IN ('A','B','C')),
    CONSTRAINT chk_abc_interval CHECK (days_interval > 0)
);

-- Semeia a política das empresas que já existem. Numa instalação nova a tabela
-- `enterprise` ainda está vazia (as migrações rodam antes do seed), então nada é
-- inserido aqui — e é por isso que o agendamento da contagem NÃO pode depender
-- desta tabela estar preenchida: ele usa o padrão da classe quando não encontra
-- linha. Sem isso, empresa criada depois da migração ficaria com intervalo zero,
-- que significa "nunca contar".
INSERT INTO stock_abc_count_policy (enterprise_id, abc_class, days_interval)
SELECT e.id, v.classe, v.dias
  FROM enterprise e
 CROSS JOIN (VALUES ('A',90),('B',180),('C',365)) AS v(classe, dias)
    ON CONFLICT DO NOTHING;

-- Registro de como a curva foi apurada, para a tela explicar a classe em vez de
-- mostrar só a letra: "A porque responde por 74% do valor consumido".
ALTER TABLE items
    ADD COLUMN IF NOT EXISTS abc_consumption_value NUMERIC(18,4),
    ADD COLUMN IF NOT EXISTS abc_share_pct         NUMERIC(9,6),
    ADD COLUMN IF NOT EXISTS abc_calculated_at     TIMESTAMPTZ;

COMMENT ON COLUMN items.abc_consumption_value IS
    'Valor consumido na janela da última apuração da curva ABC.';
COMMENT ON COLUMN items.abc_share_pct IS
    'Participação acumulada do item no valor total consumido (0-100).';
