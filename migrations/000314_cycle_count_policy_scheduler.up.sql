ALTER TABLE items
    ADD COLUMN cyclical_count_policy_activated_at TIMESTAMPTZ;

UPDATE items
SET cyclical_count_policy_activated_at = created_at
WHERE warehouse_cyclical_count_config IS NOT NULL
  AND COALESCE(
        NULLIF(warehouse_cyclical_count_config->>'days', '')::int,
        NULLIF(warehouse_cyclical_count_config->>'days_interval', '')::int,
        NULLIF(warehouse_cyclical_count_config->>'DaysInterval', '')::int,
        0
      ) > 0;

CREATE FUNCTION set_item_cycle_count_policy_activation() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT'
       OR NEW.warehouse_cyclical_count_config IS DISTINCT FROM OLD.warehouse_cyclical_count_config
       OR NEW.warehouse_code IS DISTINCT FROM OLD.warehouse_code THEN
        IF NEW.warehouse_cyclical_count_config IS NULL OR COALESCE(
            NULLIF(NEW.warehouse_cyclical_count_config->>'days', '')::int,
            NULLIF(NEW.warehouse_cyclical_count_config->>'days_interval', '')::int,
            NULLIF(NEW.warehouse_cyclical_count_config->>'DaysInterval', '')::int,
            0
        ) <= 0 THEN
            NEW.cyclical_count_policy_activated_at := NULL;
        ELSE
            NEW.cyclical_count_policy_activated_at := NOW();
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_item_cycle_count_policy_activation
BEFORE INSERT OR UPDATE OF warehouse_cyclical_count_config, warehouse_code ON items
FOR EACH ROW EXECUTE FUNCTION set_item_cycle_count_policy_activation();

ALTER TABLE stock_cycle_counts
    ADD COLUMN origin VARCHAR(20) NOT NULL DEFAULT 'MANUAL'
        CHECK (origin IN ('MANUAL', 'POLITICA_ITEM')),
    ADD COLUMN policy_days INTEGER CHECK (policy_days IS NULL OR policy_days > 0);

WITH duplicates AS (
    SELECT id,enterprise_id,state,ROW_NUMBER() OVER (
        PARTITION BY enterprise_id,warehouse_id,item_code,mask,lot_code
        ORDER BY created_at,id
    ) position
    FROM stock_cycle_counts
    WHERE warehouse_address_id IS NULL
      AND state IN ('PROGRAMADA', 'EM_CONTAGEM', 'DIVERGENTE', 'CONCLUIDA')
), cancelled AS (
    UPDATE stock_cycle_counts c SET state='CANCELADA',updated_at=NOW()
    FROM duplicates d WHERE d.id=c.id AND d.position>1
    RETURNING c.enterprise_id,c.id,d.state AS previous_state
)
INSERT INTO stock_cycle_count_audit(enterprise_id,cycle_count_id,action,previous_state,new_state,details)
SELECT enterprise_id,id,'MIGRACAO_DEDUPLICACAO',previous_state,'CANCELADA',jsonb_build_object('motivo','Mais de uma contagem aberta para o mesmo escopo')
FROM cancelled;

CREATE UNIQUE INDEX uq_stock_cycle_count_open_scope
ON stock_cycle_counts (enterprise_id, warehouse_id, item_code, mask, lot_code)
WHERE warehouse_address_id IS NULL
  AND state IN ('PROGRAMADA', 'EM_CONTAGEM', 'DIVERGENTE', 'CONCLUIDA');
