-- A marca de baixa automática ligada nos itens não volta (não se sabe quais
-- estavam desligadas de propósito); só o padrão da coluna volta.
ALTER TABLE items ALTER COLUMN warehouse_automatic_low SET DEFAULT FALSE;

DROP TRIGGER IF EXISTS trg_origem_pela_demanda ON independent_demand_orders;
DROP TRIGGER IF EXISTS trg_origem_pelo_pedido ON sales_order_planned_orders;
DROP TRIGGER IF EXISTS trg_origem_pela_cadeia ON production_chain_links;
DROP FUNCTION IF EXISTS trg_origem_ligada_a_planejada();
DROP TRIGGER IF EXISTS trg_of_passar_origem_as_filhas ON production_orders;
DROP FUNCTION IF EXISTS trg_of_passar_origem_as_filhas();
DROP TRIGGER IF EXISTS trg_of_herdar_origem ON production_orders;
DROP FUNCTION IF EXISTS trg_of_herdar_origem();
DROP FUNCTION IF EXISTS erp_of_herdar_origem(BIGINT);

ALTER TABLE production_orders
    DROP CONSTRAINT IF EXISTS production_orders_projeto_fk,
    DROP CONSTRAINT IF EXISTS production_orders_cliente_fk,
    DROP COLUMN IF EXISTS project_id,
    DROP COLUMN IF EXISTS customer_code;

DROP FUNCTION IF EXISTS erp_projeto_do_codigo(BIGINT, TEXT, TEXT, BIGINT);
DROP TABLE IF EXISTS projects;

SELECT instalar_conferencia_de_empresa();
