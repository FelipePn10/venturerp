CREATE UNIQUE INDEX IF NOT EXISTS uq_planned_order_number_tenant_active
    ON planned_orders (enterprise_id, order_number)
    WHERE enterprise_id IS NOT NULL AND is_active = TRUE;
