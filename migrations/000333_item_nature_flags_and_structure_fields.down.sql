DROP INDEX IF EXISTS idx_item_structure_history_structure;
DROP INDEX IF EXISTS idx_item_structure_history_parent;
DROP TABLE IF EXISTS item_structure_history;

ALTER TABLE item_structures
    DROP CONSTRAINT IF EXISTS ck_item_structures_setup_loss,
    DROP CONSTRAINT IF EXISTS ck_item_structures_cost_loss_type;
ALTER TABLE item_structures
    DROP COLUMN IF EXISTS generates_inspection,
    DROP COLUMN IF EXISTS is_critical_mps,
    DROP COLUMN IF EXISTS cost_center_code,
    DROP COLUMN IF EXISTS cost_loss,
    DROP COLUMN IF EXISTS cost_loss_type,
    DROP COLUMN IF EXISTS setup_loss,
    DROP COLUMN IF EXISTS line_warehouse_code,
    DROP COLUMN IF EXISTS warehouse_code;

DROP TRIGGER IF EXISTS trg_notification_item_configured ON items;
CREATE TRIGGER trg_notification_item_configured
    AFTER INSERT OR UPDATE OF nature, enterprise_id ON items
    FOR EACH ROW EXECUTE FUNCTION notification_item_configured_trigger();

ALTER TABLE items
    DROP COLUMN IF EXISTS is_process_item,
    DROP COLUMN IF EXISTS is_tool,
    DROP COLUMN IF EXISTS is_prototype,
    DROP COLUMN IF EXISTS is_configured,
    DROP COLUMN IF EXISTS is_base;
