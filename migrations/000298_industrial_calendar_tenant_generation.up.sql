BEGIN;

ALTER TABLE industrial_calendar
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT,
    ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'MANUAL';

UPDATE industrial_calendar
SET enterprise_id = (SELECT id FROM enterprise ORDER BY id LIMIT 1)
WHERE enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM enterprise) = 1;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM industrial_calendar WHERE enterprise_id IS NULL) THEN
        RAISE EXCEPTION 'Calendario legado ambiguo: associe os dias a uma empresa antes da migracao';
    END IF;
END $$;

ALTER TABLE industrial_calendar DROP CONSTRAINT IF EXISTS industrial_calendar_year_month_day_key;
ALTER TABLE industrial_calendar
    ALTER COLUMN enterprise_id SET NOT NULL,
    ADD CONSTRAINT fk_industrial_calendar_enterprise FOREIGN KEY (enterprise_id) REFERENCES enterprise(id),
    ADD CONSTRAINT chk_industrial_calendar_source CHECK (source IN ('AUTOMATICO', 'FIM_DE_SEMANA', 'FERIADO', 'MANUAL')),
    ADD CONSTRAINT uq_industrial_calendar_enterprise_date UNIQUE (enterprise_id, year, month, day);

DROP INDEX IF EXISTS idx_industrial_calendar_year_month;
DROP INDEX IF EXISTS idx_industrial_calendar_workday;
CREATE INDEX idx_industrial_calendar_tenant_month ON industrial_calendar(enterprise_id, year, month, day);
CREATE INDEX idx_industrial_calendar_tenant_workday ON industrial_calendar(enterprise_id, is_workday, year, month, day);

CREATE OR REPLACE FUNCTION subtract_workdays(base_date DATE, days_to_sub INT, tenant_id BIGINT)
RETURNS DATE LANGUAGE plpgsql AS $$
DECLARE result DATE := base_date; remaining INT := days_to_sub;
BEGIN
    IF days_to_sub <= 0 THEN RETURN base_date; END IF;
    WHILE remaining > 0 LOOP
        result := result - 1;
        IF EXISTS (SELECT 1 FROM industrial_calendar WHERE enterprise_id=tenant_id AND year=EXTRACT(YEAR FROM result)::INT AND month=EXTRACT(MONTH FROM result)::INT AND day=EXTRACT(DAY FROM result)::INT AND is_workday) THEN
            remaining := remaining - 1;
        ELSIF NOT EXISTS (SELECT 1 FROM industrial_calendar WHERE enterprise_id=tenant_id AND year=EXTRACT(YEAR FROM result)::INT AND month=EXTRACT(MONTH FROM result)::INT AND day=EXTRACT(DAY FROM result)::INT) AND EXTRACT(DOW FROM result) NOT IN (0,6) THEN
            remaining := remaining - 1;
        END IF;
    END LOOP;
    RETURN result;
END; $$;

COMMIT;
