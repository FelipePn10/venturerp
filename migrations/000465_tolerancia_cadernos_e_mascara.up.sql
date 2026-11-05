-- Limite de ajuste de inventário por perfil de acesso (SAP: grupo de tolerância).
-- Sem linha para o perfil, valem só as regras do inventário.
CREATE TABLE stock_tolerance_profiles (
    enterprise_id      bigint        NOT NULL,
    perfil_id          bigint        NOT NULL,
    max_document_value numeric(18,2) CHECK (max_document_value IS NULL OR max_document_value >= 0),
    max_line_value     numeric(18,2) CHECK (max_line_value IS NULL OR max_line_value >= 0),
    updated_by         uuid          NOT NULL REFERENCES users(id),
    updated_at         timestamptz   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, perfil_id),
    FOREIGN KEY (enterprise_id, perfil_id) REFERENCES acesso_perfis(enterprise_id, id) ON DELETE CASCADE,
    CHECK (max_document_value IS NOT NULL OR max_line_value IS NOT NULL)
);

-- Cadernos de contagem: a linha do inventário pertence a um caderno impresso.
ALTER TABLE physical_inventory_items ADD COLUMN notebook_number integer;
CREATE INDEX idx_physical_inventory_items_caderno ON physical_inventory_items (inventory_id, notebook_number) WHERE notebook_number IS NOT NULL;

CREATE TABLE physical_inventory_notebooks (
    inventory_id bigint      NOT NULL REFERENCES physical_inventories(id) ON DELETE CASCADE,
    number       integer     NOT NULL CHECK (number > 0),
    warehouse_id bigint      NOT NULL REFERENCES warehouse(id),
    print_count  integer     NOT NULL DEFAULT 0,
    printed_at   timestamptz,
    printed_by   uuid        REFERENCES users(id),
    created_at   timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY (inventory_id, number)
);

-- Política de reposição por máscara: vazio é o item sem configuração.
ALTER TABLE stock_replenishment_policies ADD COLUMN mask varchar(200) NOT NULL DEFAULT '';
ALTER TABLE stock_replenishment_policies DROP CONSTRAINT stock_replenishment_policies_enterprise_id_item_code_wareho_key;
ALTER TABLE stock_replenishment_policies ADD CONSTRAINT uq_politica_reposicao UNIQUE (enterprise_id, item_code, mask, warehouse_id);

-- A requisição de compra leva a máscara até o pedido.
ALTER TABLE purchase_requisition_items ADD COLUMN mask varchar(200) NOT NULL DEFAULT '';
