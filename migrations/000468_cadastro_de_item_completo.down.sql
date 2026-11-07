DELETE FROM item_historico WHERE evento = 'USO_COM_AVISO';
ALTER TABLE item_historico DROP CONSTRAINT item_historico_evento_check;
ALTER TABLE item_historico ADD CONSTRAINT item_historico_evento_check CHECK (evento IN
    ('CRIACAO', 'ALTERACAO', 'STATUS', 'LIBERACAO', 'ESTORNO_LIBERACAO'));

CREATE OR REPLACE FUNCTION item_conferir_utilizavel() RETURNS trigger AS $$
DECLARE
    g      item_governanca%ROWTYPE;
    codigo TEXT;
    subst  TEXT;
BEGIN
    IF NEW.item_code IS NULL THEN
        RETURN NEW;
    END IF;
    IF TG_OP = 'UPDATE' AND NEW.item_code IS NOT DISTINCT FROM OLD.item_code THEN
        RETURN NEW;
    END IF;
    SELECT * INTO g FROM item_governanca WHERE item_code = NEW.item_code;
    IF NOT FOUND OR g.status = 'LIBERADO' THEN
        RETURN NEW;
    END IF;
    SELECT business_code INTO codigo FROM items WHERE code = NEW.item_code;
    SELECT business_code INTO subst FROM items WHERE code = g.substituto_code;
    RAISE EXCEPTION USING ERRCODE = 'P0001', MESSAGE =
        CASE g.status
            WHEN 'EM_CADASTRO'   THEN format('o item %s ainda está em cadastro: as áreas precisam liberar o cadastro antes de usá-lo', codigo)
            WHEN 'BLOQUEADO'     THEN format('o item %s está bloqueado (%s)', codigo, g.motivo)
            ELSE format('o item %s foi descontinuado (%s)%s', codigo, g.motivo,
                        CASE WHEN subst IS NOT NULL THEN format(': use o item %s', subst) ELSE '' END)
        END;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION item_registrar_historico() RETURNS trigger AS $$
DECLARE
    antes  JSONB;
    depois JSONB;
    k      TEXT;
    motivo TEXT := NULLIF(current_setting('venture.item_reason', true), '');
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO item_historico(enterprise_id, item_code, evento, por, motivo)
        VALUES (NEW.enterprise_id, NEW.code, 'CRIACAO', COALESCE(item_ator(), NEW.created_by), motivo);
        RETURN NEW;
    END IF;
    antes  := to_jsonb(OLD) - ARRAY['id','abc_consumption_value','abc_share_pct','abc_calculated_at','planning_llc','cyclical_count_policy_activated_at'];
    depois := to_jsonb(NEW) - ARRAY['id','abc_consumption_value','abc_share_pct','abc_calculated_at','planning_llc','cyclical_count_policy_activated_at'];
    FOR k IN SELECT jsonb_object_keys(depois) LOOP
        IF antes -> k IS DISTINCT FROM depois -> k THEN
            INSERT INTO item_historico(enterprise_id, item_code, evento, campo, antes, depois, por, motivo)
            VALUES (NEW.enterprise_id, NEW.code, 'ALTERACAO', k, antes -> k, depois -> k, item_ator(), motivo);
        END IF;
    END LOOP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TABLE IF EXISTS item_status_politicas;
DROP TABLE IF EXISTS pdm_atributo_valores;
DROP TABLE IF EXISTS pdm_atributos;
DROP TABLE IF EXISTS pdm_grupo_modificadores;
ALTER TABLE modifier DROP COLUMN IF EXISTS abbreviation;

DROP INDEX IF EXISTS idx_items_buyer;
DROP INDEX IF EXISTS idx_items_enterprise_pdm_signature;
DROP INDEX IF EXISTS idx_items_enterprise_mpn;
DROP INDEX IF EXISTS idx_items_enterprise_previous_code;
DROP INDEX IF EXISTS uq_items_enterprise_gtin;

ALTER TABLE items
    DROP CONSTRAINT IF EXISTS chk_items_shelf_life,
    DROP CONSTRAINT IF EXISTS chk_items_un_number,
    DROP CONSTRAINT IF EXISTS chk_items_lead_times,
    DROP CONSTRAINT IF EXISTS chk_items_delivery_tolerance,
    DROP CONSTRAINT IF EXISTS chk_items_receipt_routing,
    DROP CONSTRAINT IF EXISTS chk_items_gtin,
    DROP CONSTRAINT IF EXISTS chk_items_efd_item_type,
    DROP CONSTRAINT IF EXISTS chk_items_tax_unit_factor;

ALTER TABLE items
    DROP COLUMN long_description, DROP COLUMN general_notes, DROP COLUMN revision, DROP COLUMN previous_code,
    DROP COLUMN manufacturer_name, DROP COLUMN manufacturer_part_number, DROP COLUMN warehouse_shelf_life_days,
    DROP COLUMN warehouse_min_remaining_life_days, DROP COLUMN warehouse_storage_conditions, DROP COLUMN warehouse_hazardous,
    DROP COLUMN warehouse_un_number, DROP COLUMN warehouse_hazard_class, DROP COLUMN engineering_volume,
    DROP COLUMN planning_buyer_employee_code, DROP COLUMN planning_purchase_lead_time_days,
    DROP COLUMN planning_manufacturing_lead_time_days, DROP COLUMN planning_notes,
    DROP COLUMN supplies_under_delivery_tolerance_pct, DROP COLUMN supplies_over_delivery_tolerance_pct,
    DROP COLUMN supplies_receipt_routing, DROP COLUMN supplies_classification_code, DROP COLUMN accounting_gtin,
    DROP COLUMN accounting_gtin_tax, DROP COLUMN accounting_efd_item_type, DROP COLUMN accounting_tax_unit,
    DROP COLUMN accounting_tax_unit_factor, DROP COLUMN pdm_signature;

SELECT instalar_conferencia_de_empresa();
