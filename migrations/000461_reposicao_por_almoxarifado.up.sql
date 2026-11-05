-- Fase D: reposição por almoxarifado e requisição com aprovação, reserva e
-- transferência. A ordem de reposição é a requisição do tipo TRANSFERENCIA: as
-- duas têm o mesmo ciclo (Oracle move order), só muda o que acontece na entrega.

-- Campo vazio herda o valor do cadastro do item.
CREATE TABLE stock_replenishment_policies (
    id                  BIGSERIAL     PRIMARY KEY,
    enterprise_id       BIGINT        NOT NULL REFERENCES enterprise(id),
    item_code           BIGINT        NOT NULL,
    warehouse_id        BIGINT        NOT NULL REFERENCES warehouse(id),
    minimum_qty         NUMERIC(18,6) CHECK (minimum_qty >= 0),
    maximum_qty         NUMERIC(18,6) CHECK (maximum_qty >= 0),
    safety_qty          NUMERIC(18,6) CHECK (safety_qty >= 0),
    source_type         VARCHAR(14)   NOT NULL CHECK (source_type IN ('COMPRA', 'TRANSFERENCIA', 'PRODUCAO')),
    source_warehouse_id BIGINT        REFERENCES warehouse(id),
    is_active           BOOLEAN       NOT NULL DEFAULT TRUE,
    created_by          UUID          NOT NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, item_code, warehouse_id),
    CONSTRAINT ck_politica_origem CHECK (
        (source_type = 'TRANSFERENCIA') = (source_warehouse_id IS NOT NULL)
        AND source_warehouse_id IS DISTINCT FROM warehouse_id),
    CONSTRAINT ck_politica_faixa CHECK (
        minimum_qty IS NULL OR maximum_qty IS NULL OR maximum_qty = 0 OR maximum_qty >= minimum_qty)
);

ALTER TABLE material_requisitions
    ADD COLUMN kind VARCHAR(14) NOT NULL DEFAULT 'CONSUMO' CHECK (kind IN ('CONSUMO', 'TRANSFERENCIA')),
    ADD COLUMN destination_warehouse_id BIGINT REFERENCES warehouse(id),
    ADD CONSTRAINT ck_requisicao_destino CHECK (
        (kind = 'TRANSFERENCIA') = (destination_warehouse_id IS NOT NULL)
        AND destination_warehouse_id IS DISTINCT FROM warehouse_id);

ALTER TABLE material_requisitions DROP CONSTRAINT IF EXISTS material_requisitions_status_check;
ALTER TABLE material_requisitions ADD CONSTRAINT material_requisitions_status_check
    CHECK (status IN ('PENDENTE_APROVACAO', 'ABERTA', 'PARCIAL', 'ATENDIDA', 'CANCELADA', 'REPROVADA'));

ALTER TABLE aprovacao_politicas DROP CONSTRAINT IF EXISTS aprovacao_politicas_tipo_check;
ALTER TABLE aprovacao_politicas ADD CONSTRAINT aprovacao_politicas_tipo_check
    CHECK (tipo IN ('LIBERACAO_CREDITO', 'ORCAMENTO_POLITICA_COMERCIAL', 'PEDIDO_POLITICA_COMERCIAL',
                    'AJUSTE_INVENTARIO', 'REQUISICAO_MATERIAL'));

CREATE INDEX ix_stock_reservations_documento
    ON stock_reservations (enterprise_id, reference_type, reference_item_code) WHERE status = 'ACTIVE';

SELECT instalar_conferencia_de_empresa();
