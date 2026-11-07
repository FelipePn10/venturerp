-- Hierarquia de embalagens: a caixa contém unidades, o palete contém caixas. O
-- palete costuma não ter código de barras, então o GTIN passa a ser opcional.
-- package_quantity continua sendo o total de unidades do item na embalagem.
ALTER TABLE item_gtins ALTER COLUMN gtin DROP NOT NULL;
ALTER TABLE item_gtins ADD COLUMN parent_uom VARCHAR(20);
ALTER TABLE item_gtins ADD COLUMN parent_quantity NUMERIC(18,6) CHECK (parent_quantity > 0);

-- Atributos livres: o que o PDM não padroniza (acabamento, tonalidade, norma do
-- cliente). Entram na busca do item.
CREATE TABLE item_atributos_livres (
    id            BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    item_code     BIGINT NOT NULL REFERENCES items(code) ON DELETE CASCADE,
    name          VARCHAR(60) NOT NULL,
    value         VARCHAR(200) NOT NULL,
    position      INTEGER NOT NULL DEFAULT 0,
    UNIQUE (enterprise_id, item_code, name)
);
CREATE INDEX idx_item_atributos_livres_busca ON item_atributos_livres (enterprise_id, name);

SELECT instalar_conferencia_de_empresa();
