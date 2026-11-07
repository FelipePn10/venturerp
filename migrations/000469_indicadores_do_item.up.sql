-- Indicadores do item que passam a valer, e o que faltava à metalúrgica e à
-- moveleira. A auditoria da VENT0200 achou indicadores que gravavam e ninguém lia.
ALTER TABLE items
    ADD COLUMN is_scrap                           BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN engineering_requires_drawing       BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN engineering_cutting_grain          VARCHAR(6) NOT NULL DEFAULT 'NONE',
    ADD COLUMN engineering_cutting_no_rotation    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN supplies_requires_certificate      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN commercial_exclusive_customer_code BIGINT,
    ADD COLUMN commercial_volumes_per_unit        INTEGER;

ALTER TABLE items
    ADD CONSTRAINT chk_items_cutting_grain CHECK (engineering_cutting_grain IN ('NONE', 'LENGTH', 'WIDTH')),
    ADD CONSTRAINT chk_items_volumes_per_unit CHECK (commercial_volumes_per_unit IS NULL OR commercial_volumes_per_unit > 0);

CREATE INDEX idx_items_exclusive_customer ON items (enterprise_id, commercial_exclusive_customer_code)
    WHERE commercial_exclusive_customer_code IS NOT NULL;

-- A descrição da nota sempre foi livre. O indicador passa a valer, e item antigo
-- não pode mudar de comportamento sem ninguém ter decidido: fica liberado.
UPDATE items SET commercial_allow_billing_description_change = TRUE;

-- Ferramenta do almoxarifado ligada à ficha de vida útil.
ALTER TABLE tools ADD COLUMN item_code BIGINT;
CREATE INDEX idx_tools_item ON tools (enterprise_id, item_code) WHERE item_code IS NOT NULL;

-- Grupo e modificador pertencem à empresa, mas o código do grupo era único no
-- banco inteiro: uma empresa impedia a outra de usar o código 10. A chave do item
-- passa a ser a da empresa.
ALTER TABLE items DROP CONSTRAINT fk_items_accounting_inventory_group;
ALTER TABLE groups DROP CONSTRAINT groups_code_key;
ALTER TABLE items ADD CONSTRAINT fk_items_accounting_inventory_group
    FOREIGN KEY (enterprise_id, accounting_inventory_group_code) REFERENCES groups (enterprise_id, code);

ALTER TABLE groups ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE modifier ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

SELECT instalar_conferencia_de_empresa();
