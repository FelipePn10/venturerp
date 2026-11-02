-- Liga a ordem planejada à linha do pedido de venda que ela atende.
--
-- A sugestão do MRP só sabe do pedido pelo pegging, e o pegging é refeito a cada
-- cálculo. Firmar a sugestão não levava o pedido adiante: a OF gerada pelo MRP não
-- chegava ao cliente, e o relatório de horas de beneficiamento a mostrava sem
-- cliente e sem pedido. Mesmo desenho de independent_demand_orders (migração 449).
CREATE TABLE sales_order_planned_orders (
    id                    BIGSERIAL     PRIMARY KEY,
    enterprise_id         BIGINT        NOT NULL REFERENCES enterprise(id),
    sales_order_item_code BIGINT        NOT NULL REFERENCES sales_order_items(code) ON DELETE CASCADE,
    planned_order_id      BIGINT        NOT NULL REFERENCES planned_orders(id) ON DELETE CASCADE,
    quantity              NUMERIC(18,6) NOT NULL CHECK (quantity > 0),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by            UUID          NOT NULL,
    UNIQUE (enterprise_id, sales_order_item_code, planned_order_id)
);
CREATE INDEX idx_sales_order_planned_orders_planned ON sales_order_planned_orders (planned_order_id);

SELECT instalar_conferencia_de_empresa();
