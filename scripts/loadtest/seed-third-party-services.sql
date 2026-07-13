\set ON_ERROR_STOP on

-- Dataset local e idempotente para carga de serviços de terceiros.
-- Pré-requisito: loadtest.thirdparty@panossoerp.test criado pela API e ligado
-- a uma empresa. As faixas 930000000–989999999 são reservadas ao teste.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM users WHERE email = 'loadtest.thirdparty@panossoerp.test') THEN
        RAISE EXCEPTION 'crie loadtest.thirdparty@panossoerp.test pela API antes de executar o seed';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM enterprise) THEN
        RAISE EXCEPTION 'o banco de teste precisa possuir uma empresa';
    END IF;
END $$;

INSERT INTO suppliers (
    code, name, document_type, document_number, homologated, created_by
)
SELECT
    940000000 + n,
    'Fornecedor carga industrial ' || n,
    'ESTRANGEIRO',
    'LOADSUP' || lpad(n::text, 6, '0'),
    true,
    load_user.id
FROM generate_series(1, 100) AS series(n)
CROSS JOIN LATERAL (
    SELECT id FROM users WHERE email = 'loadtest.thirdparty@panossoerp.test'
) AS load_user
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    homologated = true,
    updated_at = NOW();

INSERT INTO items (
    code, warehouse_code, created_by, pdm_description_technique,
    warehouse_unit_of_measurement, accepts_fractional_quantity
)
SELECT
    930000000 + n,
    930000000 + n,
    load_user.id,
    'Componente industrial para carga ' || n,
    'UN',
    (n % 5) <> 0
FROM generate_series(1, 1000) AS series(n)
CROSS JOIN LATERAL (
    SELECT id FROM users WHERE email = 'loadtest.thirdparty@panossoerp.test'
) AS load_user
ON CONFLICT (code) DO UPDATE SET
    pdm_description_technique = EXCLUDED.pdm_description_technique,
    accepts_fractional_quantity = EXCLUDED.accepts_fractional_quantity;

INSERT INTO operations (
    code, name, description, origin, supplier_id, service_item_code,
    cost_per_unit, lead_time_days, third_party_remittance, created_by
)
SELECT
    950000001,
    'Tratamento externo - carga',
    'Operação dedicada ao ensaio de carga de serviços de terceiros',
    'EXTERNA',
    supplier.id,
    930000001,
    12.50,
    5,
    'DEMAND_ITEMS',
    load_user.id
FROM suppliers supplier
CROSS JOIN LATERAL (
    SELECT id FROM users WHERE email = 'loadtest.thirdparty@panossoerp.test'
) AS load_user
WHERE supplier.code = 940000001
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    origin = 'EXTERNA',
    supplier_id = EXCLUDED.supplier_id,
    service_item_code = EXCLUDED.service_item_code,
    lead_time_days = EXCLUDED.lead_time_days,
    third_party_remittance = EXCLUDED.third_party_remittance,
    updated_at = NOW();

INSERT INTO manufacturing_routes (
    code, item_code, alternative, description, situation, is_standard,
    is_active, created_by
)
SELECT
    960000000 + n,
    930000000 + n,
    1,
    'Roteiro carga serviço externo ' || n,
    'APROVADA',
    true,
    true,
    load_user.id
FROM generate_series(1, 1000) AS series(n)
CROSS JOIN LATERAL (
    SELECT id FROM users WHERE email = 'loadtest.thirdparty@panossoerp.test'
) AS load_user
ON CONFLICT (code) DO UPDATE SET
    description = EXCLUDED.description,
    situation = 'APROVADA',
    is_standard = true,
    is_active = true,
    updated_at = NOW();

INSERT INTO route_operations (
    route_id, sequence, operation_id, supplier_id, service_item_code,
    cost_per_unit, lead_time_days, third_party_remittance, is_active
)
SELECT
    route.id,
    10,
    operation.id,
    supplier.id,
    930000001,
    12.50,
    5,
    'DEMAND_ITEMS',
    true
