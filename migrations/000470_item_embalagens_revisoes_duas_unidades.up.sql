-- Último pedaço do cadastro de item: GTIN por embalagem, revisão com vigência,
-- estoque em duas unidades e os campos que faltavam à NF-e.
ALTER TABLE items
    ADD COLUMN description_en               VARCHAR(255),
    ADD COLUMN warehouse_secondary_uom      VARCHAR(20),
    ADD COLUMN warehouse_secondary_factor   NUMERIC(18,6),
    ADD COLUMN accounting_ex_tipi           VARCHAR(3),
    ADD COLUMN accounting_relevant_scale    BOOLEAN,
    ADD COLUMN accounting_manufacturer_cnpj VARCHAR(14),
    ADD COLUMN accounting_fci_number        VARCHAR(36);

ALTER TABLE items
    ADD CONSTRAINT chk_items_secondary_uom CHECK (
        (warehouse_secondary_uom IS NULL) = (warehouse_secondary_factor IS NULL)
        AND (warehouse_secondary_factor IS NULL OR warehouse_secondary_factor > 0)),
    ADD CONSTRAINT chk_items_ex_tipi CHECK (accounting_ex_tipi IS NULL OR accounting_ex_tipi ~ '^[0-9]{2,3}$'),
    ADD CONSTRAINT chk_items_manufacturer_cnpj CHECK (accounting_manufacturer_cnpj IS NULL OR accounting_manufacturer_cnpj ~ '^[0-9]{14}$');

-- Cada embalagem tem o seu código: a caixa de 100 não é a unidade. Um GTIN aponta
-- para um item só na empresa.
CREATE TABLE item_gtins (
    id               BIGSERIAL PRIMARY KEY,
    enterprise_id    BIGINT NOT NULL REFERENCES enterprise(id),
    item_code        BIGINT NOT NULL REFERENCES items(code) ON DELETE CASCADE,
    gtin             VARCHAR(14) NOT NULL CHECK (gtin ~ '^([0-9]{8}|[0-9]{12,14})$'),
    uom              VARCHAR(20) NOT NULL,
    package_quantity NUMERIC(18,6) NOT NULL DEFAULT 1 CHECK (package_quantity > 0),
    description      VARCHAR(120),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, gtin),
    UNIQUE (enterprise_id, item_code, uom)
);

-- Revisão com vigência. `items.revision` continua sendo a vigente, e a OF guarda
-- a revisão com que nasceu.
CREATE TABLE item_revisoes (
    id            BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    item_code     BIGINT NOT NULL REFERENCES items(code) ON DELETE CASCADE,
    revision      VARCHAR(20) NOT NULL,
    valid_from    DATE NOT NULL,
    valid_until   DATE,
    reason        VARCHAR(500) NOT NULL,
    created_by    UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, item_code, revision),
    CHECK (valid_until IS NULL OR valid_until >= valid_from)
);
CREATE INDEX idx_item_revisoes_vigencia ON item_revisoes (enterprise_id, item_code, valid_from DESC);

ALTER TABLE production_orders ADD COLUMN item_revision VARCHAR(20);

CREATE OR REPLACE FUNCTION production_order_revisao_do_item() RETURNS trigger AS $$
BEGIN
    IF NEW.item_revision IS NULL AND NEW.item_code IS NOT NULL THEN
        -- A vigência decide: a revisão futura passa a valer na data, sem depender
        -- de alguém atualizar items.revision.
        SELECT COALESCE((SELECT r.revision FROM item_revisoes r
                          WHERE r.item_code = NEW.item_code AND r.valid_from <= CURRENT_DATE
                            AND (r.valid_until IS NULL OR r.valid_until >= CURRENT_DATE)
                          ORDER BY r.valid_from DESC LIMIT 1), i.revision)
          INTO NEW.item_revision FROM items i WHERE i.code = NEW.item_code;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_production_order_revisao BEFORE INSERT ON production_orders
    FOR EACH ROW EXECUTE FUNCTION production_order_revisao_do_item();

-- Segunda unidade: o peso real da chapa que o estoque guarda em unidades. Sem
-- valor informado vale o fator nominal do cadastro; o estorno devolve o que o
-- original movimentou, para o saldo em peso voltar exato.
ALTER TABLE stock_movements ADD COLUMN secondary_quantity NUMERIC(18,6);

CREATE OR REPLACE FUNCTION stock_movement_segunda_unidade() RETURNS trigger AS $$
DECLARE
    fator NUMERIC;
BEGIN
    IF NEW.secondary_quantity IS NOT NULL THEN
        RETURN NEW;
    END IF;
    IF NEW.reverses_movement_id IS NOT NULL THEN
        -- Mesmo tipo (ADJUSTMENT) precisa do sinal contrário; IN e OUT já se anulam pelo tipo.
        SELECT CASE WHEN m.movement_type = NEW.movement_type THEN -m.secondary_quantity ELSE m.secondary_quantity END
          INTO NEW.secondary_quantity FROM stock_movements m WHERE m.id = NEW.reverses_movement_id;
        RETURN NEW;
    END IF;
    SELECT i.warehouse_secondary_factor INTO fator FROM items i WHERE i.code = NEW.item_code;
    IF fator IS NOT NULL THEN
        NEW.secondary_quantity := NEW.quantity * fator;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_stock_movement_segunda_unidade BEFORE INSERT ON stock_movements
    FOR EACH ROW EXECUTE FUNCTION stock_movement_segunda_unidade();

SELECT instalar_conferencia_de_empresa();
