-- Catálogo de unidades, sinônimos, conversão em níveis, vigência e histórico.
--
-- Até aqui a unidade era uma lista fechada de 15 valores (enum no banco e no
-- Go). Material real chega em BARRA, CHAPA, ROLO, PCT, G… e a NF-e traz a
-- unidade comercial como TEXTO LIVRE ("UND", "PÇ", "KGS"). Nada disso casava
-- com o cadastro: a nota ficava pendente por uma diferença que não existe, e a
-- tela de conversões nem deixava cadastrar a unidade que o fornecedor usa.
--
-- O que muda:
--   • units_of_measure        — catálogo por empresa, com dimensão (contagem,
--                               comprimento, massa…) e código fiscal (NF-e/SPED
--                               aceitam no máximo 6 caracteres).
--   • unit_of_measure_aliases — sinônimos: o texto que vem no XML → unidade do
--                               catálogo; geral da empresa ou só de um fornecedor.
--   • global_unit_conversions — passa a ser a "conversão padrão" da empresa
--                               (M→MM, KG→G…), semeada, com fração exata.
--   • item_unit_conversions   — fração exata (N de = D para), vigência,
--                               origem, observação; versões sem sobreposição.
--   • item_lot_unit_conversions — fator real de um lote (peso real da chapa).
--   • item_unit_conversion_events — histórico/auditoria de toda alteração.
--   • fiscal_entry_items      — guarda a unidade tributável do XML (uTrib/qTrib)
--                               e DE ONDE veio o fator aplicado na linha.
--   • items / item_structures — unidade deixa de ser enum e passa a ser texto
--                               validado contra o catálogo.

BEGIN;

-- ─── Catálogo ────────────────────────────────────────────────────────────────
CREATE TABLE units_of_measure (
    id            BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT       NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    code          VARCHAR(10)  NOT NULL CHECK (code ~ '^[A-Z0-9]{1,10}$'),
    description   VARCHAR(80)  NOT NULL,
    dimension     VARCHAR(12)  NOT NULL DEFAULT 'OUTRA'
                  CHECK (dimension IN ('CONTAGEM','COMPRIMENTO','MASSA','AREA','VOLUME','TEMPO','OUTRA')),
    fiscal_code   VARCHAR(6)   NOT NULL,
    decimals      SMALLINT     NOT NULL DEFAULT 4 CHECK (decimals BETWEEN 0 AND 6),
    is_system     BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_by    UUID,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_units_of_measure UNIQUE (enterprise_id, code)
);
COMMENT ON TABLE units_of_measure IS
    'Catálogo de unidades de medida por empresa. Substitui o enum unit_of_measurement_enum (15 valores fixos).';
COMMENT ON COLUMN units_of_measure.fiscal_code IS
    'Código da unidade na NF-e/SPED (uCom/0190 aceitam até 6 caracteres): TONELADA → T.';

CREATE TABLE unit_of_measure_aliases (
    id            BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT      NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    alias         VARCHAR(10) NOT NULL CHECK (alias <> ''),
    unit_code     VARCHAR(10) NOT NULL,
    supplier_code BIGINT,
    is_system     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_by    UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_uom_alias_unit FOREIGN KEY (enterprise_id, unit_code)
        REFERENCES units_of_measure(enterprise_id, code) ON UPDATE CASCADE
);
CREATE UNIQUE INDEX uq_uom_alias ON unit_of_measure_aliases(enterprise_id, alias, COALESCE(supplier_code, 0));
COMMENT ON TABLE unit_of_measure_aliases IS
    'Sinônimos de unidade: texto recebido (XML da NF-e, planilha) → unidade do catálogo. supplier_code nulo = vale para todos os fornecedores da empresa.';

-- ─── Conversão padrão da empresa (mesma dimensão) ───────────────────────────
ALTER TABLE global_unit_conversions
    ALTER COLUMN factor TYPE NUMERIC(20,10),
    ADD COLUMN from_quantity NUMERIC(20,10),
    ADD COLUMN to_quantity   NUMERIC(20,10),
    ADD COLUMN is_system     BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE global_unit_conversions SET from_quantity = 1, to_quantity = factor WHERE from_quantity IS NULL;
ALTER TABLE global_unit_conversions
    ALTER COLUMN from_quantity SET NOT NULL, ALTER COLUMN from_quantity SET DEFAULT 1,
    ALTER COLUMN to_quantity SET NOT NULL,
    ADD CONSTRAINT chk_global_conv_quantities CHECK (from_quantity > 0 AND to_quantity > 0);

