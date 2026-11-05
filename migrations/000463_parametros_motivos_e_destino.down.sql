ALTER TABLE mrp_planned_suggestions DROP COLUMN priority;
ALTER TABLE stock_movements DROP COLUMN wave_line_id;
ALTER TABLE kanban_cards DROP COLUMN warehouse_id;
ALTER TABLE stock_picking_wave_lines DROP COLUMN invoiced_qty;
ALTER TABLE purchase_requisition_items DROP COLUMN warehouse_id;
DROP TABLE stock_discrepancy_reasons;
ALTER TABLE stock_parameters
    DROP COLUMN retroactive_months, DROP COLUMN tolerance_a, DROP COLUMN tolerance_b, DROP COLUMN tolerance_c,
    DROP COLUMN obsolete_days, DROP COLUMN obsolete_provision_pct;
