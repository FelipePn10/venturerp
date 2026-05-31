BEGIN;

-- ─── Cadastro de Conversões por Item (UM compra ↔ estoque) ─────────────────────
-- factor: 1 from_uom = <factor> to_uom. Conversões inversas são derivadas (1/factor)
-- quando não houver cadastro direto.
CREATE TABLE IF NOT EXISTS item_unit_conversions (
    id          BIGSERIAL PRIMARY KEY,
    item_code   BIGINT        NOT NULL,
    from_uom    VARCHAR(10)   NOT NULL,
    to_uom      VARCHAR(10)   NOT NULL,
    factor      NUMERIC(18,6) NOT NULL CHECK (factor > 0),
    is_active   BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by  UUID          NOT NULL,
    CONSTRAINT ck_item_conversion_distinct_uom CHECK (from_uom <> to_uom),
    UNIQUE (item_code, from_uom, to_uom)
);

CREATE INDEX IF NOT EXISTS ix_item_unit_conversions_item ON item_unit_conversions(item_code);

COMMIT;