-- ─── Semeadura (empresas atuais e futuras) ──────────────────────────────────
CREATE OR REPLACE FUNCTION erp_semear_unidades_de_medida(p_empresa BIGINT) RETURNS VOID AS $$
BEGIN
    INSERT INTO units_of_measure (enterprise_id, code, description, dimension, fiscal_code, decimals, is_system)
    VALUES
        (p_empresa, 'UN',         'Unidade',            'CONTAGEM',    'UN',  0, TRUE),
        (p_empresa, 'PC',         'Peça',               'CONTAGEM',    'PC',  0, TRUE),
        (p_empresa, 'PAR',        'Par',                'CONTAGEM',    'PAR', 0, TRUE),
        (p_empresa, 'CX',         'Caixa',              'CONTAGEM',    'CX',  0, TRUE),
        (p_empresa, 'PCT',        'Pacote',             'CONTAGEM',    'PCT', 0, TRUE),
        (p_empresa, 'FD',         'Fardo',              'CONTAGEM',    'FD',  0, TRUE),
        (p_empresa, 'JG',         'Jogo',               'CONTAGEM',    'JG',  0, TRUE),
        (p_empresa, 'KIT',        'Kit',                'CONTAGEM',    'KIT', 0, TRUE),
        (p_empresa, 'DZ',         'Dúzia',              'CONTAGEM',    'DZ',  0, TRUE),
        (p_empresa, 'CENTO',      'Cento',              'CONTAGEM',    'CENTO', 0, TRUE),
        (p_empresa, 'MILHEI',     'Milheiro',           'CONTAGEM',    'MILHEI', 0, TRUE),
        (p_empresa, 'BARRA',      'Barra',              'CONTAGEM',    'BARRA', 0, TRUE),
        (p_empresa, 'CHAPA',      'Chapa',              'CONTAGEM',    'CHAPA', 0, TRUE),
        (p_empresa, 'ROLO',       'Rolo',               'CONTAGEM',    'ROLO', 0, TRUE),
        (p_empresa, 'BOBINA',     'Bobina',             'CONTAGEM',    'BOBINA', 0, TRUE),
        (p_empresa, 'MM',         'Milímetro',          'COMPRIMENTO', 'MM',  4, TRUE),
        (p_empresa, 'CM',         'Centímetro',         'COMPRIMENTO', 'CM',  4, TRUE),
        (p_empresa, 'M',          'Metro',              'COMPRIMENTO', 'M',   4, TRUE),
        (p_empresa, 'IN',         'Polegada',           'COMPRIMENTO', 'IN',  4, TRUE),
        (p_empresa, 'MICROMETRO', 'Micrômetro',         'COMPRIMENTO', 'UM',  4, TRUE),
        (p_empresa, 'G',          'Grama',              'MASSA',       'G',   4, TRUE),
        (p_empresa, 'KG',         'Quilograma',         'MASSA',       'KG',  4, TRUE),
        (p_empresa, 'TONELADA',   'Tonelada',           'MASSA',       'T',   4, TRUE),
        (p_empresa, 'M2',         'Metro quadrado',     'AREA',        'M2',  4, TRUE),
        (p_empresa, 'ML',         'Mililitro',          'VOLUME',      'ML',  4, TRUE),
        (p_empresa, 'L',          'Litro',              'VOLUME',      'L',   4, TRUE),
        (p_empresa, 'GL',         'Galão (3,785 L)',    'VOLUME',      'GL',  4, TRUE),
        (p_empresa, 'M3',         'Metro cúbico',       'VOLUME',      'M3',  4, TRUE),
        (p_empresa, 'H',          'Hora',               'TEMPO',       'H',   4, TRUE)
    ON CONFLICT (enterprise_id, code) DO NOTHING;

    INSERT INTO unit_of_measure_aliases (enterprise_id, alias, unit_code, is_system)
    SELECT p_empresa, a.alias, a.unit_code, TRUE
      FROM (VALUES
        ('UND','UN'),('UNID','UN'),('UNIDAD','UN'),('UNI','UN'),('UNIT','UN'),('EA','UN'),
        ('PCA','PC'),('PECA','PC'),('PCS','PC'),('PEC','PC'),
        ('PR','PAR'),('PARES','PAR'),
        ('CXA','CX'),('CAIXA','CX'),('CXS','CX'),
        ('PCTE','PCT'),('PACOTE','PCT'),('PK','PCT'),
        ('FARDO','FD'),('JOGO','JG'),('DUZIA','DZ'),('MIL','MILHEI'),('MLH','MILHEI'),
        ('BR','BARRA'),('BAR','BARRA'),('BRR','BARRA'),('VARA','BARRA'),
        ('CH','CHAPA'),('CHP','CHAPA'),
        ('RL','ROLO'),('RLO','ROLO'),('BOB','BOBINA'),
        ('MT','M'),('MTS','M'),('MTR','M'),('METRO','M'),
        ('MILIM','MM'),('CENTIM','CM'),('POL','IN'),
        ('MICRON','MICROMETRO'),
        ('KGS','KG'),('KILO','KG'),('QUILO','KG'),('KGR','KG'),
        ('GR','G'),('GRS','G'),('GRAMA','G'),
        ('TON','TONELADA'),('TN','TONELADA'),('T','TONELADA'),('TL','TONELADA'),
        ('MQ','M2'),('MTQ','M2'),
        ('LT','L'),('LTS','L'),('LITRO','L'),
        ('GAL','GL'),('GALAO','GL'),
        ('MTC','M3'),
        ('HR','H'),('HRS','H'),('HORA','H')
      ) AS a(alias, unit_code)
     WHERE EXISTS (SELECT 1 FROM units_of_measure u WHERE u.enterprise_id = p_empresa AND u.code = a.unit_code)
    ON CONFLICT DO NOTHING;

    INSERT INTO global_unit_conversions (enterprise_id, from_uom, to_uom, factor, from_quantity, to_quantity, is_system, created_by)
    SELECT p_empresa, c.de, c.para, round(c.qpara / c.qde, 10), c.qde, c.qpara, TRUE,
           '00000000-0000-0000-0000-000000000000'::uuid
      FROM (VALUES
        ('M','MM',1::numeric,1000::numeric),('M','CM',1,100),('CM','MM',1,10),('IN','MM',1,25.4),
        ('MM','MICROMETRO',1,1000),
        ('KG','G',1,1000),('TONELADA','KG',1,1000),
        ('L','ML',1,1000),('M3','L',1,1000),('GL','L',1,3.785411784),
        ('DZ','UN',1,12),('CENTO','UN',1,100),('MILHEI','UN',1,1000)
      ) AS c(de, para, qde, qpara)
     WHERE NOT EXISTS (
        SELECT 1 FROM global_unit_conversions g
         WHERE g.enterprise_id = p_empresa
           AND ((g.from_uom = c.de AND g.to_uom = c.para) OR (g.from_uom = c.para AND g.to_uom = c.de)));
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION erp_semear_unidades_nova_empresa() RETURNS TRIGGER AS $$
BEGIN
    PERFORM erp_semear_unidades_de_medida(NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_enterprise_units_of_measure
    AFTER INSERT ON enterprise FOR EACH ROW EXECUTE FUNCTION erp_semear_unidades_nova_empresa();

SELECT erp_semear_unidades_de_medida(id) FROM enterprise;

-- Unidade que já está em uso e não faz parte da semeadura entra no catálogo
-- como "OUTRA": nenhum cadastro existente fica órfão.
INSERT INTO units_of_measure (enterprise_id, code, description, dimension, fiscal_code, decimals)
SELECT DISTINCT u.enterprise_id, u.code, u.code, 'OUTRA', left(u.code, 6), 4
  FROM (
    SELECT i.enterprise_id, upper(trim(i.warehouse_unit_of_measurement::text)) AS code FROM items i
    UNION SELECT i.enterprise_id, upper(trim(i.supplies_purchase_uom)) FROM items i
    UNION SELECT i.enterprise_id, upper(trim(i.accounting_sale_unit_of_measurement)) FROM items i
    UNION SELECT i.enterprise_id, upper(trim(i.accounting_purchase_unit_of_measurement)) FROM items i
    UNION SELECT i.enterprise_id, upper(trim(i.warehouse_secondary_uom)) FROM items i
    UNION SELECT i.enterprise_id, upper(trim(c.from_uom)) FROM item_unit_conversions c JOIN items i ON i.code = c.item_code
    UNION SELECT i.enterprise_id, upper(trim(c.to_uom))   FROM item_unit_conversions c JOIN items i ON i.code = c.item_code
    UNION SELECT g.enterprise_id, upper(trim(g.from_uom)) FROM global_unit_conversions g
    UNION SELECT g.enterprise_id, upper(trim(g.to_uom))   FROM global_unit_conversions g
    UNION SELECT i.enterprise_id, upper(trim(s.unit_of_measurement::text)) FROM item_structures s JOIN items i ON i.code = s.child_code
  ) u
 WHERE u.code ~ '^[A-Z0-9]{1,10}$'
ON CONFLICT (enterprise_id, code) DO NOTHING;

-- ─── Unidade deixa de ser enum ──────────────────────────────────────────────
ALTER TABLE items ALTER COLUMN warehouse_unit_of_measurement DROP DEFAULT;
ALTER TABLE items ALTER COLUMN warehouse_unit_of_measurement TYPE VARCHAR(10)
    USING upper(trim(warehouse_unit_of_measurement::text));
ALTER TABLE items ALTER COLUMN warehouse_unit_of_measurement SET DEFAULT 'UN';
ALTER TABLE items ADD CONSTRAINT fk_items_warehouse_uom
    FOREIGN KEY (enterprise_id, warehouse_unit_of_measurement)
    REFERENCES units_of_measure(enterprise_id, code) ON UPDATE CASCADE;

ALTER TABLE item_structures ALTER COLUMN unit_of_measurement DROP DEFAULT;
ALTER TABLE item_structures ALTER COLUMN unit_of_measurement TYPE VARCHAR(10)
    USING upper(trim(unit_of_measurement::text));
ALTER TABLE item_structures ALTER COLUMN unit_of_measurement SET DEFAULT 'UN';

DROP TYPE IF EXISTS unit_of_measurement_enum;

-- ─── Conversão do item: fração exata, vigência, origem ──────────────────────
ALTER TABLE item_unit_conversions
    ADD COLUMN from_quantity NUMERIC(20,10),
    ADD COLUMN to_quantity   NUMERIC(20,10),
    ADD COLUMN valid_from    DATE NOT NULL DEFAULT DATE '2000-01-01',
    ADD COLUMN valid_to      DATE,
    ADD COLUMN origin        VARCHAR(20) NOT NULL DEFAULT 'MANUAL'
               CHECK (origin IN ('MANUAL','NOTA_FISCAL','PEDIDO_COMPRA','IMPORTACAO')),
    ADD COLUMN notes         TEXT,
    ADD COLUMN updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN updated_by    UUID;
UPDATE item_unit_conversions SET from_quantity = 1, to_quantity = factor WHERE from_quantity IS NULL;
UPDATE item_unit_conversions SET from_uom = upper(trim(from_uom)), to_uom = upper(trim(to_uom));
ALTER TABLE item_unit_conversions
    ALTER COLUMN from_quantity SET NOT NULL, ALTER COLUMN from_quantity SET DEFAULT 1,
    ALTER COLUMN to_quantity SET NOT NULL,
    ADD CONSTRAINT chk_item_conv_quantities CHECK (from_quantity > 0 AND to_quantity > 0),
    ADD CONSTRAINT chk_item_conv_validity CHECK (valid_to IS NULL OR valid_to >= valid_from);

-- Várias versões do mesmo par convivem (uma por vigência); só as ativas contam.
DROP INDEX IF EXISTS uq_item_unit_conversions_config;
CREATE UNIQUE INDEX uq_item_unit_conversions_version
    ON item_unit_conversions(item_code, mask, from_uom, to_uom, valid_from) WHERE is_active;
CREATE INDEX ix_item_unit_conversions_lookup
    ON item_unit_conversions(item_code, is_active, valid_from);

-- Duas versões ativas do mesmo fato não podem se sobrepor no tempo. "1 UN = 23 KG"
-- e "1 KG = 0,05 UN" são o MESMO fato (um é a inversa do outro): também contam.
CREATE OR REPLACE FUNCTION erp_item_conversion_sem_sobreposicao() RETURNS TRIGGER AS $$
BEGIN
    IF NOT NEW.is_active THEN
        RETURN NEW;
    END IF;
    IF EXISTS (
        SELECT 1 FROM item_unit_conversions c
         WHERE c.is_active AND c.id <> NEW.id
           AND c.item_code = NEW.item_code AND c.mask = NEW.mask
           AND ((c.from_uom = NEW.from_uom AND c.to_uom = NEW.to_uom)
             OR (c.from_uom = NEW.to_uom AND c.to_uom = NEW.from_uom))
           AND daterange(c.valid_from, c.valid_to, '[]') && daterange(NEW.valid_from, NEW.valid_to, '[]')
    ) THEN
        RAISE EXCEPTION 'já existe conversão % ↔ % vigente no mesmo período para o item %',
            NEW.from_uom, NEW.to_uom, NEW.item_code USING ERRCODE = '23P01';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_item_conversion_sem_sobreposicao
    BEFORE INSERT OR UPDATE ON item_unit_conversions
    FOR EACH ROW EXECUTE FUNCTION erp_item_conversion_sem_sobreposicao();

-- ─── Fator real do lote ─────────────────────────────────────────────────────
CREATE TABLE item_lot_unit_conversions (
    id            BIGSERIAL PRIMARY KEY,
    item_code     BIGINT        NOT NULL,
    lot           VARCHAR(50)   NOT NULL,
    from_uom      VARCHAR(10)   NOT NULL,
    to_uom        VARCHAR(10)   NOT NULL,
    from_quantity NUMERIC(20,10) NOT NULL CHECK (from_quantity > 0),
    to_quantity   NUMERIC(20,10) NOT NULL CHECK (to_quantity > 0),
    factor        NUMERIC(20,10) NOT NULL CHECK (factor > 0),
    notes         TEXT,
    is_active     BOOLEAN       NOT NULL DEFAULT TRUE,
    created_by    UUID          NOT NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CHECK (from_uom <> to_uom)
);
CREATE UNIQUE INDEX uq_item_lot_unit_conversions
    ON item_lot_unit_conversions(item_code, lot, from_uom, to_uom) WHERE is_active;
COMMENT ON TABLE item_lot_unit_conversions IS
    'Fator medido de um lote (ex.: peso real da chapa da corrida X). Tem precedência sobre o fator do item. Empresa = a do item.';

-- ─── Histórico ──────────────────────────────────────────────────────────────
CREATE TABLE item_unit_conversion_events (
    id                BIGSERIAL PRIMARY KEY,
    item_code         BIGINT      NOT NULL,
    conversion_id     BIGINT,
    lot_conversion_id BIGINT,
    action            VARCHAR(20) NOT NULL
                      CHECK (action IN ('CRIADA','ALTERADA','NOVA_VIGENCIA','EXCLUIDA')),
    before_data       JSONB,
    after_data        JSONB,
    reason            TEXT,
    created_by        UUID        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ix_item_unit_conversion_events_item ON item_unit_conversion_events(item_code, created_at DESC);

-- ─── Nota de entrada: unidade tributável e origem do fator ──────────────────
ALTER TABLE fiscal_entry_items
    ADD COLUMN unidade_tributavel    VARCHAR(10),
    ADD COLUMN quantidade_tributavel NUMERIC(18,6),
    ADD COLUMN unidade_normalizada   VARCHAR(10),
    ADD COLUMN origem_conversao      VARCHAR(20)
        CHECK (origem_conversao IN ('MESMA_UNIDADE','SINONIMO','VINCULO_FORNECEDOR','PEDIDO_COMPRA',
                                    'CADASTRO_ITEM','LOTE','PADRAO','XML_TRIBUTAVEL','MANUAL')),
    ADD COLUMN caminho_conversao     TEXT;
-- ─── Fração × fator: quem grava só o fator continua funcionando ─────────────
-- O fator é derivado da fração (to/from). Quem ainda grava só `factor`
-- (integrações, scripts, código antigo) recebe a fração 1 : fator; quem grava
-- a fração tem o fator recalculado. O nome começa com "aa" para rodar antes do
-- gatilho de sobreposição (gatilhos BEFORE rodam em ordem alfabética).
CREATE OR REPLACE FUNCTION erp_conversao_fracao_e_fator() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.factor IS DISTINCT FROM OLD.factor
       AND NEW.from_quantity IS NOT DISTINCT FROM OLD.from_quantity
       AND NEW.to_quantity IS NOT DISTINCT FROM OLD.to_quantity THEN
        NEW.from_quantity := 1;
        NEW.to_quantity := NEW.factor;
    END IF;
    IF NEW.from_quantity IS NULL OR NEW.to_quantity IS NULL THEN
        NEW.from_quantity := COALESCE(NEW.from_quantity, 1);
        NEW.to_quantity := NEW.factor * NEW.from_quantity;
    END IF;
    IF NEW.from_quantity > 0 AND NEW.to_quantity IS NOT NULL THEN
        NEW.factor := round(NEW.to_quantity / NEW.from_quantity, 10);
    END IF;
    RETURN NEW;
END $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_aa_item_conversion_fracao BEFORE INSERT OR UPDATE ON item_unit_conversions
    FOR EACH ROW EXECUTE FUNCTION erp_conversao_fracao_e_fator();
CREATE TRIGGER trg_aa_global_conversion_fracao BEFORE INSERT OR UPDATE ON global_unit_conversions
    FOR EACH ROW EXECUTE FUNCTION erp_conversao_fracao_e_fator();

-- O sinônimo por fornecedor aponta para fornecedor da mesma empresa.
SELECT instalar_conferencia_de_empresa();

COMMENT ON COLUMN fiscal_entry_items.origem_conversao IS
    'De onde veio o fator_conversao da linha: vínculo do fornecedor, pedido, cadastro do item (VSUP0110), conversão padrão, lote, unidade tributável do próprio XML, sinônimo ou digitado.';

COMMIT;