FROM manufacturing_routes route
JOIN operations operation ON operation.code = 950000001
JOIN suppliers supplier ON supplier.code = 940000001
WHERE route.code BETWEEN 960000001 AND 960001000
ON CONFLICT (route_id, sequence) DO UPDATE SET
    operation_id = EXCLUDED.operation_id,
    supplier_id = EXCLUDED.supplier_id,
    service_item_code = EXCLUDED.service_item_code,
    cost_per_unit = EXCLUDED.cost_per_unit,
    lead_time_days = EXCLUDED.lead_time_days,
    third_party_remittance = EXCLUDED.third_party_remittance,
    is_active = true,
    updated_at = NOW();

INSERT INTO production_orders (
    order_number, item_code, mask, planned_qty, status, start_date, end_date,
    route_id, enterprise_id, created_by
)
SELECT
    970000000 + n,
    930000000 + ((n - 1) % 1000) + 1,
    CASE WHEN n % 4 = 0 THEN 'CONFIG-' || (n % 20) ELSE '' END,
    100 + (n % 900),
    'RELEASED',
    CURRENT_DATE - (n % 45),
    CURRENT_DATE + (n % 30),
    route.id,
    enterprise.id,
    load_user.id
FROM generate_series(1, 20000) AS series(n)
JOIN manufacturing_routes route
  ON route.code = 960000000 + ((n - 1) % 1000) + 1
CROSS JOIN LATERAL (SELECT id FROM enterprise ORDER BY id LIMIT 1) AS enterprise
CROSS JOIN LATERAL (
    SELECT id FROM users WHERE email = 'loadtest.thirdparty@panossoerp.test'
) AS load_user
ON CONFLICT (enterprise_id, order_number) DO UPDATE SET
    planned_qty = EXCLUDED.planned_qty,
    status = EXCLUDED.status,
    start_date = EXCLUDED.start_date,
    end_date = EXCLUDED.end_date,
    route_id = EXCLUDED.route_id,
    updated_at = NOW();

INSERT INTO third_party_service_orders (
    code, enterprise_id, production_order_id, route_operation_id, operation_id,
    item_code, mask, supplier_code, service_item_code, uom, quantity,
    fulfilled_quantity, start_date, due_date, status, remittance_type, kanban,
    notes, created_by
)
SELECT
    980000000 + n,
    production_order.enterprise_id,
    production_order.id,
    route_operation.id,
    route_operation.operation_id,
    production_order.item_code,
    production_order.mask,
    940000000 + ((n - 1) % 100) + 1,
    930000001,
    'UN',
    production_order.planned_qty,
    CASE WHEN n % 10 = 0 THEN production_order.planned_qty ELSE 0 END,
    COALESCE(production_order.start_date, CURRENT_DATE),
    COALESCE(production_order.end_date, CURRENT_DATE + 5),
    CASE
        WHEN n % 10 = 0 THEN 'COMPLETED'
        WHEN n % 3 = 0 THEN 'RELEASED_WITH_PO'
        WHEN n % 3 = 1 THEN 'RELEASED_WITHOUT_PO'
        ELSE 'FIRM'
    END,
    CASE WHEN n % 7 = 0 THEN 'ORDER_ITEM' ELSE 'DEMAND_ITEMS' END,
    n % 5 = 0,
    'Dataset representativo de carga',
    load_user.id
FROM generate_series(1, 20000) AS series(n)
JOIN production_orders production_order
  ON production_order.order_number = 970000000 + n
