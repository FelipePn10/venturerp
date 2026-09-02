DO $$
DECLARE table_to_audit TEXT;
BEGIN
    FOREACH table_to_audit IN ARRAY ARRAY[
        'maintenance_plans','maintenance_orders','machine_downtimes',
        'planning_params','mrp_calculation_logs','configured_item_rules',
        'shipment_loads','shipment_load_shipments','sales_forecasts',
        'mrp_item_profiles','mrp_planned_suggestions','capacity_requirements','production_sequences'
    ] LOOP
        IF to_regclass('public.'||table_to_audit) IS NOT NULL THEN
            EXECUTE format('DROP TRIGGER IF EXISTS trg_operational_audit ON public.%I',table_to_audit);
        END IF;
    END LOOP;
END $$;
DROP TRIGGER IF EXISTS trg_operational_mutation_audit_immutable ON operational_mutation_audit;
DROP TRIGGER IF EXISTS trg_audit_log_immutable ON audit_log;
DROP FUNCTION IF EXISTS protect_operational_mutation_audit();
DROP FUNCTION IF EXISTS record_operational_mutation();
DROP TABLE IF EXISTS operational_mutation_audit;
DROP TABLE IF EXISTS http_idempotency_records;