JOIN manufacturing_routes route ON route.id = production_order.route_id
JOIN route_operations route_operation ON route_operation.route_id = route.id
CROSS JOIN LATERAL (
    SELECT id FROM users WHERE email = 'loadtest.thirdparty@panossoerp.test'
) AS load_user
ON CONFLICT (enterprise_id, code) DO UPDATE SET
    supplier_code = EXCLUDED.supplier_code,
    quantity = EXCLUDED.quantity,
    fulfilled_quantity = EXCLUDED.fulfilled_quantity,
    due_date = EXCLUDED.due_date,
    status = EXCLUDED.status,
    kanban = EXCLUDED.kanban,
    updated_at = NOW();

INSERT INTO third_party_service_prices (
    enterprise_id, item_code, mask, supplier_code, operation_id, uom,
    reference_date, preferred, unit_price, conversion_factor, freight_type,
    freight_value, tax_percent, is_active, created_by
)
SELECT
    enterprise.id,
    930000000 + ((n - 1) % 1000) + 1,
    CASE WHEN n % 4 = 0 THEN 'CONFIG-' || (n % 20) ELSE '' END,
    940000000 + (((n - 1) / 1000) % 10) + 1,
    operation.id,
    'UN',
    CURRENT_DATE - ((((n - 1) / 10000) % 5) * 90),
    (((n - 1) / 1000) % 10) = 0 AND (((n - 1) / 10000) % 5) = 0,
    10 + (n % 500)::numeric / 10,
    1,
    CASE WHEN n % 3 = 0 THEN 'PERCENT' ELSE 'FIXED' END,
    1 + (n % 25)::numeric / 10,
    n % 12,
    true,
    load_user.id
FROM generate_series(1, 50000) AS series(n)
CROSS JOIN LATERAL (SELECT id FROM enterprise ORDER BY id LIMIT 1) AS enterprise
JOIN operations operation ON operation.code = 950000001
CROSS JOIN LATERAL (
    SELECT id FROM users WHERE email = 'loadtest.thirdparty@panossoerp.test'
) AS load_user
ON CONFLICT (
    enterprise_id, item_code, mask, supplier_code, operation_id, reference_date
) DO UPDATE SET
    unit_price = EXCLUDED.unit_price,
    conversion_factor = EXCLUDED.conversion_factor,
    freight_type = EXCLUDED.freight_type,
    freight_value = EXCLUDED.freight_value,
    tax_percent = EXCLUDED.tax_percent,
    is_active = true,
    updated_at = NOW();

INSERT INTO global_unit_conversions (
    enterprise_id, from_uom, to_uom, factor, is_active, created_by
)
SELECT
    enterprise.id,
    'L' || lpad(n::text, 5, '0'),
    'U' || lpad(n::text, 5, '0'),
    1 + n::numeric / 1000,
    true,
    load_user.id
FROM generate_series(1, 1000) AS series(n)
CROSS JOIN LATERAL (SELECT id FROM enterprise ORDER BY id LIMIT 1) AS enterprise
CROSS JOIN LATERAL (
    SELECT id FROM users WHERE email = 'loadtest.thirdparty@panossoerp.test'
) AS load_user
ON CONFLICT (enterprise_id, from_uom, to_uom) DO UPDATE SET
    factor = EXCLUDED.factor,
    is_active = true,
    updated_at = NOW();

ANALYZE items;
ANALYZE suppliers;
ANALYZE third_party_service_prices;
ANALYZE third_party_service_orders;
ANALYZE global_unit_conversions;

SELECT 'items' AS entity, count(*) AS rows
FROM items WHERE code BETWEEN 930000001 AND 930001000
UNION ALL
SELECT 'suppliers', count(*) FROM suppliers WHERE code BETWEEN 940000001 AND 940000100
UNION ALL
SELECT 'prices', count(*) FROM third_party_service_prices
WHERE item_code BETWEEN 930000001 AND 930001000
UNION ALL
SELECT 'orders', count(*) FROM third_party_service_orders
WHERE code BETWEEN 980000001 AND 980020000
UNION ALL
SELECT 'conversions', count(*) FROM global_unit_conversions
WHERE from_uom BETWEEN 'L00001' AND 'L01000';
